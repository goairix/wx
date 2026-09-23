package openplatform

import (
	"net/http"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
)

// Config configures a WeChat Open Platform component.
type Config struct {
	AppID          string
	AppSecret      string
	Token          string
	EncodingAESKey string
}

type option struct {
	httpClient *http.Client
	baseURL    string
	cache      corecache.Cache
	retry      transport.RetryPolicy
	hook       observability.Hook
}

// Option customizes an Open Platform client.
type Option func(*option)

// WithHTTPClient configures the HTTP client used for API calls.
func WithHTTPClient(client *http.Client) Option {
	return func(settings *option) {
		settings.httpClient = client
	}
}

// WithBaseURL overrides the WeChat API endpoint.
func WithBaseURL(baseURL string) Option {
	return func(settings *option) {
		settings.baseURL = baseURL
	}
}

// WithCache configures the component and authorizer credential cache.
func WithCache(store corecache.Cache) Option {
	return func(settings *option) {
		settings.cache = store
	}
}

// WithRetryPolicy configures HTTP retry behavior.
func WithRetryPolicy(policy transport.RetryPolicy) Option {
	return func(settings *option) {
		settings.retry = policy
	}
}

// WithHook configures request observability callbacks.
func WithHook(hook observability.Hook) Option {
	return func(settings *option) {
		settings.hook = hook
	}
}
