// Package multiterminal provides WeChat multi-terminal identity verification.
package multiterminal

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// LoginInfo describes the terminal used for a login.
type LoginInfo struct {
	Type      string `json:"type"`
	LoginTime string `json:"login_time"`
	AppID     string `json:"appid"`
}

// UserInfo combines the identities returned by multi-terminal login.
type UserInfo struct {
	UserID      string `json:"user_id"`
	OpenAppInfo struct {
		AppID      string `json:"appid"`
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		HeadImgURL string `json:"headimgurl"`
		Nickname   string `json:"nickname"`
	} `json:"openapp_info"`
	PhoneInfo struct {
		Phone string `json:"phone"`
	} `json:"phone_info"`
	AppleInfo struct {
		BundleID    string `json:"bundleid"`
		AppleUserID string `json:"apple_user_id"`
	} `json:"apple_info"`
	MiniProgramInfo struct {
		AppID   string `json:"appid"`
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
	} `json:"miniprogram_info"`
}

// VerifyInfo is the verified terminal and user identity.
type VerifyInfo struct {
	LoginInfo LoginInfo `json:"login_info"`
	UserInfo  UserInfo  `json:"user_info"`
}

// Client exchanges multi-terminal authorization codes.
type Client struct {
	appID     string
	appSecret string
	transport request.Caller
}

// NewClient constructs a multi-terminal client.
func NewClient(
	appID string,
	appSecret string,
	transportClient request.Caller,
) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		transport: transportClient,
	}
}

// CodeToVerifyInfo exchanges a login code for verified user information.
func (c *Client) CodeToVerifyInfo(
	ctx context.Context,
	code string,
) (*VerifyInfo, error) {
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		VerifyInfo
	}
	meta := new(request.ResponseMeta)
	err := c.transport.Do(ctx, request.Request{
		Operation: "miniapp.multiterminal.verify",
		RetryMode: request.RetryNever,
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "donut/code2verifyinfo",
		Query: url.Values{
			"appid":      []string{c.appID},
			"appsecret":  []string{c.appSecret},
			"code":       []string{code},
			"grant_type": []string{"authorization_code"},
		},
		Result: &result,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, &wxerrors.Error{
			Platform:   "miniapp",
			Operation:  "miniapp.multiterminal.verify",
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(result.ErrCode),
			Message:    result.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return &result.VerifyInfo, nil
}
