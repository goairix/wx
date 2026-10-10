package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/internal/wechat"
)

// Raw executes an authenticated request and returns JSON or binary response bytes.
func (c *Client) Raw(ctx context.Context, operation, method, path string, query url.Values, header http.Header, body interface{}) ([]byte, *request.ResponseMeta, error) {
	var data []byte
	meta, err := c.raw(ctx, operation, method, path, query, header, body, &data, nil)
	if err != nil {
		return nil, meta, err
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
	var meta *request.ResponseMeta
	err := wechat.DoWithToken(ctx, c.auth, platform, operation, func(credential auth.Credential) (bool, error) {
		attemptHeader := header.Clone()
		if attemptHeader == nil {
			attemptHeader = make(http.Header)
		}
		attemptHeader.Set("Authorization", "Bearer "+credential.AccessToken)
		var tracker *wechat.WriteTracker
		var writer io.Writer
		if destination != nil {
			tracker = &wechat.WriteTracker{Destination: destination}
			writer = tracker
		}
		data, buffered := result.(*[]byte)
		if buffered {
			*data = nil
		}
		meta = new(request.ResponseMeta)
		err := c.transport.Do(ctx, request.Request{
			Operation: operation, Platform: platform, Method: method, Path: path,
			Query: wechat.TokenQuery(query, credential.AccessToken), Header: attemptHeader,
			Body: body, Result: result, ResponseWriter: writer, Meta: meta,
		})
		// Custom callers may return bytes without decoding platform errors.
		if err == nil && buffered {
			if platformErr := wxerrors.ParseResponseError(platform, operation, meta.StatusCode, *data, meta.RequestID); platformErr != nil {
				err = platformErr
			}
		}
		return tracker == nil || !tracker.Started, err
	})
	return meta, err
}
