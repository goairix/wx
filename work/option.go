package work

import (
	"github.com/goairix/wx/kernel/contracts"
	"github.com/goairix/wx/support/cache"
	"github.com/goairix/wx/support/lock"
)

type config struct {
	corpId                 string
	secret                 string
	token                  string
	aesKey                 string
	authorizerRefreshToken string
	authorizerAccount      contracts.AuthorizerInterface
}

type option struct {
	cache               cache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
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

// WithAccessTokenProvider 设置外部access_token提供者
// 设置后将使用外部提供者获取access_token，不再使用内置的token获取逻辑
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
	return func(o *option) {
		o.accessTokenProvider = provider
	}
}
