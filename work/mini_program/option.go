package mini_program

import (
	"github.com/goairix/wx/kernel/contracts"
	"github.com/goairix/wx/support/cache"
	"github.com/goairix/wx/support/lock"
)

// config 小程序配置
type config struct {
	isOpenPlatform         bool
	corpId                 string
	secret                 string
	token                  string
	aesKey                 string
	authorizerRefreshToken string
	authorizerAccount      contracts.AuthorizerInterface
}

// option 小程序选项
type option struct {
	cache          cache.Cache
	cacheKeyPrefix string
	locker         lock.Locker
}

// Option 配置客户端使用的缓存、锁和 access_token 提供者。
type Option func(*option)

// WithCache 设置缓存
func WithCache(cache cache.Cache) Option {
	return func(o *option) {
		o.cache = cache
	}
}

// WithCacheKeyPrefix 设置缓存key前缀
func WithCacheKeyPrefix(cacheKeyPrefix string) Option {
	return func(o *option) {
		o.cacheKeyPrefix = cacheKeyPrefix
	}
}

// WithLocker 设置锁
func WithLocker(locker lock.Locker) Option {
	return func(o *option) {
		o.locker = locker
	}
}
