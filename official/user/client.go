package user

import (
	"context"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/official/internal/api"
)

// Client provides official account user APIs.
type Client struct {
	transport *transport.Client
	auth      *auth.Manager
	api       *api.Client
	tags      *TagClient
}

// NewClient constructs a user domain client.
func NewClient(tr *transport.Client, manager *auth.Manager) *Client {
	executor := api.New(tr, manager)
	return &Client{
		transport: tr,
		auth:      manager,
		api:       executor,
		tags:      &TagClient{api: executor},
	}
}

// Remark sets the account-local remark for a user.
func (c *Client) Remark(
	ctx context.Context,
	openID string,
	remark string,
) error {
	return c.api.Post(
		ctx,
		"official.user.remark",
		"cgi-bin/user/info/updateremark",
		map[string]string{"openid": openID, "remark": remark},
		nil,
	)
}

// Info fetches a user's profile.
func (c *Client) Info(ctx context.Context, openID string) (*Info, error) {
	result := new(Info)
	err := c.api.Get(
		ctx,
		"official.user.info",
		"cgi-bin/user/info",
		url.Values{
			"openid": []string{openID},
			"lang":   []string{"zh_CN"},
		},
		result,
	)
	return result, err
}

// BatchInfo fetches profiles for several users.
func (c *Client) BatchInfo(
	ctx context.Context,
	users []map[string]string,
) ([]Info, error) {
	var result struct {
		Users []Info `json:"user_info_list"`
	}
	err := c.api.Post(
		ctx,
		"official.user.batch_info",
		"cgi-bin/user/info/batchget",
		map[string]interface{}{"user_list": users},
		&result,
	)
	return result.Users, err
}

// List returns one page of users following the account.
func (c *Client) List(ctx context.Context, nextOpenID string) (*List, error) {
	result := new(List)
	err := c.api.Get(
		ctx,
		"official.user.list",
		"cgi-bin/user/get",
		url.Values{"next_openid": []string{nextOpenID}},
		result,
	)
	return result, err
}

// BlackList returns one page of blocked users.
func (c *Client) BlackList(
	ctx context.Context,
	beginOpenID string,
) (*List, error) {
	result := new(List)
	err := c.api.Post(
		ctx,
		"official.user.blacklist",
		"cgi-bin/tags/members/getblacklist",
		map[string]string{"begin_openid": beginOpenID},
		result,
	)
	return result, err
}

// BatchBlackUser blocks several users.
func (c *Client) BatchBlackUser(
	ctx context.Context,
	openIDs []string,
) error {
	return c.changeBlacklist(
		ctx,
		"batchblacklist",
		openIDs,
	)
}

// BatchUnBlackUser removes several users from the blacklist.
func (c *Client) BatchUnBlackUser(
	ctx context.Context,
	openIDs []string,
) error {
	return c.changeBlacklist(
		ctx,
		"batchunblacklist",
		openIDs,
	)
}

// Tags returns user tag operations.
func (c *Client) Tags() *TagClient {
	return c.tags
}

func (c *Client) changeBlacklist(
	ctx context.Context,
	action string,
	openIDs []string,
) error {
	return c.api.Post(
		ctx,
		"official.user."+action,
		"cgi-bin/tags/members/"+action,
		map[string]interface{}{"openid_list": openIDs},
		nil,
	)
}
