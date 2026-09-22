package official

import (
	"net/http"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/kernel/contracts"
	legacycache "github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

// config stores legacy constructor settings.
type config struct {
	isOpenPlatform         bool
	appId                  string
	appSecret              string
	token                  string
	aesKey                 string
	authorizerRefreshToken string
	authorizerAccount      contracts.AuthorizerInterface
}

// option contains options shared by the legacy and v2 constructors.
type option struct {
	cache               legacycache.Cache
	coreCache           corecache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
	baseURL             string
	httpClient          *http.Client
	retry               transport.RetryPolicy
	hook                observability.Hook
}

type Option func(*option)

// WithCache accepts either the v2 core cache or the legacy cache interface.
func WithCache(value interface{}) Option {
	return func(o *option) {
		switch cache := value.(type) {
		case corecache.Cache:
			o.coreCache = cache
		case legacycache.Cache:
			o.cache = cache
		}
	}
}

func WithCacheKeyPrefix(cacheKeyPrefix string) Option {
	return func(o *option) { o.cacheKeyPrefix = cacheKeyPrefix }
}
func WithLocker(locker lock.Locker) Option { return func(o *option) { o.locker = locker } }
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
	return func(o *option) { o.accessTokenProvider = provider }
}

// WithBaseURL overrides the API endpoint, primarily for tests and proxies.
func WithBaseURL(baseURL string) Option { return func(o *option) { o.baseURL = baseURL } }

// WithHTTPClient injects the HTTP client used by the v2 transport.
func WithHTTPClient(client *http.Client) Option { return func(o *option) { o.httpClient = client } }

// WithRetryPolicy configures v2 transport retries.
func WithRetryPolicy(policy transport.RetryPolicy) Option {
	return func(o *option) { o.retry = policy }
}

// WithHook attaches v2 request observability hooks.
func WithHook(hook observability.Hook) Option { return func(o *option) { o.hook = hook } }
