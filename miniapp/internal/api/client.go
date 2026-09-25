// Package api contains the authenticated miniapp request executor.
package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/internal/wechat"
)

const platform = "miniapp"

// Client executes authenticated miniapp requests.
type Client struct {
	transport request.Caller
	auth      *auth.Manager
}

// New creates an authenticated request executor.
func New(transportClient request.Caller, manager *auth.Manager) *Client {
	return &Client{transport: transportClient, auth: manager}
}

// Do executes a request and converts platform error fields to core errors.
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
	credential, err := c.auth.Token(ctx)
	if err != nil {
		return err
	}
	if query == nil {
		query = make(url.Values)
	}
	query.Set("access_token", credential.AccessToken)

	meta := new(request.ResponseMeta)
	envelope := wechat.Response{
		Platform:  platform,
		Operation: operation,
		Value:     result,
	}
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  platform,
		Method:    method,
		Path:      path,
		Query:     query,
		Body:      body,
		Result:    &envelope,
		Meta:      meta,
		RetryMode: retryMode,
	})
	if err != nil {
		return err
	}
	return envelope.Error(*meta)
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
