package authorizer

import (
	"context"
)

// TesterClient manages miniapp experience members.
type TesterClient struct {
	api Caller
}

// Bind adds an experience member and returns its user identifier.
func (c *TesterClient) Bind(ctx context.Context, weChatID string) (string, error) {
	var result struct {
		User string `json:"userstr"`
	}
	err := c.api.Post(
		ctx,
		"miniapp.authorizer.tester.bind",
		"wxa/bind_tester",
		map[string]string{"wechatid": weChatID},
		&result,
	)
	return result.User, err
}

// Unbind removes an experience member. data may contain wechatid or userstr.
func (c *TesterClient) Unbind(
	ctx context.Context,
	data map[string]string,
) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.tester.unbind",
		"wxa/unbind_tester",
		data,
		nil,
	)
}

// GetMemberList returns all bound experience members.
func (c *TesterClient) GetMemberList(
	ctx context.Context,
) ([]map[string]interface{}, error) {
	var result struct {
		Members []map[string]interface{} `json:"members"`
	}
	err := c.api.Post(
		ctx,
		"miniapp.authorizer.tester.list",
		"wxa/memberauth",
		map[string]string{"action": "get_experiencer"},
		&result,
	)
	return result.Members, err
}
