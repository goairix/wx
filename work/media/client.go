package media

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides enterprise media APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client {
	return &Client{
		api: executor,
	}
}

// Upload uploads temporary media.
func (c *Client) Upload(
	ctx context.Context,
	mediaType string,
	filename string,
	data []byte,
) (*UploadResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("media", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Set("Content-Type", writer.FormDataContentType())
	raw, _, err := c.api.Raw(
		ctx,
		"work.media.upload",
		http.MethodPost,
		"cgi-bin/media/upload",
		url.Values{"type": []string{mediaType}},
		header,
		body.Bytes(),
	)
	if err != nil {
		return nil, err
	}
	result := new(UploadResult)
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Download downloads temporary media.
func (c *Client) Download(ctx context.Context, mediaID string) ([]byte, string, error) {
	return c.download(ctx, "work.media.download", "cgi-bin/media/get", mediaID)
}

// GetJSSDK downloads high-definition voice media uploaded by JSSDK.
func (c *Client) GetJSSDK(ctx context.Context, mediaID string) ([]byte, string, error) {
	return c.download(ctx, "work.media.get_jssdk", "cgi-bin/media/get/jssdk", mediaID)
}

func (c *Client) download(
	ctx context.Context,
	operation string,
	path string,
	mediaID string,
) ([]byte, string, error) {
	raw, meta, err := c.api.Raw(
		ctx,
		operation,
		http.MethodGet,
		path,
		url.Values{"media_id": []string{mediaID}},
		nil,
		nil,
	)
	if err != nil {
		return nil, "", err
	}
	return raw, meta.Header.Get("Content-Type"), nil
}

// UploadImage uploads a permanent image.
func (c *Client) UploadImage(
	ctx context.Context,
	filename string,
	data []byte,
) (*UploadImageResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("media", filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	header := make(http.Header)
	header.Set("Content-Type", writer.FormDataContentType())
	raw, _, err := c.api.Raw(
		ctx,
		"work.media.upload_image",
		http.MethodPost,
		"cgi-bin/media/uploadimg",
		nil,
		header,
		body.Bytes(),
	)
	if err != nil {
		return nil, err
	}
	result := new(UploadImageResult)
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	return result, nil
}

// AsyncUpload starts an asynchronous URL upload.
func (c *Client) AsyncUpload(
	ctx context.Context,
	input AsyncUploadRequest,
) (*AsyncUploadResult, error) {
	result := new(AsyncUploadResult)
	err := c.api.Post(
		ctx,
		"work.media.async_upload",
		"cgi-bin/media/upload_by_url",
		input,
		result,
	)
	return result, err
}
