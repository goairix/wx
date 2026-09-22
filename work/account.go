package work

import (
	"github.com/goairix/wx/support/cache"
)

// Token 返回回调校验使用的 Token。
func (w *Work) Token() string {
	if w.IsOpenPlatform() {
		return w.config.authorizerAccount.AuthorizerAccountToken()
	}
	return w.config.token
}

// AesKey 返回回调消息解密使用的 EncodingAESKey。
func (w *Work) AesKey() string {
	if w.IsOpenPlatform() {
		return w.config.authorizerAccount.AuthorizerAccountAesKey()
	}
	return w.config.aesKey
}

// AppId 返回企业 ID。
func (w *Work) AppId() string {
	return w.config.corpId
}

// AppSecret 返回应用 Secret。
func (w *Work) AppSecret() string {
	return w.config.secret
}

// ComponentAppId 返回开放平台组件 AppID；普通企业账号返回空字符串。
func (w *Work) ComponentAppId() string {
	if w.IsOpenPlatform() {
		return w.config.authorizerAccount.ComponentAppId()
	}
	return ""
}

// ComponentAccessToken 返回开放平台组件 access_token；普通企业账号返回空字符串。
func (w *Work) ComponentAccessToken() string {
	if w.IsOpenPlatform() {
		return w.config.authorizerAccount.ComponentAccessToken()
	}
	return ""
}

// IsOpenPlatform 标识当前账号是否由开放平台托管。
func (w *Work) IsOpenPlatform() bool {
	return false
}

// PlatformType 返回账号平台类型标识。
func (w *Work) PlatformType() string {
	return "work"
}

// Cache 返回客户端使用的缓存实例和键前缀。
func (w *Work) Cache() (cache.Cache, string) {
	return w.option.cache, w.option.cacheKeyPrefix
}
