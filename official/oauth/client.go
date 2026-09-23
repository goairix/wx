package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// Client provides official account OAuth APIs.
type Client struct {
	transport request.Caller
	appID     string
	appSecret string
}

// NewClient constructs an OAuth domain client.
func NewClient(tr request.Caller, appID, appSecret string) *Client {
	return &Client{transport: tr, appID: appID, appSecret: appSecret}
}

// TokenFromCode exchanges an OAuth authorization code for an access token.
func (c *Client) TokenFromCode(ctx context.Context, code string) (*AccessTokenResponse, error) {
	result := new(AccessTokenResponse)
	meta := &request.ResponseMeta{}
	err := c.transport.Do(ctx, request.Request{
		Operation: "official.oauth.token",
		Platform:  "official",
		Method:    http.MethodGet,
		Path:      "sns/oauth2/access_token",
		Query: url.Values{
			"appid":      {c.appID},
			"secret":     {c.appSecret},
			"code":       {code},
			"grant_type": {"authorization_code"},
		},
		Result: result,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return nil, apiError("official.oauth.token", result.ErrCode, result.ErrMsg, meta)
	}
	return result, nil
}

// UserFromCode exchanges code and fetches the corresponding user profile.
func (c *Client) UserFromCode(ctx context.Context, code string) (*User, error) {
	token, err := c.TokenFromCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return c.UserInfo(ctx, token.AccessToken, token.Openid)
}

// UserInfo fetches a profile by OAuth access token and openid.
func (c *Client) UserInfo(ctx context.Context, accessToken, openID string) (*User, error) {
	result := new(User)
	meta := &request.ResponseMeta{}
	var envelope struct {
		*User
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	envelope.User = result
	err := c.transport.Do(ctx, request.Request{
		Operation: "official.oauth.userinfo",
		Platform:  "official",
		Method:    http.MethodGet,
		Path:      "sns/userinfo",
		Query: url.Values{
			"access_token": {accessToken},
			"openid":       {openID},
			"lang":         {"zh_CN"},
		},
		Result: &envelope,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if envelope.ErrCode != 0 {
		return nil, apiError("official.oauth.userinfo", envelope.ErrCode, envelope.ErrMsg, meta)
	}
	return result, nil
}

func apiError(operation string, code int, message string, meta *request.ResponseMeta) error {
	status := 200
	requestID := ""
	if meta != nil {
		status, requestID = meta.StatusCode, meta.RequestID
	}
	return &wxerrors.Error{
		Platform:   "official",
		Operation:  operation,
		HTTPStatus: status,
		Code:       fmt.Sprintf("%d", code),
		Message:    message,
		RequestID:  requestID,
	}
}
