package dropbox

// Copy copies a file to a location in the user's dropbox from either
// another location in the same dropbox (fromPath) or from a location
// anywhere in all users' dropboxes (fromCopyRef). To copy from a globally
// unique location in all dropboxes, the api call CopyRef must be used.
func (c *Client) Copy(toPath, fromPath, fromCopyRef string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("to_path", toPath)
	params.Set("root", string(c.root))
	if fromPath != "" {
		params.Set("from_path", fromPath)
	}
	if fromCopyRef != "" {
		params.Set("from_copy_ref", fromCopyRef)
	}

	err = c.postFormJSON(FileOpsCopyURL, params, &meta)
	return
}

// CreateFolder creates a folder in the dropbox at the given path.
func (c *Client) CreateFolder(path string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("path", path)
	params.Set("root", string(c.root))
	err = c.postFormJSON(FileOpsCreateFolderURL, params, &meta)
	return
}

// Delete deletes the object at the given path in the dropbox.
func (c *Client) Delete(path string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("path", path)
	params.Set("root", string(c.root))
	err = c.postFormJSON(FileOpsDeleteURL, params, &meta)
	return
}

// Move moves (or renames) a file from one location in the dropbox
// to another.
func (c *Client) Move(toPath, fromPath string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("from_path", fromPath)
	params.Set("to_path", toPath)
	params.Set("root", string(c.root))

	err = c.postFormJSON(FileOpsMoveURL, params, &meta)
	return
}
