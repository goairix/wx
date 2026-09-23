package customer

import "github.com/goairix/wx/v2/work/internal/api"

// Client provides enterprise customer APIs.
type Client struct {
	contacts   *ContactClient
	tags       *TagClient
	strategies *StrategyClient
	groupChats *GroupChatClient
}

// NewClient constructs a customer client.
func NewClient(executor *api.Client) *Client {
	return &Client{
		contacts:   &ContactClient{api: executor},
		tags:       &TagClient{api: executor},
		strategies: &StrategyClient{api: executor},
		groupChats: &GroupChatClient{api: executor},
	}
}

func (c *Client) Contacts() *ContactClient     { return c.contacts }
func (c *Client) Tags() *TagClient             { return c.tags }
func (c *Client) Strategies() *StrategyClient  { return c.strategies }
func (c *Client) GroupChats() *GroupChatClient { return c.groupChats }

type ContactClient struct{ api *api.Client }
type TagClient struct{ api *api.Client }
type StrategyClient struct{ api *api.Client }
type GroupChatClient struct{ api *api.Client }
