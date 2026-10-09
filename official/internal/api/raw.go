package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// Raw executes an authenticated request and returns JSON or binary response bytes.
func (c *Client) Raw(ctx context.Context, operation, method, path string, query url.Values, header http.Header, body interface{}) ([]byte, *request.ResponseMeta, error) {
	var data []byte
	meta, err := c.raw(ctx, operation, method, path, query, header, body, &data, nil)
	if err != nil {
		return nil, meta, err
	}
	// Custom request callers may return raw bytes without decoding platform errors.
	if platformErr := wxerrors.ParseResponseError(platform, operation, meta.StatusCode, data, meta.RequestID); platformErr != nil {
		return nil, meta, platformErr
	}
	return data, meta, nil
}

// RawTo streams a successful response to destination. JSON platform errors are
// intercepted by transport before any bytes are written.
func (c *Client) RawTo(ctx context.Context, operation, method, path string, query url.Values, header http.Header, body interface{}, destination io.Writer) (*request.ResponseMeta, error) {
	if destination == nil {
		return nil, fmt.Errorf("official: download destination is required")
	}
	return c.raw(ctx, operation, method, path, query, header, body, nil, destination)
}

func (c *Client) raw(ctx context.Context, operation, method, path string, query url.Values, header http.Header, body, result interface{}, destination io.Writer) (*request.ResponseMeta, error) {
	credential, err := c.auth.Token(ctx)
	if err != nil {
		return nil, err
	}
	queryCopy := make(url.Values, len(query)+1)
	for key, values := range query {
		queryCopy[key] = append([]string(nil), values...)
	}
	queryCopy.Set("access_token", credential.AccessToken)
	header = header.Clone()
	if header == nil {
		header = make(http.Header)
	}
	header.Set("Authorization", "Bearer "+credential.AccessToken)
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: operation, Platform: platform, Method: method, Path: path,
		Query: queryCopy, Header: header, Body: body, Result: result,
		ResponseWriter: destination, Meta: meta,
	})
	return meta, err
}
