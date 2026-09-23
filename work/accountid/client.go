// Package accountid provides enterprise account identifier conversions.
package accountid

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides account identifier conversion APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client { return &Client{api: executor} }
