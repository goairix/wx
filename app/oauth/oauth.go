package oauth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/goairix/wx/v2/support/lock"
	"github.com/pkg/errors"
)

// OAuth App用户授权
type OAuth struct {
	account contracts.AccountInterface
	locker  lock.Locker
}

type BaseUserInfo struct {
	OpenID  string `json:"openid"`
	UnionID string `json:"unionid"`
}

type UserInfo struct {
	BaseUserInfo
	Nickname   string `json:"nickname"`
	HeadImgUrl string `json:"headimgurl"`
	Sex        uint8  `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func New(account contracts.AccountInterface, opts ...Option) *OAuth {
	auth := &OAuth{account: account}

	for _, opt := range opts {
		opt(auth)
	}

	if auth.locker == nil {
		auth.locker = &lock.Mutex{}
	}

	return auth
}

// LoginCodeAccessToken 用户登录授权code换取access_token
func (oauth *OAuth) LoginCodeAccessToken(code string) (*BaseUserInfo, error) {
	apiUrl := fmt.Sprintf(
		"sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		oauth.account.AppId(),
		oauth.account.AppSecret(),
		code,
	)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type accessToken struct {
		kernelError.ApiError
		AccessToken
		BaseUserInfo
	}

	var result accessToken
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	cache, _ := oauth.account.Cache()
	// 保存accessToken
	expiration := result.AccessToken.ExpiresIn - 600
	if expiration <= 0 {
		expiration = result.AccessToken.ExpiresIn
	}
	err = cache.Put(
		oauth.account.AccessTokenCacheKey()+":app_user_access_token:"+result.OpenID,
		result.AccessToken.AccessToken,
		time.Second*time.Duration(expiration),
	)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 保存refreshAccessToken
	_ = cache.Put(
		oauth.account.AccessTokenCacheKey()+":app_user_refresh_access_token:"+result.OpenID,
		result.AccessToken.RefreshToken,
		time.Hour*4,
	)

	return &result.BaseUserInfo, nil
}

// UserInfo 获取用户信息
func (oauth *OAuth) UserInfo(openid string) (*UserInfo, error) {
	accessToken, err := oauth.accessToken(openid, false)
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf(
		"sns/userinfo?access_token=%s&openid=%s",
		accessToken,
		openid,
	)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type info struct {
		kernelError.ApiError
		UserInfo
	}
	var result info
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UserInfo, nil
}

// accessToken 获取accessToken
func (oauth *OAuth) accessToken(openid string, refresh bool) (string, error) {
	cache, _ := oauth.account.Cache()
	cacheKey := oauth.account.AccessTokenCacheKey() + ":app_user_access_token:" + openid
cacheLoop:
	if !refresh && cache.IsExist(cacheKey) {
		tokenString, err := cache.Get(cacheKey)
		if err == nil {
			return tokenString, nil
		}
	}

	oauth.locker.Lock()
	defer func() {
		oauth.locker.Unlock()
	}()

	if cache.IsExist(cacheKey) {
		goto cacheLoop
	}

	// 刷新access_token
	return oauth.refreshAccessToken(openid)
}

// refreshAccessToken 刷新accessToken
func (oauth *OAuth) refreshAccessToken(openid string) (string, error) {
	cache, _ := oauth.account.Cache()
	cacheKey := oauth.account.AccessTokenCacheKey() + ":app_user_refresh_access_token:" + openid
	if !cache.IsExist(cacheKey) {
		return "", errors.New("refresh_access_token expire")
	}

	refreshTokenString, _ := cache.Get(cacheKey)
	if refreshTokenString == "" {
		return "", errors.New("refresh_access_token expire")
	}

	apiUrl := fmt.Sprintf(
		"sns/oauth2/refresh_token?appid=%s&grant_type=refresh_token&refresh_token=%s",
		oauth.account.AppId(),
		refreshTokenString,
	)

	res, err := http.Get(apiUrl)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	type accessToken struct {
		kernelError.ApiError
		AccessToken
		BaseUserInfo
	}

	var result accessToken
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	// 保存accessToken
	expiration := result.AccessToken.ExpiresIn - 600
	if expiration <= 0 {
		expiration = result.AccessToken.ExpiresIn
	}
	err = cache.Put(
		oauth.account.AccessTokenCacheKey()+":app_user_access_token:"+result.OpenID,
		result.AccessToken.AccessToken,
		time.Second*time.Duration(expiration),
	)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	// 保存refreshAccessToken
	_ = cache.Put(
		oauth.account.AccessTokenCacheKey()+":app_user_refresh_access_token:"+result.OpenID,
		result.AccessToken.RefreshToken,
		time.Hour*4,
	)

	return result.AccessToken.AccessToken, nil
}
