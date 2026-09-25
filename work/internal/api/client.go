// Package api contains the enterprise WeChat domain request executor.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

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
		RetryMode: retryMode,
		Meta:      meta,
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
	credential, err := c.auth.Token(ctx)
	if err != nil {
		return nil, nil, err
	}
	if query == nil {
		query = make(url.Values)
	}
	query.Set("access_token", credential.AccessToken)
	result := make([]byte, 0)
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  platform,
		Method:    method,
		Path:      path,
		Query:     query,
		Header:    header,
		Body:      body,
		Result:    &result,
		Meta:      meta,
	})
	if err != nil {
		return nil, meta, err
	}
	if looksLikeJSON(meta.Header.Get("Content-Type"), result) {
		var fields struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if json.Unmarshal(result, &fields) == nil && fields.ErrCode != 0 {
			return nil, meta, &wxerrors.Error{
				Platform:   platform,
				Operation:  operation,
				HTTPStatus: meta.StatusCode,
				Code:       strconv.Itoa(fields.ErrCode),
				Message:    fields.ErrMsg,
				RequestID:  meta.RequestID,
			}
		}
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
	credential, err := c.auth.Token(ctx)
	if err != nil {
		return nil, err
	}
	if query == nil {
		query = make(url.Values)
	}
	query.Set("access_token", credential.AccessToken)
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation:      operation,
		Platform:       platform,
		Method:         method,
		Path:           path,
		Query:          query,
		Header:         header,
		Body:           body,
		ResponseWriter: destination,
		Meta:           meta,
	})
	return meta, err
}

func looksLikeJSON(contentType string, data []byte) bool {
	return bytes.Contains([]byte(contentType), []byte("json")) ||
		len(bytes.TrimSpace(data)) > 0 && bytes.TrimSpace(data)[0] == '{'
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
