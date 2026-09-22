package contracts

import "github.com/goairix/wx/v2/support/cache"

// AccountInterface 公众账号接口
type AccountInterface interface {
	AccessTokenInterface
	// IsOpenPlatform 是否为开放平台下授权的公众账号
	IsOpenPlatform() bool
	// Token 获取公众账号token
	Token() string
	// AesKey 获取公众账号aes_key
	AesKey() string
	// AppId 获取公众账号appID
	AppId() string
	// AppSecret 获取公众账号app_secret
	AppSecret() string
	// ComponentAppId 获取开放平台appID
	ComponentAppId() string
	// ComponentAccessToken 获取开放平台access_token
	ComponentAccessToken() string
	// Cache 获取缓存实例
	Cache() (cache.Cache, string)
	// PlatformType 平台类型
	PlatformType() string
}
