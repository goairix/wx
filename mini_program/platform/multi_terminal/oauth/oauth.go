package oauth

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/goairix/wx/v2/support/lock"
)

type OAuth struct {
	account contracts.AccountInterface
	locker  lock.Locker
}

type LoginInfo struct {
	Type      string `json:"type"`
	LoginTime string `json:"login_time"`
	AppID     string `json:"appid"`
}

type UserInfo struct {
	UserID      string `json:"user_id"`
	OpenAppInfo struct {
		AppID      string `json:"appid"`
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		HeadImgUrl string `json:"headimgurl"`
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

type VerifyInfo struct {
	LoginInfo LoginInfo `json:"login_info"`
	UserInfo  UserInfo  `json:"user_info"`
}

func NewOAuth(account contracts.AccountInterface, opts ...Option) *OAuth {
	auth := &OAuth{account: account}

	for _, opt := range opts {
		opt(auth)
	}

	if auth.locker == nil {
		auth.locker = &lock.Mutex{}
	}

	return auth
}

func (o *OAuth) CodeToVerifyInfo(code string) (*VerifyInfo, error) {
	apiUrl := fmt.Sprintf(
		"donut/code2verifyinfo?appid=%s&appsecret=%s&code=%s&grant_type=authorization_code",
		o.account.AppId(),
		o.account.AppSecret(),
		code,
	)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type codeResult struct {
		kernelError.ApiError
		VerifyInfo
	}

	var result codeResult
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.VerifyInfo, nil
}
