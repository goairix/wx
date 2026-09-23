// Package authorizer provides enterprise third-party authorization APIs.
package authorizer

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides authorized enterprise APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client { return &Client{api: executor} }
