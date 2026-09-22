package open_platform

import (
	"github.com/goairix/wx/v2/kernel/contracts"
	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

// config 开放平台配置
type config struct {
	appId     string
	appSecret string
	token     string
	aesKey    string
}

type option struct {
	cache               cache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
}

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
