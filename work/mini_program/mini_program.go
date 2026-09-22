package mini_program

import (
	"github.com/goairix/wx/support/cache"
	"github.com/goairix/wx/support/lock"
	"github.com/goairix/wx/work/mini_program/auth"
)

// MiniProgram 企业微信小程序
type MiniProgram struct {
	config *config
	option *option
}

// New 创建企业微信小程序客户端。
func New(corpId, secret, token, aesKey string, opts ...Option) *MiniProgram {
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

	return &MiniProgram{
		config: c,
		option: o,
	}
}

// Auth 用户登录
func (w *MiniProgram) Auth() *auth.Auth {
	return auth.New(w)
}
