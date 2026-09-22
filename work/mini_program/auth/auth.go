package auth

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// Auth 小程序用户登录
// @see https://developer.work.weixin.qq.com/document/path/91507
type Auth struct {
	account contracts.AccountInterface
}

// Session 登录凭证
type Session struct {
	CorpId     string `json:"corpid"`
	UserId     string `json:"userid"`
	SessionKey string `json:"session_key"`
}

// New 创建小程序登录客户端。
func New(account contracts.AccountInterface) *Auth {
	return &Auth{account: account}
}

// Session 使用小程序临时登录凭证换取会话信息。
func (auth *Auth) Session(code string) (Session, error) {
	apiUrl := auth.getCodeUrl(code)
	res, err := http.Get(apiUrl)
	if err != nil {
		return Session{}, err
	}

	type sessionResult struct {
		kernelError.ApiError
		Session
	}
	var result sessionResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return Session{}, err
	}

	if result.ErrCode != 0 {
		return Session{}, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Session, nil
}

func (auth *Auth) getCodeUrl(code string) string {
	if auth.account.IsOpenPlatform() {
		accessToken := auth.account.ComponentAccessToken()
		return fmt.Sprintf(
			"cgi-bin/service/miniprogram/jscode2session?suite_access_token=%s&js_code=%s&grant_type=authorization_code",
			accessToken,
			code,
		)
	}

	accessToken, err := auth.account.AccessToken()
	if err != nil {
		return ""
	}
	return fmt.Sprintf(
		"cgi-bin/miniprogram/jscode2session?access_token=%s&js_code=%s&grant_type=authorization_code",
		accessToken.AccessToken,
		code,
	)
}
