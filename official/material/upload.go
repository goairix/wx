package material

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/official/internal/fileupload"
)

// UploadTemporary uploads an image, voice, video or thumbnail for temporary use.
func (c *Client) UploadTemporary(ctx context.Context, mediaType Type, filename string, data []byte) (*UploadResult, error) {
	const operation = "official.material.temporary.upload"
	if !uploadType(mediaType) {
		return nil, inputError(operation, "unsupported temporary media type")
	}
	result := new(UploadResult)
	err := c.upload(ctx, operation, "cgi-bin/media/upload", url.Values{"type": {string(mediaType)}}, filename, data, nil, result)
	return result, err
}

// Add uploads a permanent media file. Videos require Description.
func (c *Client) Add(ctx context.Context, input UploadRequest) (*AddResult, error) {
	const operation = "official.material.add"
	if !uploadType(input.Type) {
		return nil, inputError(operation, "unsupported permanent media type")
	}
	if input.Type == TypeVideo && input.Description == nil {
		return nil, inputError(operation, "video description is required")
	}
	var fields map[string]string
	if input.Description != nil {
		description, err := json.Marshal(input.Description)
		if err != nil {
			return nil, err
		}
		fields = map[string]string{"description": string(description)}
	}
	result := new(AddResult)
	err := c.upload(ctx, operation, "cgi-bin/material/add_material", url.Values{"type": {string(input.Type)}}, input.Filename, input.Data, fields, result)
	return result, err
}

// UploadImage uploads an image for article content and returns its URL.
func (c *Client) UploadImage(ctx context.Context, filename string, data []byte) (*ImageResult, error) {
	result := new(ImageResult)
	err := c.upload(ctx, "official.material.upload_image", "cgi-bin/media/uploadimg", nil, filename, data, nil, result)
	return result, err
}

func (c *Client) upload(ctx context.Context, operation, path string, query url.Values, filename string, data []byte, fields map[string]string, result interface{}) error {
	body, header, err := fileupload.Build(filename, data, fields)
	if err != nil {
		return err
	}
	raw, _, err := c.api.Raw(ctx, operation, http.MethodPost, path, query, header, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}
