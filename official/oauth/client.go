package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

const authorizationEndpoint = "https://open.weixin.qq.com/connect/oauth2/authorize"

// Scope selects the information requested during Official Account OAuth.
type Scope string

const (
	// ScopeBase performs silent authorization and returns the user's OpenID.
	ScopeBase Scope = "snsapi_base"
	// ScopeUserInfo asks the user to authorize access to their profile.
	ScopeUserInfo Scope = "snsapi_userinfo"
)

// Client provides official account OAuth APIs.
type Client struct {
	transport           request.Caller
	appID               string
	appSecret           string
	componentAppID      string
	componentCredential auth.Provider
}

// Option configures an OAuth domain client.
type Option func(*Client)

// WithComponent configures OAuth for an Official Account authorized through
// a WeChat Open Platform component.
func WithComponent(componentAppID string, credential auth.Provider) Option {
	return func(client *Client) {
		client.componentAppID = componentAppID
		client.componentCredential = credential
	}
}

// NewClient constructs an OAuth domain client.
func NewClient(
	tr request.Caller,
	appID string,
	appSecret string,
	options ...Option,
) *Client {
	client := &Client{transport: tr, appID: appID, appSecret: appSecret}
	for _, configure := range options {
		if configure != nil {
			configure(client)
		}
	}
	return client
}

// AuthorizationURL builds the WeChat Official Account OAuth authorization URL.
func (c *Client) AuthorizationURL(redirectURL string, scope Scope, state string) string {
	if scope == "" {
		scope = ScopeBase
	}
	query := url.Values{
		"appid":         {c.appID},
		"redirect_uri":  {redirectURL},
		"response_type": {"code"},
		"scope":         {string(scope)},
		"state":         {state},
	}
	if c.componentAppID != "" {
		query.Set("component_appid", c.componentAppID)
	}
	return authorizationEndpoint + "?" + query.Encode() + "#wechat_redirect"
}

// TokenFromCode exchanges an OAuth authorization code for an access token.
func (c *Client) TokenFromCode(ctx context.Context, code string) (*AccessTokenResponse, error) {
	if c.componentAppID != "" {
		return c.componentTokenFromCode(ctx, code)
	}

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

func (c *Client) componentTokenFromCode(
	ctx context.Context,
	code string,
) (*AccessTokenResponse, error) {
	if c.componentCredential == nil {
		return nil, fmt.Errorf("official oauth: component credential is required")
	}
	credential, err := c.componentCredential.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("official oauth: get component credential: %w", err)
	}

	const operation = "official.oauth.component_token"
	result := new(AccessTokenResponse)
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  "official",
		Method:    http.MethodGet,
		Path:      "sns/oauth2/component/access_token",
		Query: url.Values{
			"appid":                  {c.appID},
			"code":                   {code},
			"grant_type":             {"authorization_code"},
			"component_appid":        {c.componentAppID},
			"component_access_token": {credential.AccessToken},
		},
		Result: result,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return nil, apiError(operation, result.ErrCode, result.ErrMsg, meta)
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
