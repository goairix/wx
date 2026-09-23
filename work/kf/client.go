package kf

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides WeChat customer service APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client { return &Client{api: executor} }
