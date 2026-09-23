package healthcard

import (
	"net/http"
	"strings"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
)

type option struct {
	baseURL            string
	httpClient         *http.Client
	transport          *transport.Client
	retry              transport.RetryPolicy
	hook               observability.Hook
	logger             logging.Logger
	cache              cache.Cache
	credentialManager  *auth.Manager
	credentialProvider auth.Provider
	initialAppToken    string
	channelNum         int
	relatedAppID       string
	now                func() time.Time
	requestID          func() string
}

// Option configures a health card client.
type Option func(*option)

// WithBaseURL overrides the Tencent API base URL.
func WithBaseURL(baseURL string) Option {
	return func(settings *option) {
		settings.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient configures the HTTP client used by the shared transport.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(settings *option) {
		if httpClient != nil {
			settings.httpClient = httpClient
		}
	}
}

// WithTransport injects a shared core transport.
func WithTransport(value *transport.Client) Option {
	return func(settings *option) {
		if value != nil {
			settings.transport = value
		}
	}
}

// WithRetry configures transient HTTP retries.
func WithRetry(value transport.RetryPolicy) Option {
	return func(settings *option) {
		settings.retry = value
	}
}

// WithHook observes requests and responses without exposing sensitive values.
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

// WithCache configures the shared appToken cache.
func WithCache(value cache.Cache) Option {
	return func(settings *option) {
		if value != nil {
			settings.cache = value
		}
	}
}

// WithCredentialManager injects a fully configured appToken manager.
func WithCredentialManager(value *auth.Manager) Option {
	return func(settings *option) {
		if value != nil {
			settings.credentialManager = value
		}
	}
}

// WithCredentialProvider obtains appToken credentials from an external source.
func WithCredentialProvider(value auth.Provider) Option {
	return func(settings *option) {
		if value != nil {
			settings.credentialProvider = value
		}
	}
}

// WithAccessTokenProvider is an alias for WithCredentialProvider.
func WithAccessTokenProvider(value auth.Provider) Option {
	return WithCredentialProvider(value)
}

// WithTokenProvider is an alias for WithCredentialProvider.
func WithTokenProvider(value auth.Provider) Option {
	return WithCredentialProvider(value)
}

// WithAppToken seeds the appToken manager with a caller supplied token.
func WithAppToken(appToken string) Option {
	return func(settings *option) {
		settings.initialAppToken = appToken
	}
}

// WithChannelNum configures the channel number included in commonIn.
func WithChannelNum(channelNum int) Option {
	return func(settings *option) {
		settings.channelNum = channelNum
	}
}

// WithRelatedAppID overrides Config.RelatedAppID.
func WithRelatedAppID(appID string) Option {
	return func(settings *option) {
		settings.relatedAppID = appID
	}
}

// WithClock replaces the request timestamp clock, primarily for tests.
func WithClock(now func() time.Time) Option {
	return func(settings *option) {
		if now != nil {
			settings.now = now
		}
	}
}

// WithRequestID replaces the request ID generator, primarily for tracing and tests.
func WithRequestID(requestID func() string) Option {
	return func(settings *option) {
		if requestID != nil {
			settings.requestID = requestID
		}
	}
}
