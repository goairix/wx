package authorizer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

// Authorizer 公众账号授权
type Authorizer struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *Authorizer {
	return &Authorizer{account: account}
}

// PreAuthorizationUrl 获取公众账号授权链接
func (authorizer *Authorizer) PreAuthorizationUrl(callbackUrl string, authType AuthType) (string, error) {
	preAuthCode, err := authorizer.getPreAuthCode()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=%d",
		authorizer.account.AppId(),
		preAuthCode.PreAuthCode,
		callbackUrl,
		authType,
	), nil
}

// MobilePreAuthorizationUrl 获取移动端公众账号授权链接
func (authorizer *Authorizer) MobilePreAuthorizationUrl(callbackUrl string, authType AuthType) (string, error) {
	preAuthCode, err := authorizer.getPreAuthCode()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"https://open.weixin.qq.com/wxaopen/safe/bindcomponent?action=%s&no_scan=1&component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=%d#wechat_redirect",
		"bindcomponent",
		authorizer.account.AppId(),
		preAuthCode.PreAuthCode,
		callbackUrl,
		authType,
	), nil
}

// getPreAuthCode 获取预授权码
func (authorizer *Authorizer) getPreAuthCode() (*PreAuthCode, error) {
	accessToken, err := authorizer.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/component/api_create_preauthcode?component_access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"component_appid": authorizer.account.AppId(),
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type preAuthCode struct {
		kernelError.ApiError
		PreAuthCode
	}
	var result preAuthCode
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.PreAuthCode, nil
}

// AuthorizationInfo 使用授权码获取授权信息
func (authorizer *Authorizer) AuthorizationInfo(authCode string) (*AuthorizationInfo, error) {
	accessToken, err := authorizer.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/component/api_query_auth?component_access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"component_appid":    authorizer.account.AppId(),
		"authorization_code": authCode,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type info struct {
		kernelError.ApiError
		AuthorizationInfo
	}
	var result info
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.AuthorizationInfo, nil
}

// AuthorizerInfo 获取授权帐号详情
func (authorizer *Authorizer) AuthorizerInfo(appId string) (*Info, error) {
	accessToken, err := authorizer.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/component/api_get_authorizer_info?component_access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"component_appid":  authorizer.account.AppId(),
		"authorizer_appid": appId,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type info struct {
		kernelError.ApiError
		Info
	}
	var result info
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.Info, nil
}
