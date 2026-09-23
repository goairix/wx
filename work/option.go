package work

import (
	"net/http"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
)

type option struct {
	coreCache  corecache.Cache
	baseURL    string
	httpClient *http.Client
	retry      transport.RetryPolicy
	hook       observability.Hook
	logger     logging.Logger
}

// Option configures an enterprise WeChat client.
type Option func(*option)

// WithCache configures the shared credential cache.
func WithCache(value corecache.Cache) Option {
	return func(settings *option) {
		settings.coreCache = value
	}
}

// WithCoreCache is an alias for WithCache.
func WithCoreCache(value corecache.Cache) Option {
	return WithCache(value)
}

// WithBaseURL overrides the enterprise WeChat API endpoint.
func WithBaseURL(value string) Option {
	return func(settings *option) {
		settings.baseURL = value
	}
}

// WithHTTPClient configures the HTTP client.
func WithHTTPClient(value *http.Client) Option {
	return func(settings *option) {
		settings.httpClient = value
	}
}

// WithRetryPolicy configures transport retries.
func WithRetryPolicy(value transport.RetryPolicy) Option {
	return func(settings *option) {
		settings.retry = value
	}
}

// WithHook configures request observability.
func WithHook(value observability.Hook) Option {
	return func(settings *option) {
		settings.hook = value
	}
}

// WithLogger configures structured request logging.
func WithLogger(logger logging.Logger) Option {
	return func(settings *option) {
		settings.logger = logger
	}
}
