package customer

import (
	"context"
	"net/url"
)

// CreateSession assigns a customer to an online, bound customer service account.
func (c *Client) CreateSession(ctx context.Context, account, openID string) error {
	return c.api.Post(ctx, "official.customer.session.create", "customservice/kfsession/create", map[string]string{"kf_account": account, "openid": openID}, nil)
}

// CloseSession closes an assigned session.
func (c *Client) CloseSession(ctx context.Context, account, openID string) error {
	return c.api.Post(ctx, "official.customer.session.close", "customservice/kfsession/close", map[string]string{"kf_account": account, "openid": openID}, nil)
}

// GetSession returns the customer's current assigned session.
func (c *Client) GetSession(ctx context.Context, openID string) (*SessionStatus, error) {
	result := new(SessionStatus)
	err := c.api.Get(ctx, "official.customer.session.get", "customservice/kfsession/getsession", url.Values{"openid": {openID}}, result)
	return result, err
}

// ListSessions returns the customers currently served by an account.
func (c *Client) ListSessions(ctx context.Context, account string) ([]Session, error) {
	var result struct {
		Sessions []Session `json:"sessionlist"`
	}
	err := c.api.Get(ctx, "official.customer.session.list", "customservice/kfsession/getsessionlist", url.Values{"kf_account": {account}}, &result)
	return result.Sessions, err
}

// WaitingSessions returns the total and up to 100 waiting customer sessions.
func (c *Client) WaitingSessions(ctx context.Context) (*WaitingSessionList, error) {
	result := new(WaitingSessionList)
	err := c.api.Get(ctx, "official.customer.session.waiting", "customservice/kfsession/getwaitcase", nil, result)
	return result, err
}
