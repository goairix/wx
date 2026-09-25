package miniapp

import (
	"net/http"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
)

// Config configures a WeChat miniapp client.
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
	observer   observability.Observer
	logger     logging.Logger
	transport  *transport.Client
	provider   auth.Provider
	identity   string
	manager    *auth.Manager
}

// Option customizes a miniapp client.
type Option func(*option)

func WithHTTPClient(c *http.Client) Option {
	return func(o *option) {
		o.httpClient = c
	}
}

func WithBaseURL(v string) Option {
	return func(o *option) {
		o.baseURL = v
	}
}

func WithCache(c corecache.Cache) Option {
	return func(o *option) {
		o.cache = c
	}
}

func WithRetryPolicy(p transport.RetryPolicy) Option {
	return func(o *option) {
		o.retry = p
	}
}

func WithHook(h observability.Hook) Option {
	return func(o *option) {
		o.hook = h
	}
}

// WithLogger configures structured request logging.
func WithLogger(logger logging.Logger) Option {
	return func(o *option) {
		o.logger = logger
	}
}

// WithTransport reuses an existing core transport.
func WithTransport(client *transport.Client) Option {
	return func(o *option) {
		o.transport = client
	}
}

// WithCredentialProvider configures externally managed server credentials.
func WithCredentialProvider(identity string, provider auth.Provider) Option {
	return func(o *option) {
		o.identity = identity
		o.provider = provider
	}
}

// WithCredentialManager reuses an existing credential manager.
func WithCredentialManager(manager *auth.Manager) Option {
	return func(o *option) {
		o.manager = manager
	}
}

// WithObserver configures a context-propagating observer for each HTTP attempt.
// When a shared transport is supplied, configure its observer on that transport.
func WithObserver(observer observability.Observer) Option {
	return func(settings *option) {
		settings.observer = observer
	}
}
