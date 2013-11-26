package dropbox

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

// The AccessRoot type represents the enumeration of Dropbox Root options either "sandbox", where the
// user provides access only to a app-specific folder, and "dropbox" where their entire dropbox is
// available.
type AccessRoot string

// Constants to control Client access to dropbox
const (
	DropboxRoot   AccessRoot = "dropbox" // Access full dropbox
	SandboxRoot   AccessRoot = "sandbox" // Access app sandbox
	AppFolderRoot AccessRoot = "sandbox" // Alias for "sandbox"
)

// A Client provides access to the Dropbox services.
type Client struct {
	*Session
	root AccessRoot
}

// URLs for all the Dropbox REST-API Calls
const (
	AccountInfoURL         = APIPrefix + "/account/info"
	FilesURL               = ContentPrefix + "/files"
	FilesPutURL            = ContentPrefix + "/files_put"
	MetadataURL            = APIPrefix + "/metadata"
	DeltaURL               = APIPrefix + "/delta"
	RevisionsURL           = APIPrefix + "/revisions"
	RestoreURL             = APIPrefix + "/restore"
	SearchURL              = APIPrefix + "/search"
	SharesURL              = APIPrefix + "/shares"
	MediaURL               = APIPrefix + "/media"
	CopyRefURL             = APIPrefix + "/copy_ref"
	ThumbnailsURL          = ContentPrefix + "/thumbnails"
	ChunkedUploadURL       = ContentPrefix + "/chunked_upload"
	CommitChunkedUploadURL = ContentPrefix + "/commit_chunked_upload"
	FileOpsCopyURL         = APIPrefix + "/fileops/copy"
	FileOpsCreateFolderURL = APIPrefix + "/fileops/create_folder"
	FileOpsDeleteURL       = APIPrefix + "/fileops/delete"
	FileOpsMoveURL         = APIPrefix + "/fileops/move"
)

// NewClient creates a new client using the given authorized session, and working on
// the given Dropbox Root (defined by the App entry). Passing an unauthorized
// session will cause NewClient to panic.
func NewClient(session *Session, root AccessRoot) *Client {
	if !session.Authorized() {
		panic("Session Not Authorized!")
	}

	return &Client{
		Session: session,
		root:    root,
	}
}

// AccountInfo performs the account/info API call and returns the result.
func (c *Client) AccountInfo() (account *AccountInfo, err error) {
	err = c.getJSON(AccountInfoURL, c.makeParams(true), &account)
	return
}

// GetFile downloads the data for a single file as a io.ReadCloser, as well as fetches
// its metadata.
func (c *Client) GetFile(path string, rev string) (io.ReadCloser, *Metadata, error) {
	uri := FilesURL + c.filePath(path)
	params := c.makeParams(false)
	if rev != "" {
		params.Set("rev", rev)
	}
	return c.fileAccess(uri, params)
}

// Thumbnail downloads a thumbnail image for the given path. If either format or size
// are not the empty string they will be sent as part of the request.
func (c *Client) Thumbnail(path, format, size string) (io.ReadCloser, *Metadata, error) {
	uri := ThumbnailsURL + c.filePath(path)
	params := c.makeParams(false)
	if format != "" {
		params.Set("format", format)
	}
	if size != "" {
		params.Set("size", size)
	}
	return c.fileAccess(uri, params)
}

// PutFile uploads size bytes from the given io.Reader as the contents of a file at the
// given url. If parentRev is not the empty string, it is set as part of the request.
func (c Client) PutFile(path string, overwrite bool, parentRev string, data io.Reader, size int64) (meta *Metadata, err error) {
	uri := FilesPutURL + c.filePath(path)
	params := c.makeParams(true)
	if overwrite {
		params.Set("overwrite", "true")
	}
	if parentRev != "" {
		params.Set("parent_rev", parentRev)
	}
	err = c.putJSON(uri, params, &meta, data, size)
	return
}

// Metadata returns the metadata for a file or folder at a given path.
//
//	file_limit: if > 0, return an error if more than that many files exist in the directory. If <= 0 use default limit (10,000).
//	hash: if set, compare hash to hash in metadata, if same, return (nil, true, error (304 Not Modified))
//	list: list contents of directories
//	deleted: show deleted files in listings
//	rev: if set, use given revision of file instead of latest
func (c *Client) Metadata(path string, fileLimit int, hash string, list, deleted bool, rev string) (meta *Metadata, unmodified bool, err error) {
	params := c.makeParams(true)
	if fileLimit > 0 {
		params.Set("file_limit", strconv.FormatInt(int64(fileLimit), 10))
	}
	if hash != "" {
		params.Set("hash", hash)
	}
	if !list {
		params.Set("list", "false")
	}
	if deleted {
		params.Set("include_deleted", "true")
	}
	if rev != "" {
		params.Set("rev", rev)
	}

	err = c.getJSON(MetadataURL+c.filePath(path), params, &meta)
	if apierr, ok := err.(*APIError); ok && apierr.Code == http.StatusNotModified {
		unmodified = true
		err = nil
	}

	return
}

