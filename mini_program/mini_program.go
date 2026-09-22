package mini_program

import (
	"github.com/goairix/wx/v2/base/server"
	"github.com/goairix/wx/v2/kernel/contracts"
	"github.com/goairix/wx/v2/mini_program/auth"
	"github.com/goairix/wx/v2/mini_program/authorizer"
	"github.com/goairix/wx/v2/mini_program/content"
	"github.com/goairix/wx/v2/mini_program/encryptor"
	"github.com/goairix/wx/v2/mini_program/message"
	"github.com/goairix/wx/v2/mini_program/qr_code"
	"github.com/goairix/wx/v2/mini_program/user"
	"github.com/goairix/wx/v2/mini_program/wxa_code"
	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

// MiniProgram 小程序
type MiniProgram struct {
	config *config
	option *option
}

func New(appId, appSecret, token, aesKey string, opts ...Option) *MiniProgram {
	c := &config{
		isOpenPlatform: false,
		appId:          appId,
		appSecret:      appSecret,
		token:          token,
		aesKey:         aesKey,
	}

	o := &option{
		cacheKeyPrefix: cache.DefaultCacheKeyPrefix,
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.cache == nil {
		o.cache = cache.NewMemoryCache()
	}
	if o.locker == nil {
		o.locker = &lock.Mutex{}
	}

	return &MiniProgram{
		config: c,
		option: o,
	}
}

// NewWithOpenPlatform 开放平台代小程序调用接口
func NewWithOpenPlatform(
	appId,
	authorizerRefreshToken,
	token,
	aesKey string,
	authorizerAccount contracts.AuthorizerInterface,
	opts ...Option,
) *MiniProgram {
	c := &config{
		isOpenPlatform:         true,
		appId:                  appId,
		token:                  token,
		aesKey:                 aesKey,
		authorizerRefreshToken: authorizerRefreshToken,
		authorizerAccount:      authorizerAccount,
	}

	o := &option{
		cacheKeyPrefix: cache.DefaultCacheKeyPrefix,
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.cache == nil {
		o.cache = cache.NewMemoryCache()
	}
	if o.locker == nil {
		o.locker = &lock.Mutex{}
	}

	return &MiniProgram{
		config: c,
		option: o,
	}
}

// Authorizer 小程序授权开放平台后的相关接口
// 此类接口只能由授权到开放平台的小程序调用或直接由开放平台调用
func (mp *MiniProgram) Authorizer() *authorizer.Authorizer {
	return authorizer.New(mp)
}

// Server 服务端
func (mp *MiniProgram) Server() *server.Server {
	return server.New(mp)
}

// Encryptor 小程序加密数据的解密
func (mp *MiniProgram) Encryptor() *encryptor.Encryptor {
	return encryptor.New(mp)
}

// Auth 用户登录
func (mp *MiniProgram) Auth() *auth.Auth {
	return auth.New(mp)
}

// WxaCode 小程序码
func (mp *MiniProgram) WxaCode() *wxa_code.WxaCode {
	return wxa_code.New(mp)
}

// QrCode 普通链接二维码
func (mp *MiniProgram) QrCode() *qr_code.QrCode {
	return qr_code.New(mp)
}

// ContentSecurity 内容安全
func (mp *MiniProgram) ContentSecurity() *content.Security {
	return content.NewSecurity(mp)
}

// SubscribeMessage 订阅消息
func (mp *MiniProgram) SubscribeMessage() *message.Subscribe {
	return message.NewSubscribe(mp)
}

// User 小程序用户信息
func (mp *MiniProgram) User() *user.User {
	return user.New(mp)
}
