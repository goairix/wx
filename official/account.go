package official

import "github.com/goairix/wx/v2/support/cache"

func (official *Official) Token() string {
	if official.IsOpenPlatform() {
		return official.config.authorizerAccount.AuthorizerAccountToken()
	}
	return official.config.token
}

func (official *Official) AesKey() string {
	if official.IsOpenPlatform() {
		return official.config.authorizerAccount.AuthorizerAccountAesKey()
	}
	return official.config.aesKey
}

func (official *Official) AppId() string {
	return official.config.appId
}

func (official *Official) AppSecret() string {
	return official.config.appSecret
}

func (official *Official) ComponentAppId() string {
	if official.config.isOpenPlatform {
		return official.config.authorizerAccount.ComponentAppId()
	}
	return ""
}

func (official *Official) ComponentAccessToken() string {
	if official.config.isOpenPlatform {
		return official.config.authorizerAccount.ComponentAccessToken()
	}
	return ""
}

// IsOpenPlatform 是否为开放平台下的公众账号
func (official *Official) IsOpenPlatform() bool {
	return official.config.isOpenPlatform
}

func (official *Official) PlatformType() string {
	return "official"
}

// Cache 获取缓存实例
func (official *Official) Cache() (cache.Cache, string) {
	return official.option.cache, official.option.cacheKeyPrefix
}
