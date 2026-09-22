package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/kernel/user"

	supportHttp "github.com/goairix/wx/v2/support/http"

	wxerrors "github.com/goairix/wx/v2/core/errors"
)

const oauthBaseUrl = "https://open.weixin.qq.com/connect/oauth2/authorize"

// OAuth 公众号用户授权
type OAuth struct {
	account     contracts.AccountInterface
	scope       string
	redirectUrl string
	state       string
}

func New(account contracts.AccountInterface) *OAuth {
	return &OAuth{account: account, state: "state"}
}

func (auth *OAuth) WithScope(scope string) *OAuth {
	auth.scope = scope
	return auth
}

func (auth *OAuth) WithRedirectUrl(redirectUrl string) *OAuth {
	auth.redirectUrl = redirectUrl
	return auth
}

func (auth *OAuth) WithState(state string) *OAuth {
	auth.state = state
	return auth
}

func (auth *OAuth) buildAuthUrl() string {
	if auth.account.IsOpenPlatform() {
		return fmt.Sprintf(
			"%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&component_appid=%s#wechat_redirect",
			oauthBaseUrl,
			auth.account.AppId(),
			url.QueryEscape(auth.redirectUrl),
			auth.scope,
			auth.state,
			auth.account.ComponentAppId(),
		)
	}

	return fmt.Sprintf(
		"%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		oauthBaseUrl,
		auth.account.AppId(),
		url.QueryEscape(auth.redirectUrl),
		auth.scope,
		auth.state,
	)
}

func (auth *OAuth) AuthUrl() string {
	return auth.buildAuthUrl()
}

func (auth *OAuth) Redirect(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, auth.buildAuthUrl(), http.StatusFound)
}

func (auth *OAuth) UserFromCode(code string) (*user.User, error) {
	token, err := auth.TokenFromCode(code)
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf(
		"sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		token.AccessToken,
		token.Openid,
	)

	res, err := supportHttp.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result userResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, wxerrors.New(result.ErrMsg)
	}

	return &result.User, nil
}

func (auth *OAuth) TokenFromCode(code string) (*AccessTokenResponse, error) {
	apiUrl := auth.getTokenUrl(code)

	res, err := supportHttp.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result AccessTokenResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, wxerrors.New(result.ErrMsg)
	}

	return &result, nil
}

func (auth *OAuth) getTokenUrl(code string) string {
	if auth.account.IsOpenPlatform() {
		return fmt.Sprintf(
			"sns/oauth2/component/access_token?appid=%s&code=%s&grant_type=authorization_code&component_appid=%s&component_access_token=%s",
			auth.account.AppId(),
			code,
			auth.account.ComponentAppId(),
			auth.account.ComponentAccessToken(),
		)
	}

	return fmt.Sprintf(
		"sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		auth.account.AppId(),
		auth.account.AppSecret(),
		code,
	)
}

type userResponse struct {
	kernelError.ApiError
	user.User
}
