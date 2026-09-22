package app

import (
	"github.com/goairix/wx/v2/support/cache"
)

func (app *App) Token() string {
	if app.IsOpenPlatform() {
		return app.config.authorizerAccount.AuthorizerAccountToken()
	}
	return app.config.token
}

func (app *App) AesKey() string {
	if app.IsOpenPlatform() {
		return app.config.authorizerAccount.AuthorizerAccountAesKey()
	}
	return app.config.aesKey
}

func (app *App) AppId() string {
	return app.config.appId
}

func (app *App) AppSecret() string {
	return app.config.appSecret
}

func (app *App) ComponentAppId() string {
	if app.IsOpenPlatform() {
		return app.config.authorizerAccount.ComponentAppId()
	}
	return ""
}

func (app *App) ComponentAccessToken() string {
	if app.IsOpenPlatform() {
		return app.config.authorizerAccount.ComponentAccessToken()
	}
	return ""
}

func (app *App) IsOpenPlatform() bool {
	return false
}

func (app *App) PlatformType() string {
	return "app"
}

func (app *App) Cache() (cache.Cache, string) {
	return app.option.cache, app.option.cacheKeyPrefix
}
