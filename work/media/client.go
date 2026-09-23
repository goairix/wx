package media

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides enterprise media APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client { return &Client{api: executor} }
