// Package api contains the enterprise WeChat domain request executor.
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

const platform = "work"

// Client executes authenticated enterprise WeChat requests.
type Client struct {
	transport request.Caller
	auth      *auth.Manager
}

// New constructs a domain request executor.
func New(transportClient request.Caller, manager *auth.Manager) *Client {
	return &Client{
		transport: transportClient,
		auth:      manager,
	}
}

// Do executes a JSON request and converts platform errors to core errors.
func (c *Client) Do(
	ctx context.Context,
	operation string,
	method string,
	path string,
	query url.Values,
	body interface{},
	result interface{},
) error {
	return c.do(ctx, operation, method, path, query, body, result, request.RetryDefault)
}

func (c *Client) do(
	ctx context.Context,
	operation string,
	method string,
	path string,
	query url.Values,
	body interface{},
	result interface{},
	retryMode request.RetryMode,
) error {
	return wechat.DoWithToken(ctx, c.auth, platform, operation, func(credential auth.Credential) (bool, error) {
		meta := new(request.ResponseMeta)
		envelope := wechat.Response{Platform: platform, Operation: operation, Value: result}
		err := c.transport.Do(ctx, request.Request{
			Operation: operation, Platform: platform, Method: method, Path: path,
			Query: wechat.TokenQuery(query, credential.AccessToken), Body: body,
			Result: &envelope, RetryMode: retryMode, Meta: meta,
		})
		if err == nil {
			err = envelope.Error(*meta)
		}
		return true, err
	})
}

// Get executes an authenticated GET request.
func (c *Client) Get(
	ctx context.Context,
	operation string,
	path string,
	query url.Values,
	result interface{},
) error {
	return c.Do(ctx, operation, http.MethodGet, path, query, nil, result)
}

// Post executes an authenticated JSON POST request.
func (c *Client) Post(
	ctx context.Context,
	operation string,
	path string,
	body interface{},
	result interface{},
) error {
	return c.Do(ctx, operation, http.MethodPost, path, nil, body, result)
}

// Raw executes an authenticated request and returns its undecoded body.
func (c *Client) Raw(
	ctx context.Context,
	operation string,
	method string,
	path string,
	query url.Values,
	header http.Header,
	body []byte,
) ([]byte, *request.ResponseMeta, error) {
	var result []byte
	var meta *request.ResponseMeta
	err := wechat.DoWithToken(ctx, c.auth, platform, operation, func(credential auth.Credential) (bool, error) {
		result = nil
		meta = new(request.ResponseMeta)
		err := c.transport.Do(ctx, request.Request{
			Operation: operation, Platform: platform, Method: method, Path: path,
			Query: wechat.TokenQuery(query, credential.AccessToken), Header: header.Clone(),
			Body: body, Result: &result, Meta: meta,
		})
		if err == nil {
			if platformErr := wxerrors.ParseResponseError(platform, operation, meta.StatusCode, result, meta.RequestID); platformErr != nil {
				err = platformErr
			}
		}
		return true, err
	})
	if err != nil {
		return nil, meta, err
	}
	return result, meta, nil
}

// RawTo executes an authenticated request and streams a successful response.
func (c *Client) RawTo(
	ctx context.Context,
	operation string,
	method string,
	path string,
	query url.Values,
	header http.Header,
	body []byte,
	destination io.Writer,
) (*request.ResponseMeta, error) {
	if destination == nil {
		return nil, fmt.Errorf("work: download destination is required")
	}
	var meta *request.ResponseMeta
	err := wechat.DoWithToken(ctx, c.auth, platform, operation, func(credential auth.Credential) (bool, error) {
		tracker := &wechat.WriteTracker{Destination: destination}
		meta = new(request.ResponseMeta)
		err := c.transport.Do(ctx, request.Request{
			Operation: operation, Platform: platform, Method: method, Path: path,
			Query: wechat.TokenQuery(query, credential.AccessToken), Header: header.Clone(),
			Body: body, ResponseWriter: tracker, Meta: meta,
		})
		return !tracker.Started, err
	})
	return meta, err
}

func (c *Client) String() string {
	return fmt.Sprintf("work api client %p", c)
}

// GetOnce executes a GET whose side effects or one-time input prohibit retries.
func (c *Client) GetOnce(
	ctx context.Context,
	operation string,
	path string,
	query url.Values,
	result interface{},
) error {
	return c.do(ctx, operation, http.MethodGet, path, query, nil, result, request.RetryNever)
}
