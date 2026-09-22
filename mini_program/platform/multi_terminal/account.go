package multi_terminal

import "github.com/goairix/wx/v2/support/cache"

func (m *MultiTerminal) IsOpenPlatform() bool {
	return false
}

func (m *MultiTerminal) Token() string {
	return ""
}

func (m *MultiTerminal) AesKey() string {
	return ""
}

func (m *MultiTerminal) AppId() string {
	return m.config.appId
}

func (m *MultiTerminal) AppSecret() string {
	return m.config.appSecret
}

func (m *MultiTerminal) ComponentAppId() string {
	if m.IsOpenPlatform() {
		return m.config.authorizerAccount.ComponentAppId()
	}
	return ""
}

func (m *MultiTerminal) ComponentAccessToken() string {
	if m.IsOpenPlatform() {
		return m.config.authorizerAccount.ComponentAccessToken()
	}
	return ""
}

func (m *MultiTerminal) PlatformType() string {
	return "multi_terminal"
}

func (m *MultiTerminal) Cache() (cache.Cache, string) {
	return m.option.cache, m.option.cacheKeyPrefix
}
