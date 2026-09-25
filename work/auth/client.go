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
	api     Caller
	corpID  string
	agentID int64
}

// NewClient constructs an identity client.
func NewClient(executor *api.Client, corpID string, agentID int64) *Client {
	return NewWithCaller(executor, corpID, agentID)
}

// NewWithCaller constructs a domain client with an authenticated caller.
func NewWithCaller(executor Caller, corpID string, agentID int64) *Client {
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
	err := c.api.GetOnce(
		ctx,
		"work.auth.user_from_code",
		"cgi-bin/auth/getuserinfo",
		url.Values{"code": []string{code}},
		result,
	)
	return result, err
}

// UserDetail returns sensitive identity fields for a user ticket.
func (c *Client) UserDetail(ctx context.Context, userTicket string) (*UserDetail, error) {
	body := struct {
		UserTicket string `json:"user_ticket"`
	}{
		UserTicket: userTicket,
	}
	result := new(UserDetail)
	err := c.api.Post(
		ctx,
		"work.auth.user_detail",
		"cgi-bin/auth/getuserdetail",
		body,
		result,
	)
	return result, err
}

// TFAInfo returns two-factor authentication information for a login code.
func (c *Client) TFAInfo(ctx context.Context, code string) (*TfaInfo, error) {
	body := struct {
		Code string `json:"code"`
	}{
		Code: code,
	}
	result := new(TfaInfo)
	err := c.api.Post(ctx, "work.auth.tfa.info", "cgi-bin/auth/get_tfa_info", body, result)
	return result, err
}

// ConfirmLoginTFA confirms login two-factor authentication.
func (c *Client) ConfirmLoginTFA(ctx context.Context, userID string) error {
	return c.api.GetOnce(
		ctx,
		"work.auth.tfa.login_success",
		"cgi-bin/user/authsucc",
		url.Values{"userid": []string{userID}},
		nil,
	)
}

// ConfirmTFA verifies a two-factor authentication code.
func (c *Client) ConfirmTFA(ctx context.Context, userID, code string) error {
	body := struct {
		UserID string `json:"userid"`
		Code   string `json:"tfa_code"`
	}{
		UserID: userID,
		Code:   code,
	}
	return c.api.Post(ctx, "work.auth.tfa.success", "cgi-bin/user/tfa_succ", body, nil)
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
