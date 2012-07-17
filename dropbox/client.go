package dropbox

import (
	"io"
	"net/http"
	"strconv"
)

// The AccessRoot type represents the enumeration of Dropbox Root options either "sandbox", where the
// user provides access only to a app-specific folder, and "dropbox" where their entire dropbox is
// available.
type AccessRoot string

const (
	DropboxRoot   AccessRoot = "dropbox"
	SandboxRoot   AccessRoot = "sandbox"
	AppFolderRoot AccessRoot = "sandbox" // Alias for "sandbox"
)

// A Client provides access to the Dropbox services.
type Client struct {
	*Session
	root AccessRoot
}

// URLs for all the Dropbox REST-API Calls
const (
	API_ACCOUNT_INFO          = API_PREFIX + "/account/info"
	API_FILES                 = CNT_PREFIX + "/files"
	API_FILES_PUT             = CNT_PREFIX + "/files_put"
	API_METADATA              = API_PREFIX + "/metadata"
	API_DELTA                 = API_PREFIX + "/delta"
	API_REVISIONS             = API_PREFIX + "/revisions"
	API_RESTORE               = API_PREFIX + "/restore"
	API_SEARCH                = API_PREFIX + "/search"
	API_SHARES                = API_PREFIX + "/shares"
	API_MEDIA                 = API_PREFIX + "/media"
	API_COPY_REF              = API_PREFIX + "/copy_ref"
	API_THUMBNAILS            = CNT_PREFIX + "/thumbnails"
	API_FILEOPS_COPY          = API_PREFIX + "/fileops/copy"
	API_FILEOPS_CREATE_FOLDER = API_PREFIX + "/fileops/create_folder"
	API_FILEOPS_DELETE        = API_PREFIX + "/fileops/delete"
	API_FILEOPS_MOVE          = API_PREFIX + "/fileops/move"
)

// Create a new client using the given authorized session, and working on
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
	err = c.getJSON(API_ACCOUNT_INFO, c.makeParams(true), &account)
	return
}

func (c *Client) GetFile(path string, rev string) (io.ReadCloser, *Metadata, error) {
	uri := API_FILES + c.filePath(path)
	params := c.makeParams(false)
	if rev != "" {
		params.Set("rev", rev)
	}
	return c.fileAccess(uri, params)
}

func (c *Client) Thumbnail(path, format, size string) (io.ReadCloser, *Metadata, error) {
	uri := API_THUMBNAILS + c.filePath(path)
	params := c.makeParams(false)
	if format != "" {
		params.Set("format", format)
	}
	if size != "" {
		params.Set("size", size)
	}
	return c.fileAccess(uri, params)
}

func (c Client) PutFile(path string, overwrite bool, parent_rev string, data io.Reader, size int64) (meta *Metadata, err error) {
	uri := API_FILES_PUT + c.filePath(path)
	params := c.makeParams(true)
	if overwrite {
		params.Set("overwrite", "true")
	}
	if parent_rev != "" {
		params.Set("parent_rev", parent_rev)
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
func (c *Client) Metadata(path string, file_limit int, hash string, list, deleted bool, rev string) (meta *Metadata, unmodified bool, err error) {
	params := c.makeParams(true)
	if file_limit > 0 {
		params.Set("file_limit", strconv.FormatInt(int64(file_limit), 10))
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

	err = c.getJSON(API_METADATA+c.filePath(path), params, &meta)
	if apierr, ok := err.(*APIError); ok && apierr.Code == http.StatusNotModified {
		unmodified = true
		err = nil
	}

	return
}

func (c *Client) Search(path, query string, file_limit int, deleted bool) (meta []*Metadata, err error) {
	params := c.makeParams(true)
	params.Set("query", query)
	if file_limit > 0 {
		params.Set("file_limit", strconv.FormatInt(int64(file_limit), 10))
	}
	if deleted {
		params.Set("include_deleted", "true")
	}

	err = c.getJSON(API_SEARCH+c.filePath(path), params, &meta)
	return
}

func (c *Client) Delta(cursor string) (delta *Delta, err error) {
	params := c.makeParams(true)
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	err = c.postFormJSON(API_DELTA, params, &delta)
	return
}

func (c *Client) Media(path string) (media *Share, err error) {
	params := c.makeParams(true)
	err = c.postFormJSON(API_MEDIA+c.filePath(path), params, &media)
	return
}

func (c *Client) Shares(path string, short_url bool) (share *Share, err error) {
	params := c.makeParams(true)
	if short_url {
		params.Set("short_url", "true")
	}
	err = c.postFormJSON(API_SHARES+c.filePath(path), params, &share)
	return
}

func (c *Client) Revisions(path string, rev_limit int) (revs []Metadata, err error) {
	params := c.makeParams(true)
	if rev_limit > 0 {
		params.Set("rev_limit", strconv.FormatInt(int64(rev_limit), 10))
	}
	err = c.getJSON(API_REVISIONS+c.filePath(path), params, &revs)
	return
}

func (c *Client) Restore(path, rev string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("rev", rev)
	err = c.getJSON(API_RESTORE+c.filePath(path), params, &meta)
	return
}

func (c *Client) CopyRef(path string) (ref *CopyRef, err error) {
	params := c.makeParams(false)
	err = c.getJSON(API_COPY_REF+c.filePath(path), params, &ref)
	return
}
