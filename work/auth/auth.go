package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	workHttp "github.com/goairix/wx/v2/work/http"
)

const oauthBaseUrl = "https://open.weixin.qq.com/connect/oauth2/authorize"
const qrLoginBaseUrl = "https://login.work.weixin.qq.com/wwlogin/sso/login"

// Auth 企业微信网页授权登录
type Auth struct {
	account     contracts.AccountInterface
	scope       string
	redirectUrl string
	state       string
	agentId     string
}

// NewAuth 创建 Auth 实例。
func NewAuth(account contracts.AccountInterface) *Auth {
	return &Auth{account: account, state: "state", scope: "snsapi_base"}
}

// WithScope 设置授权作用域
func (auth *Auth) WithScope(scope string) *Auth {
	auth.scope = scope
	return auth
}

// WithRedirectUrl 设置回调URL
func (auth *Auth) WithRedirectUrl(redirectUrl string) *Auth {
	auth.redirectUrl = redirectUrl
	return auth
}

// WithState 设置state参数
func (auth *Auth) WithState(state string) *Auth {
	auth.state = state
	return auth
}

// WithAgentId 设置agentid
func (auth *Auth) WithAgentId(agentId string) *Auth {
	auth.agentId = agentId
	return auth
}

func (auth *Auth) buildAuthUrl() string {
	authUrl := fmt.Sprintf(
		"%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		oauthBaseUrl,
		auth.account.AppId(),
		url.QueryEscape(auth.redirectUrl),
		auth.scope,
		auth.state,
	)
	if auth.agentId != "" {
		authUrl += fmt.Sprintf("&agentid=%s", auth.agentId)
	}
	authUrl += "#wechat_redirect"
	return authUrl
}

// AuthUrl 构造授权链接
func (auth *Auth) AuthUrl() string {
	return auth.buildAuthUrl()
}

// Redirect 302重定向到授权页
func (auth *Auth) Redirect(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, auth.buildAuthUrl(), http.StatusFound)
}

// UserFromCode 通过code获取用户身份
func (auth *Auth) UserFromCode(code string) (*UserIdentity, error) {
	accessToken, err := auth.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/auth/getuserinfo?access_token=%s&code=%s", accessToken.AccessToken, code)
	res, err := workHttp.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result userIdentityResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UserIdentity, nil
}

// GetUserDetail 获取用户敏感信息
func (auth *Auth) GetUserDetail(userTicket string) (*UserDetail, error) {
	accessToken, err := auth.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/auth/getuserdetail?access_token=%s", accessToken.AccessToken)
	res, err := workHttp.PostJSON(apiUrl, map[string]string{"user_ticket": userTicket})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result userDetailResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UserDetail, nil
}

// QrLoginUrl 构造企业微信扫码登录链接
func (auth *Auth) QrLoginUrl(loginType string) string {
	if loginType == "" {
		loginType = "CorpApp"
	}
	qrUrl := fmt.Sprintf(
		"%s?login_type=%s&appid=%s&redirect_uri=%s&state=%s",
		qrLoginBaseUrl,
		loginType,
		auth.account.AppId(),
		url.QueryEscape(auth.redirectUrl),
		auth.state,
	)
	if auth.agentId != "" {
		qrUrl += fmt.Sprintf("&agentid=%s", auth.agentId)
	}
	return qrUrl
}

// QrLoginRedirect 302重定向到扫码登录页
func (auth *Auth) QrLoginRedirect(writer http.ResponseWriter, request *http.Request, loginType string) {
	http.Redirect(writer, request, auth.QrLoginUrl(loginType), http.StatusFound)
}

// TFA 二次验证
func (auth *Auth) TFA() *Tfa {
	return NewTfa(auth.account)
}
