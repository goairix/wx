package official

import (
	"github.com/goairix/wx/v2/base/jssdk"
	"github.com/goairix/wx/v2/base/server"
	"github.com/goairix/wx/v2/kernel/contracts"
	"github.com/goairix/wx/v2/official/article"
	"github.com/goairix/wx/v2/official/authorizer"
	"github.com/goairix/wx/v2/official/menu"
	"github.com/goairix/wx/v2/official/message"
	"github.com/goairix/wx/v2/official/oauth"
	"github.com/goairix/wx/v2/official/qr_code"
	"github.com/goairix/wx/v2/official/user"
	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

// Official 公众号
type Official struct {
	config *config
	option *option
}

// New 公众号sdk实例
func New(appId, appSecret, token, aesKey string, opts ...Option) *Official {
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

	return &Official{
		config: c,
		option: o,
	}
}

// NewWithOpenPlatform 开放平台代公众号调用接口
func NewWithOpenPlatform(
	appId,
	authorizerRefreshToken,
	token,
	aesKey string,
	authorizerAccount contracts.AuthorizerInterface,
	opts ...Option,
) *Official {
	c := &config{
		isOpenPlatform:         true,
		appId:                  appId,
		authorizerRefreshToken: authorizerRefreshToken,
		token:                  token,
		aesKey:                 aesKey,
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

	return &Official{
		config: c,
		option: o,
	}
}

// Authorizer 公众号授权开放平台后的相关接口
// 此类接口只能由授权到开放平台的公众号调用或直接由开放平台调用
func (official *Official) Authorizer() *authorizer.Authorizer {
	return authorizer.New(official)
}

// Server 服务端
func (official *Official) Server() *server.Server {
	return server.New(official)
}

// OAuth 用户授权
func (official *Official) OAuth() *oauth.OAuth {
	return oauth.New(official)
}

// User 用户管理
func (official *Official) User() *user.User {
	return user.New(official)
}

// Menu 菜单管理
func (official *Official) Menu() *menu.Menu {
	return menu.New(official)
}

// Article 文章管理
func (official *Official) Article() *article.Article {
	return article.New(official)
}

// TemplateMessage 模板消息
func (official *Official) TemplateMessage() *message.TemplateMessage {
	return message.NewTemplateMessage(official)
}

// Jssdk 微信JSSDK
func (official *Official) Jssdk() *jssdk.Jssdk {
	return jssdk.New(official, jssdk.WithLocker(official.option.locker.Clone("jssdk_"+official.config.appId)))
}

// QrCode 带参数的二维码
func (official *Official) QrCode() *qr_code.QrCode {
	return qr_code.New(official)
}
