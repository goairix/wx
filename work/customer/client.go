package customer

import (
	"context"
	"net/url"
	"strconv"

	"github.com/goairix/wx/v2/work/internal/api"
)

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

func (c *Client) Contacts() *ContactClient {
	return c.contacts
}

func (c *Client) Tags() *TagClient {
	return c.tags
}

func (c *Client) Strategies() *StrategyClient {
	return c.strategies
}

func (c *Client) GroupChats() *GroupChatClient {
	return c.groupChats
}

type ContactClient struct{ api *api.Client }
type TagClient struct{ api *api.Client }
type StrategyClient struct{ api *api.Client }
type GroupChatClient struct{ api *api.Client }

func (c *ContactClient) FollowUsers(ctx context.Context) ([]string, error) {
	var result struct {
		Users []string `json:"follow_user"`
	}
	err := c.api.Get(
		ctx,
		"work.customer.contact.follow_users",
		"cgi-bin/externalcontact/get_follow_user_list",
		nil,
		&result,
	)
	return result.Users, err
}

func (c *ContactClient) List(ctx context.Context, userID string) ([]string, error) {
	var result struct {
		ExternalUserIDs []string `json:"external_userid"`
	}
	err := c.api.Get(
		ctx,
		"work.customer.contact.list",
		"cgi-bin/externalcontact/list",
		url.Values{"userid": []string{userID}},
		&result,
	)
	return result.ExternalUserIDs, err
}

func (c *ContactClient) Get(
	ctx context.Context,
	externalUserID string,
	cursor string,
) (*ExternalContactDetail, error) {
	query := url.Values{"external_userid": []string{externalUserID}}
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	result := new(ExternalContactDetail)
	err := c.api.Get(ctx, "work.customer.contact.get", "cgi-bin/externalcontact/get", query, result)
	return result, err
}

func (c *ContactClient) Remark(ctx context.Context, input RemarkRequest) error {
	return c.api.Post(ctx, "work.customer.contact.remark", "cgi-bin/externalcontact/remark", input, nil)
}

func (c *ContactClient) BatchGet(
	ctx context.Context,
	input BatchGetByUserRequest,
) (*BatchGetByUserResult, error) {
	result := new(BatchGetByUserResult)
	err := c.api.Post(
		ctx,
		"work.customer.contact.batch_get",
		"cgi-bin/externalcontact/batch/get_by_user",
		input,
		result,
	)
	return result, err
}

func (c *TagClient) List(ctx context.Context, input GetCorpTagListRequest) ([]TagGroup, error) {
	var result struct {
		Groups []TagGroup `json:"tag_group"`
	}
	err := c.api.Post(
		ctx,
		"work.customer.tag.list",
		"cgi-bin/externalcontact/get_corp_tag_list",
		input,
		&result,
	)
	return result.Groups, err
}

func (c *TagClient) Create(ctx context.Context, input AddCorpTagRequest) (*TagGroup, error) {
	var result struct {
		Group TagGroup `json:"tag_group"`
	}
	err := c.api.Post(
		ctx,
		"work.customer.tag.create",
		"cgi-bin/externalcontact/add_corp_tag",
		input,
		&result,
	)
	return &result.Group, err
}

func (c *TagClient) Update(ctx context.Context, input EditCorpTagRequest) error {
	return c.api.Post(
		ctx,
		"work.customer.tag.update",
		"cgi-bin/externalcontact/edit_corp_tag",
		input,
		nil,
	)
}

func (c *TagClient) Delete(ctx context.Context, input DelCorpTagRequest) error {
	return c.api.Post(
		ctx,
		"work.customer.tag.delete",
		"cgi-bin/externalcontact/del_corp_tag",
		input,
		nil,
	)
}

func (c *StrategyClient) List(
	ctx context.Context,
	cursor string,
	limit int,
) (*StrategyListResult, error) {
	body := struct {
		Cursor string `json:"cursor,omitempty"`
		Limit  int    `json:"limit,omitempty"`
	}{
		Cursor: cursor,
		Limit:  limit,
	}
	result := new(StrategyListResult)
	err := c.api.Post(
		ctx,
		"work.customer.strategy.list",
		"cgi-bin/externalcontact/customer_strategy/list",
		body,
		result,
	)
	return result, err
}

func (c *StrategyClient) Get(ctx context.Context, strategyID int) (*StrategyInfo, error) {
	result := new(StrategyInfo)
	err := c.api.Post(
		ctx,
		"work.customer.strategy.get",
		"cgi-bin/externalcontact/customer_strategy/get",
		map[string]int{"strategy_id": strategyID},
		result,
	)
	return result, err
}

func (c *StrategyClient) Create(ctx context.Context, input CreateStrategyRequest) (int, error) {
	var result struct {
		ID int `json:"strategy_id"`
	}
	err := c.api.Post(
		ctx,
		"work.customer.strategy.create",
		"cgi-bin/externalcontact/customer_strategy/create",
		input,
		&result,
	)
	return result.ID, err
}

func (c *GroupChatClient) List(
	ctx context.Context,
	input GroupChatListRequest,
) (*GroupChatListResult, error) {
	result := new(GroupChatListResult)
	err := c.api.Post(
		ctx,
		"work.customer.group_chat.list",
		"cgi-bin/externalcontact/groupchat/list",
		input,
		result,
	)
	return result, err
}

func (c *GroupChatClient) Get(
	ctx context.Context,
	input GroupChatGetRequest,
) (*GroupChatDetail, error) {
	result := new(GroupChatDetail)
	err := c.api.Post(
		ctx,
		"work.customer.group_chat.get",
		"cgi-bin/externalcontact/groupchat/get",
		input,
		result,
	)
	return result, err
}

func (c *GroupChatClient) DeleteJoinWay(ctx context.Context, configID string) error {
	return c.api.Post(
		ctx,
		"work.customer.group_chat.delete_join_way",
		"cgi-bin/externalcontact/groupchat/del_join_way",
		map[string]string{"config_id": configID},
		nil,
	)
}

func (c *GroupChatClient) OpenGIDToChatID(ctx context.Context, openGID string) (string, error) {
	var result struct {
		ChatID string `json:"chat_id"`
	}
	err := c.api.Post(
		ctx,
		"work.customer.group_chat.opengid_to_chatid",
		"cgi-bin/externalcontact/opengid_to_chatid",
		map[string]string{"opengid": openGID},
		&result,
	)
	return result.ChatID, err
}

func strategyIDQuery(strategyID int) url.Values {
	return url.Values{"strategy_id": []string{strconv.Itoa(strategyID)}}
}
