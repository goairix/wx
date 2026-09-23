package message

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides application and group message APIs.
type Client struct {
	api     *api.Client
	agentID int64
}

func NewClient(executor *api.Client, agentID int64) *Client {
	return &Client{api: executor, agentID: agentID}
}
