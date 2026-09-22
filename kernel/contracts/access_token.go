package contracts

import "github.com/goairix/wx/v2/support/lock"

// AccessToken access token
type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// AccessTokenInterface 公众账号获取token接口
type AccessTokenInterface interface {
	// AccessToken 获取公众账号access_token
	AccessToken() (AccessToken, error)
	// AccessTokenCacheKey 获取公众账号access_token缓存key
	AccessTokenCacheKey() string
}

// AuthorizerAccessTokenInterface 开放平台代公众账号获取token接口
type AuthorizerAccessTokenInterface interface {
	// AuthorizerAccessToken 代公众账号获取access_token
	AuthorizerAccessToken(appId, authorizerRefreshToken string, refresh bool, locker lock.Locker) (AccessToken, error)
	// AuthorizerAccessTokenCacheKey 公众账号access_token缓存key
	AuthorizerAccessTokenCacheKey(appId string) string
}

// AccessTokenProvider 外部access_token提供者接口
// 当使用外部服务统一管理access_token时，实现此接口
// 实现方需自行处理token的缓存和刷新
type AccessTokenProvider interface {
	// GetAccessToken 获取access_token
	GetAccessToken() (AccessToken, error)
}
