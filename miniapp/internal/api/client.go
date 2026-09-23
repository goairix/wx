// Package api contains the authenticated miniapp request executor.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
)

const platform = "miniapp"

// Client executes authenticated official account requests.
type Client struct {
	transport *transport.Client
	auth      *auth.Manager
}

// New creates an authenticated request executor.
func New(transportClient *transport.Client, manager *auth.Manager) *Client {
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
	credential, err := c.auth.Token(ctx)
	if err != nil {
		return err
	}
	if query == nil {
		query = make(url.Values)
	}
	query.Set("access_token", credential.AccessToken)

	meta := new(request.ResponseMeta)
	envelope := responseEnvelope{value: result}
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  platform,
		Method:    method,
		Path:      path,
		Query:     query,
		Body:      body,
		Result:    &envelope,
		Meta:      meta,
	})
	if err != nil {
		return err
	}
	if envelope.ErrCode != 0 {
		return &wxerrors.Error{
			Platform:   platform,
			Operation:  operation,
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(envelope.ErrCode),
			Message:    envelope.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return nil
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

type responseEnvelope struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	value   interface{}
}

func (r *responseEnvelope) UnmarshalJSON(data []byte) error {
	type errorFields struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	var fields errorFields
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	r.ErrCode = fields.ErrCode
	r.ErrMsg = fields.ErrMsg
	if r.value == nil || r.ErrCode != 0 {
		return nil
	}
	return json.Unmarshal(data, r.value)
}
