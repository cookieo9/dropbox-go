package dropbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Time is a special structure which represents a time value (using an embedded time.Time)
// and supports conversion from JSON with the appropriate format used by the
// Dropbox API.
type Time struct {
	time.Time
}

// UnmarshalJSON exists to implement the json.Unmarshaller interface
func (t *Time) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	t2, err := time.Parse(time.RFC1123Z, str)
	if err != nil {
		return err
	}
	t.Time = t2
	return nil
}

// MarshalJSON exists to implement the json.Marshaller interface
func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(time.RFC1123Z))
}

// Metadata represents all possible metadata JSON responses from the
// server.
type Metadata struct {
	Size        string     `json:"size"`
	Hash        string     `json:"hash"`
	Rev         string     `json:"rev"`
	ThumbExists bool       `json:"thumb_exists"`
	Bytes       int64      `json:"bytes"`
	Modified    Time       `json:"modified"`
	ClientMTime Time       `json:"client_mtime"`
	Path        string     `json:"path"`
	IsDir       bool       `json:"is_dir"`
	Icon        string     `json:"icon"`
	Root        string     `json:"root"`
	MimeType    string     `json:"mime_type"`
	Revision    uint64     `json:"revision"`
	Contents    []Metadata `json:"contents"`
}

// An Entry is a [path, metadata] pair that represents a file change in
// the dropbox.
type Entry struct {
	Path string    `json:"path"`
	Meta *Metadata `json:"meta"`
}

// UnmarshalJSON converts a delta's 'entry' into this Entry value.
// This is necessary since the JSON code for an entry is an array
// with different types in its two fields (ie: a tuple), and some
// work needs to be done to convert it into a struct, since Go doesn't
// support either arrays with different types in the fields or tuples.
func (e *Entry) UnmarshalJSON(data []byte) error {
	els := []interface{}{&e.Path, &e.Meta}
	return json.Unmarshal(data, &els)
}

// MarshalJSON exists to implement the json.Marshaller interface properly
func (e *Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{e.Path, e.Meta})
}

// A Delta represents a list of changes to a dropbox. The cursor field
// is used to resume processing all changes since this delta.
type Delta struct {
	Entries []Entry `json:"entries"`
	Reset   bool    `json:"reset"`
	Cursor  string  `json:"cursor"`
	HasMore bool    `json:"has_more"`
}

// A Share represents a resource in the user's dropbox which can be accessed
// through an external URL with no authentication needed. Perfect for embedding
// into an email, sending as a link, or downloading without involving your own
// app.
type Share struct {
	URL     string `json:"url"`
	Expires Time   `json:"expires"`
}

// A CopyRef represents a file in a user's dropbox which can be atomically copied
// to another user's dropbox without downloading/uploading it.
type CopyRef struct {
	CopyRef string `json:"copy_ref"`
	Expires Time   `json:"expires"`
}

// An AccountInfo represents the user's account information in dropbox.
type AccountInfo struct {
	ReferralLink string    `json:"referral_link"`
	DisplayName  string    `json:"display_name"`
	UID          uint64    `json:"uid"`
	Country      string    `json:"country"`
	QuotaInfo    QuotaInfo `json:"quota_info"`
}

// A QuotaInfo represents the data usage in a dropbox.
type QuotaInfo struct {
	Shared int64 `json:"shared"`
	Quota  int64 `json:"quota"`
	Normal int64 `json:"normal"`
}

// A ChunkedUpload represents the state of a chunked upload session.
type ChunkedUpload struct {
	UploadId string `json:"upload_id"`
	Offset   int64  `json:"offset"`
	Expires  Time   `json:"expires"`
}

// An APIError contains an error returned by the API.
type APIError struct {
	Code    int
	Message string `json:"error"`
}

func (ae *APIError) Error() string {
	return fmt.Sprintf("Dropbox API Error(%d): %s", ae.Code, ae.Message)
}

func (c *Client) filePath(p string) string {
	return path.Clean(path.Join("/", string(c.root), p))
}

func checkResponse(response *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusUnauthorized:
		context := response.Request.URL.Path[3:]
		drainAndClose(response.Body)
		return nil, &AuthorizationError{context, errors.New("bad or expired token")}
	}
	return response, err
}

func (c *Client) put(urlStr string, params url.Values, body io.Reader, contentLength int64) (*http.Response, error) {
	if err := c.signParam("PUT", urlStr, params); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", urlStr+"?"+params.Encode(), body)
	if err != nil {
		return nil, err
	}

	if contentLength > 0 {
		req.ContentLength = contentLength
	}

	return checkResponse(c.client().Do(req))
}

func (c *Client) postForm(urlStr string, params url.Values) (*http.Response, error) {
	if err := c.signParam("POST", urlStr, params); err != nil {
		return nil, err
	}

	return checkResponse(c.client().PostForm(urlStr, params))
}

func (c *Client) get(urlStr string, params url.Values) (*http.Response, error) {
	if err := c.signParam("GET", urlStr, params); err != nil {
		return nil, err
	}

	return checkResponse(c.client().Get(urlStr + "?" + params.Encode()))
}

func drain(r io.Reader) error {
	_, err := io.Copy(ioutil.Discard, r)
	if err == io.EOF {
		return nil
	}
	return err
}

func drainAndClose(rc io.ReadCloser) error {
	err1 := drain(rc)
	err2 := rc.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func parseJSON(resp *http.Response, target interface{}) error {
	d := json.NewDecoder(resp.Body)

	if resp.StatusCode == http.StatusOK {
		if err := d.Decode(target); err != nil {
			return err
		}
		return nil
	}

	apierr := &APIError{
		Code: resp.StatusCode,
	}

	if err := d.Decode(&apierr); err != nil {
		if err != io.EOF {
			return err
		}
	}

	return apierr
}

func (c *Client) putJSON(urlStr string, params url.Values, target interface{}, data io.Reader, size int64) error {
	r, err := c.put(urlStr, params, data, size)
	if err != nil {
		return err
	}
	defer drainAndClose(r.Body)
	return parseJSON(r, target)
}

func (c *Client) getJSON(urlStr string, params url.Values, target interface{}) error {
	r, err := c.get(urlStr, params)
	if err != nil {
		return err
	}
	defer drainAndClose(r.Body)
	return parseJSON(r, target)
}

func (c *Client) postFormJSON(urlStr string, params url.Values, target interface{}) error {
	r, err := c.postForm(urlStr, params)
	if err != nil {
		return err
	}
	defer drainAndClose(r.Body)
	return parseJSON(r, target)
}

func (c *Client) fileAccess(uri string, params url.Values) (io.ReadCloser, *Metadata, error) {
	response, err := c.get(uri, params)
	if err != nil {
		return nil, nil, err
	}
	var meta *Metadata

	metaStr := response.Header.Get("x-dropbox-metadata")
	if len(metaStr) > 0 {
		if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
			drainAndClose(response.Body)
			return nil, nil, err
		}
	}

	return response.Body, meta, nil
}
