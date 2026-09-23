// Package miniapp provides enterprise mini program APIs.
package miniapp

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides enterprise mini program login APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client { return &Client{api: executor} }
