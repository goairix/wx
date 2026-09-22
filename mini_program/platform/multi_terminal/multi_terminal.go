package multi_terminal

import (
	"github.com/goairix/wx/v2/mini_program/platform/multi_terminal/oauth"
	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

// MultiTerminal 多端能力
type MultiTerminal struct {
	config *config
	option *option
}

func New(appId, appSecret, token, aesKey string, opts ...Option) *MultiTerminal {
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
	return &MultiTerminal{
		config: c,
		option: o,
	}
}

func (m *MultiTerminal) OAuth() *oauth.OAuth {
	return oauth.NewOAuth(m)
}
