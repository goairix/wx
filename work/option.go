package work

import (
	"net/http"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/kernel/contracts"
	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
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
	coreCache           corecache.Cache
	baseURL             string
	httpClient          *http.Client
	retry               transport.RetryPolicy
	hook                observability.Hook
}

// WithCoreCache configures the v2 credential cache.
func WithCoreCache(value corecache.Cache) Option {
	return func(o *option) {
		o.coreCache = value
	}
}

// WithBaseURL overrides the enterprise WeChat API endpoint.
func WithBaseURL(value string) Option {
	return func(o *option) {
		o.baseURL = value
	}
}

// WithHTTPClient configures the HTTP client.
func WithHTTPClient(value *http.Client) Option {
	return func(o *option) {
		o.httpClient = value
	}
}

// WithRetryPolicy configures transport retries.
func WithRetryPolicy(value transport.RetryPolicy) Option {
	return func(o *option) {
		o.retry = value
	}
}

// WithHook configures request observability.
func WithHook(value observability.Hook) Option {
	return func(o *option) {
		o.hook = value
	}
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
