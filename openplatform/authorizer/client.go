// Package authorizer provides Open Platform account authorization APIs.
package authorizer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

// AuthType selects the account types displayed by the authorization page.
type AuthType uint8

const (
	AuthOfficial AuthType = 1
	AuthMiniApp  AuthType = 2
	AuthAll      AuthType = 3
)

// Client provides component account authorization operations.
type Client struct {
	appID      string
	transport  *transport.Client
	credential interface {
		Token(context.Context) (auth.Credential, error)
	}
}

// NewClient constructs an account authorization client.
func NewClient(
	appID string,
	transportClient *transport.Client,
	credential interface {
		Token(context.Context) (auth.Credential, error)
	},
) *Client {
	return &Client{
		appID:      appID,
		transport:  transportClient,
		credential: credential,
	}
}

// PreAuthCode is a short lived code used to open an authorization page.
type PreAuthCode struct {
	Code      string `json:"pre_auth_code"`
	ExpiresIn int64  `json:"expires_in"`
}

// PreAuthorizationCode creates a pre-authorization code.
func (c *Client) PreAuthorizationCode(ctx context.Context) (*PreAuthCode, error) {
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return nil, err
	}

	var response struct {
		api.ErrorFields
		PreAuthCode
	}
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: "openplatform.authorizer.pre_auth_code",
		Platform:  "openplatform",
		Method:    http.MethodPost,
		Path:      "cgi-bin/component/api_create_preauthcode",
		Query: url.Values{
			"component_access_token": {credential.AccessToken},
		},
		Body: struct {
			ComponentAppID string `json:"component_appid"`
		}{
			ComponentAppID: c.appID,
		},
		Result: &response,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if err := api.Error(
		"openplatform.authorizer.pre_auth_code",
		response.ErrorFields,
		meta,
	); err != nil {
		return nil, err
	}
	return &response.PreAuthCode, nil
}

// PreAuthorizationURL creates a desktop account authorization URL.
func (c *Client) PreAuthorizationURL(
	ctx context.Context,
	callbackURL string,
	authType AuthType,
) (string, error) {
	code, err := c.PreAuthorizationCode(ctx)
	if err != nil {
		return "", err
	}
	values := url.Values{
		"component_appid": {c.appID},
		"pre_auth_code":   {code.Code},
		"redirect_uri":    {callbackURL},
		"auth_type":       {fmt.Sprint(authType)},
	}
	return "https://mp.weixin.qq.com/cgi-bin/componentloginpage?" + values.Encode(), nil
}

// MobilePreAuthorizationURL creates a mobile account authorization URL.
func (c *Client) MobilePreAuthorizationURL(
	ctx context.Context,
	callbackURL string,
	authType AuthType,
) (string, error) {
	code, err := c.PreAuthorizationCode(ctx)
	if err != nil {
		return "", err
	}
	values := url.Values{
		"action":          {"bindcomponent"},
		"no_scan":         {"1"},
		"component_appid": {c.appID},
		"pre_auth_code":   {code.Code},
		"redirect_uri":    {callbackURL},
		"auth_type":       {fmt.Sprint(authType)},
	}
	return "https://open.weixin.qq.com/wxaopen/safe/bindcomponent?" +
		values.Encode() + "#wechat_redirect", nil
}

// AuthorizationInfo exchanges an authorization code for account credentials.
func (c *Client) AuthorizationInfo(
	ctx context.Context,
	authorizationCode string,
) (*AuthorizationInfo, error) {
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return nil, err
	}

	var response struct {
		api.ErrorFields
		AuthorizationInfo AuthorizationInfo `json:"authorization_info"`
	}
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: "openplatform.authorizer.authorization_info",
		Platform:  "openplatform",
		Method:    http.MethodPost,
		Path:      "cgi-bin/component/api_query_auth",
		Query: url.Values{
			"component_access_token": {credential.AccessToken},
		},
		Body: struct {
			ComponentAppID    string `json:"component_appid"`
			AuthorizationCode string `json:"authorization_code"`
		}{
			ComponentAppID:    c.appID,
			AuthorizationCode: authorizationCode,
		},
		Result: &response,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if err := api.Error(
		"openplatform.authorizer.authorization_info",
		response.ErrorFields,
		meta,
	); err != nil {
		return nil, err
	}
	return &response.AuthorizationInfo, nil
}

// Info fetches metadata for an authorized account.
func (c *Client) Info(ctx context.Context, authorizerAppID string) (*Info, error) {
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return nil, err
	}

	response := new(Info)
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: "openplatform.authorizer.info",
		Platform:  "openplatform",
		Method:    http.MethodPost,
		Path:      "cgi-bin/component/api_get_authorizer_info",
		Query: url.Values{
			"component_access_token": {credential.AccessToken},
		},
		Body: struct {
			ComponentAppID  string `json:"component_appid"`
			AuthorizerAppID string `json:"authorizer_appid"`
		}{
			ComponentAppID:  c.appID,
			AuthorizerAppID: authorizerAppID,
		},
		Result: response,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if err := api.Error("openplatform.authorizer.info", response.ErrorFields, meta); err != nil {
		return nil, err
	}
	return response, nil
}
