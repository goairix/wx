// Package menu provides official account menu operations.
package menu

import (
	"context"

	"github.com/goairix/wx/v2/official/internal/api"
)

// Item is one menu item.
type Item struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Key       string `json:"key,omitempty"`
	URL       string `json:"url,omitempty"`
	MediaID   string `json:"media_id,omitempty"`
	ArticleID string `json:"article_id,omitempty"`
	AppID     string `json:"appid,omitempty"`
	PagePath  string `json:"pagepath,omitempty"`
	SubButton []Item `json:"sub_button,omitempty"`
}

// ConditionalMenuItem is a conditional menu and its match rule.
type ConditionalMenuItem struct {
	Button    []Item `json:"button,omitempty"`
	MenuID    int64  `json:"menuid,omitempty"`
	MatchRule struct {
		GroupID            int64 `json:"group_id,omitempty"`
		ClientPlatformType int   `json:"client_platform_type,omitempty"`
	} `json:"matchrule,omitempty"`
}

// List contains the default and conditional menus.
type List struct {
	Menu struct {
		Button []Item `json:"button"`
		MenuID int64  `json:"menuid"`
	} `json:"menu"`
	ConditionalMenu []ConditionalMenuItem `json:"conditionalmenu"`
}

// Client provides official account menu APIs.
type Client struct {
	api Caller
}

// NewClient constructs a menu client.
func NewClient(executor *api.Client) *Client {
	return NewWithCaller(executor)
}

// NewWithCaller constructs a domain client with an authenticated caller.
func NewWithCaller(executor Caller) *Client {
	return &Client{api: executor}
}

// Create replaces the default menu.
func (c *Client) Create(ctx context.Context, items []Item) error {
	return c.api.Post(
		ctx,
		"official.menu.create",
		"cgi-bin/menu/create",
		map[string]interface{}{"button": items},
		nil,
	)
}

// Delete removes the default menu.
func (c *Client) Delete(ctx context.Context) error {
	return c.api.GetOnce(ctx, "official.menu.delete", "cgi-bin/menu/delete", nil, nil)
}

// Info returns the current default and conditional menus.
func (c *Client) Info(ctx context.Context) (*List, error) {
	result := new(List)
	err := c.api.Get(ctx, "official.menu.info", "cgi-bin/menu/get", nil, result)
	return result, err
}
