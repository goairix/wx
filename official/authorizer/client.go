// Package authorizer manages an official account's Open Platform binding.
package authorizer

import (
	"context"

	"github.com/goairix/wx/v2/official/internal/api"
)

// Client provides authorization-related domains.
type Client struct {
	open *OpenClient
}

// NewClient constructs an authorizer domain client.
func NewClient(executor *api.Client) *Client {
	return &Client{open: &OpenClient{api: executor}}
}

// Open returns Open Platform account binding operations.
func (c *Client) Open() *OpenClient {
	return c.open
}

// OpenClient manages Open Platform account bindings.
type OpenClient struct {
	api *api.Client
}

// Create creates an Open Platform account and binds appID to it.
func (c *OpenClient) Create(ctx context.Context, appID string) (string, error) {
	var result struct {
		OpenAppID string `json:"open_appid"`
	}
	err := c.api.Post(
		ctx,
		"official.authorizer.open.create",
		"cgi-bin/open/create",
		map[string]string{"appid": appID},
		&result,
	)
	return result.OpenAppID, err
}

// Bind binds appID to an existing Open Platform account.
func (c *OpenClient) Bind(ctx context.Context, appID, openAppID string) error {
	return c.api.Post(
		ctx,
		"official.authorizer.open.bind",
		"cgi-bin/open/bind",
		map[string]string{"appid": appID, "open_appid": openAppID},
		nil,
	)
}

// Unbind removes appID from an Open Platform account.
func (c *OpenClient) Unbind(ctx context.Context, appID, openAppID string) error {
	return c.api.Post(
		ctx,
		"official.authorizer.open.unbind",
		"cgi-bin/open/unbind",
		map[string]string{"appid": appID, "open_appid": openAppID},
		nil,
	)
}

// Get returns the Open Platform account currently bound to appID.
func (c *OpenClient) Get(ctx context.Context, appID string) (string, error) {
	var result struct {
		OpenAppID string `json:"open_appid"`
	}
	err := c.api.Post(
		ctx,
		"official.authorizer.open.get",
		"cgi-bin/open/get",
		map[string]string{"appid": appID},
		&result,
	)
	return result.OpenAppID, err
}
