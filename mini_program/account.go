package mini_program

import "github.com/goairix/wx/v2/support/cache"

func (mp *MiniProgram) IsOpenPlatform() bool {
	return mp.config.isOpenPlatform
}

func (mp *MiniProgram) Token() string {
	if mp.IsOpenPlatform() {
		return mp.config.authorizerAccount.AuthorizerAccountToken()
	}
	return ""
}

func (mp *MiniProgram) AesKey() string {
	if mp.IsOpenPlatform() {
		return mp.config.authorizerAccount.AuthorizerAccountAesKey()
	}
	return ""
}

func (mp *MiniProgram) AppId() string {
	return mp.config.appId
}

func (mp *MiniProgram) AppSecret() string {
	return mp.config.appSecret
}

func (mp *MiniProgram) ComponentAppId() string {
	if mp.IsOpenPlatform() {
		return mp.config.authorizerAccount.ComponentAppId()
	}
	return ""
}

func (mp *MiniProgram) ComponentAccessToken() string {
	if mp.IsOpenPlatform() {
		return mp.config.authorizerAccount.ComponentAccessToken()
	}
	return ""
}

func (mp *MiniProgram) PlatformType() string {
	return "mini_program"
}

func (mp *MiniProgram) Cache() (cache.Cache, string) {
	return mp.option.cache, mp.option.cacheKeyPrefix
}
