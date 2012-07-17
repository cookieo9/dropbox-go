package dropbox

func (c *Client) Copy(to_path, from_path, from_copy_ref string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("to_path", to_path)
	params.Set("root", string(c.root))
	if from_path != "" {
		params.Set("from_path", from_path)
	}
	if from_copy_ref != "" {
		params.Set("from_copy_ref", from_copy_ref)
	}

	err = c.postFormJSON(API_FILEOPS_COPY, params, &meta)
	return
}

func (c *Client) CreateFolder(path string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("path", path)
	params.Set("root", string(c.root))
	err = c.postFormJSON(API_FILEOPS_CREATE_FOLDER, params, &meta)
	return
}

func (c *Client) Delete(path string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("path", path)
	params.Set("root", string(c.root))
	err = c.postFormJSON(API_FILEOPS_DELETE, params, &meta)
	return
}

func (c *Client) Move(to_path, from_path string) (meta *Metadata, err error) {
	params := c.makeParams(true)
	params.Set("from_path", from_path)
	params.Set("to_path", to_path)
	params.Set("root", string(c.root))

	err = c.postFormJSON(API_FILEOPS_MOVE, params, &meta)
	return
}
