package auth

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides enterprise identity APIs.
type Client struct {
	api     *api.Client
	corpID  string
	agentID int64
}

// NewClient constructs an identity client.
func NewClient(executor *api.Client, corpID string, agentID int64) *Client {
	return &Client{
		api:     executor,
		corpID:  corpID,
		agentID: agentID,
	}
}

// AuthorizationURL builds an enterprise OAuth URL.
func (c *Client) AuthorizationURL(redirectURL, scope, state string) string {
	if scope == "" {
		scope = "snsapi_base"
	}
	query := url.Values{
		"appid":         []string{c.corpID},
		"redirect_uri":  []string{redirectURL},
		"response_type": []string{"code"},
		"scope":         []string{scope},
		"state":         []string{state},
	}
	if c.agentID > 0 {
		query.Set("agentid", strconv.FormatInt(c.agentID, 10))
	}
	return "https://open.weixin.qq.com/connect/oauth2/authorize?" + query.Encode() + "#wechat_redirect"
}

// UserFromCode exchanges an OAuth code for an enterprise identity.
func (c *Client) UserFromCode(ctx context.Context, code string) (*UserIdentity, error) {
	result := new(UserIdentity)
	err := c.api.Get(
		ctx,
		"work.auth.user_from_code",
		"cgi-bin/auth/getuserinfo",
		url.Values{"code": []string{code}},
		result,
	)
	return result, err
}

// QRLoginURL builds the enterprise QR login URL.
func (c *Client) QRLoginURL(redirectURL, state string) string {
	query := url.Values{
		"login_type":   []string{"CorpApp"},
		"appid":        []string{c.corpID},
		"agentid":      []string{fmt.Sprint(c.agentID)},
		"redirect_uri": []string{redirectURL},
		"state":        []string{state},
	}
	return "https://login.work.weixin.qq.com/wwlogin/sso/login?" + query.Encode()
}
