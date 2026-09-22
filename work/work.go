package work

import (
	"github.com/goairix/wx/v2/base/server"
	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
	"github.com/goairix/wx/v2/work/account_id"
	"github.com/goairix/wx/v2/work/auth"
	"github.com/goairix/wx/v2/work/contact"
	"github.com/goairix/wx/v2/work/customer"
	"github.com/goairix/wx/v2/work/kf"
	"github.com/goairix/wx/v2/work/media"
	"github.com/goairix/wx/v2/work/message"
	"github.com/goairix/wx/v2/work/mini_program"
)

// Work 企业微信
type Work struct {
	config *config
	option *option
}

// New 创建企业微信客户端。
func New(corpId, secret, token, aesKey string, opts ...Option) *Work {
	c := &config{
		corpId: corpId,
		secret: secret,
		token:  token,
		aesKey: aesKey,
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

	return &Work{
		config: c,
		option: o,
	}
}

// Auth 身份验证
func (w *Work) Auth() *auth.Auth {
	return auth.NewAuth(w)
}

// AccountId 账号ID
func (w *Work) AccountId() *account_id.AccountId {
	return account_id.New(w)
}

// Contact 通讯录管理
func (w *Work) Contact() *contact.Contact {
	return contact.New(w, w.config.token, w.config.aesKey)
}

// Customer 客户管理
func (w *Work) Customer() *customer.Customer {
	return customer.NewCustomer(w)
}

// Kf 微信客服
func (w *Work) Kf() *kf.Kf {
	return kf.NewKf(w)
}

// Media 素材管理
func (w *Work) Media() *media.Media {
	return media.NewMedia(w)
}

// Message 应用消息
func (w *Work) Message() *message.Message {
	return message.NewMessage(w)
}

// MiniProgram 小程序
func (w *Work) MiniProgram() *mini_program.MiniProgram {
	return mini_program.New(w.config.corpId, w.config.secret, w.config.token, w.config.aesKey)
}

// Server 服务端
func (w *Work) Server() *server.Server {
	return server.New(w,
		server.WithEncryptMode(server.EncryptModeAES),
		server.WithEchoStrMode(server.EchoStrDecrypt),
	)
}