// Search searches the given path for files matching the query string.
//
//	fileLimit: if > 0, return at most this many results, instead of the default
//	deleted: if true, show deleted files
func (c *Client) Search(path, query string, fileLimit int, deleted bool) (meta []*Metadata, err error) {
	params := c.makeParams(true)
	params.Set("query", query)
	if fileLimit > 0 {
		params.Set("file_limit", strconv.FormatInt(int64(fileLimit), 10))
	}
	if deleted {
		params.Set("include_deleted", "true")
	}

	err = c.getJSON(SearchURL+c.filePath(path), params, &meta)
	return
}

// Delta returns a list of all changes to the dropbox. If cursor is
// the empty string, then changes from the creation of the dropbox are
// given, otherwise, changes since the mentioned cursor are given.
func (c *Client) Delta(cursor string) (delta *Delta, err error) {
	params := c.makeParams(true)
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	err = c.postFormJSON(DeltaURL, params, &delta)
	return
}

// Media gets a URL to the given path that is accessible without login.
// It is expected to not last long.
func (c *Client) Media(path string) (media *Share, err error) {
	params := c.makeParams(true)
	err = c.postFormJSON(MediaURL+c.filePath(path), params, &media)
	return
}

// Shares returns a semi-permanent url to access the given path. If shortURL is set
// then a URL shortened version is provided.
func (c *Client) Shares(path string, shortURL bool) (share *Share, err error) {
	params := c.makeParams(true)
	if shortURL {
		params.Set("short_url", "true")
	}
	err = c.postFormJSON(SharesURL+c.filePath(path), params, &share)
	return
}

// Revisions returns up to revLimit (or default # if 0) sets of metadata for previous
// versions of the file/folder at the given path.
func (c *Client) Revisions(path string, revLimit int) (revs []Metadata, err error) {
	params := c.makeParams(true)
	if revLimit > 0 {
		params.Set("rev_limit", strconv.FormatInt(int64(revLimit), 10))
	}
	err = c.getJSON(RevisionsURL+c.filePath(path), params, &revs)
	return
}

// Restore restores a file to the given path with the given revision.
func (c *Client) Restore(path, rev string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("rev", rev)
	err = c.getJSON(RestoreURL+c.filePath(path), params, &meta)
	return
}

// CopyRef get a reference to the file/folder at the path provided that is
// unique across Dropbox, and can be use to share across users.
func (c *Client) CopyRef(path string) (ref *CopyRef, err error) {
	params := c.makeParams(false)
	err = c.getJSON(CopyRefURL+c.filePath(path), params, &ref)
	return
}

// ChunkedUpload performs file upload in multiple chunks. Set uploadId to empty string for initial chunk.
// If offset does not match the expected, an APIError with bad request code and a ChunkedUpload with
// expected state are returned.
func (c *Client) ChunkedUpload(uploadId string, offset int64, data io.Reader, size int64) (*ChunkedUpload, error) {
	params := c.makeParams(false)
	if uploadId != "" {
		params.Set("upload_id", uploadId)
		params.Set("offset", strconv.FormatInt(offset, 10))
	}

	r, err := c.put(ChunkedUploadURL, params, data, size)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var state ChunkedUpload

	// When offset does not match, 400 status and expected state are returned.
	if r.StatusCode == http.StatusBadRequest {
		apierr := &APIError{
			Code: r.StatusCode,
		}
		if body, err := ioutil.ReadAll(r.Body); err != nil {
			return nil, err
		} else if err := json.Unmarshal(body, &state); err != nil {
			return nil, err
		} else if err := json.Unmarshal(body, apierr); err != nil {
			return nil, err
		}
		return &state, apierr
	}

	if err := parseJSON(r, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// CommitChunkedUpload commits a chunked upload.
func (c *Client) CommitChunkedUpload(path string, overwrite bool, parentRev, uploadId string) (meta *Metadata, err error) {
	uri := CommitChunkedUploadURL + c.filePath(path)
	params := c.makeParams(true)
	params.Set("overwrite", strconv.FormatBool(overwrite))
	if parentRev != "" {
		params.Set("parent_rev", parentRev)
	}
	params.Set("upload_id", uploadId)
	err = c.postFormJSON(uri, params, &meta)
	return
}
