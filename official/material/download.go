package material

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Get retrieves a permanent material as news, video metadata or binary bytes.
func (c *Client) Get(ctx context.Context, mediaID string) (*GetResult, error) {
	raw, meta, err := c.api.Raw(ctx, "official.material.get", http.MethodPost, "cgi-bin/material/get_material", nil, nil, map[string]string{"media_id": mediaID})
	if err != nil {
		return nil, err
	}
	result := &GetResult{ContentType: meta.Header.Get("Content-Type")}
	if isJSON(raw, result.ContentType) {
		if err := json.Unmarshal(raw, result); err != nil {
			return nil, err
		}
	} else {
		result.Data = raw
	}
	return result, nil
}

// GetTemporary retrieves a temporary file or a video URL.
func (c *Client) GetTemporary(ctx context.Context, mediaID string) (*DownloadResult, error) {
	return c.download(ctx, "official.material.temporary.get", "cgi-bin/media/get", mediaID)
}

// GetHDVoice retrieves a high-definition JSSDK voice file.
func (c *Client) GetHDVoice(ctx context.Context, mediaID string) (*DownloadResult, error) {
	return c.download(ctx, "official.material.temporary.get_hd_voice", "cgi-bin/media/get/jssdk", mediaID)
}

func (c *Client) download(ctx context.Context, operation, path, mediaID string) (*DownloadResult, error) {
	raw, meta, err := c.api.Raw(ctx, operation, http.MethodGet, path, url.Values{"media_id": {mediaID}}, nil, nil)
	if err != nil {
		return nil, err
	}
	result := &DownloadResult{ContentType: meta.Header.Get("Content-Type")}
	if isJSON(raw, result.ContentType) {
		if err := json.Unmarshal(raw, result); err != nil {
			return nil, err
		}
	} else {
		result.Data = raw
	}
	return result, nil
}

func isJSON(data []byte, contentType string) bool {
	trimmed := bytes.TrimSpace(data)
	return strings.Contains(strings.ToLower(contentType), "json") || len(trimmed) > 0 && trimmed[0] == '{'
}

// DownloadTo streams a permanent material's response and returns its content
// type. Use Get to decode news/video metadata; this method writes that JSON as-is.
func (c *Client) DownloadTo(ctx context.Context, mediaID string, destination io.Writer) (string, error) {
	return c.downloadTo(ctx, "official.material.get", http.MethodPost, "cgi-bin/material/get_material", nil, map[string]string{"media_id": mediaID}, destination)
}

// DownloadTemporaryTo streams a temporary response. Video URL JSON is written
// as-is; use GetTemporary to decode it. No secondary URL is fetched.
func (c *Client) DownloadTemporaryTo(ctx context.Context, mediaID string, destination io.Writer) (string, error) {
	return c.downloadTo(ctx, "official.material.temporary.get", http.MethodGet, "cgi-bin/media/get", url.Values{"media_id": {mediaID}}, nil, destination)
}

// DownloadHDVoiceTo streams a high-definition JSSDK voice file.
func (c *Client) DownloadHDVoiceTo(ctx context.Context, mediaID string, destination io.Writer) (string, error) {
	return c.downloadTo(ctx, "official.material.temporary.get_hd_voice", http.MethodGet, "cgi-bin/media/get/jssdk", url.Values{"media_id": {mediaID}}, nil, destination)
}

func (c *Client) downloadTo(ctx context.Context, operation, method, path string, query url.Values, body interface{}, destination io.Writer) (string, error) {
	meta, err := c.api.RawTo(ctx, operation, method, path, query, nil, body, destination)
	if err != nil {
		return "", err
	}
	return meta.Header.Get("Content-Type"), nil
}
