package official

import (
	"net/http"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/official/oauth"
)

type option struct {
	coreCache          corecache.Cache
	baseURL            string
	httpClient         *http.Client
	retry              transport.RetryPolicy
	hook               observability.Hook
	observer           observability.Observer
	logger             logging.Logger
	transport          *transport.Client
	credentialProvider auth.Provider
	credentialIdentity string
	credentialManager  *auth.Manager
	oauthOptions       []oauth.Option
}

// Option configures an official account client.
type Option func(*option)

// WithCache configures the shared credential cache.
func WithCache(value corecache.Cache) Option {
	return func(settings *option) {
		settings.coreCache = value
	}
}

// WithBaseURL overrides the API endpoint, primarily for tests and proxies.
func WithBaseURL(baseURL string) Option {
	return func(settings *option) {
		settings.baseURL = baseURL
	}
}

// WithHTTPClient injects the HTTP client used by the shared transport.
func WithHTTPClient(client *http.Client) Option {
	return func(settings *option) {
		settings.httpClient = client
	}
}

// WithRetryPolicy configures transport retries.
func WithRetryPolicy(policy transport.RetryPolicy) Option {
	return func(settings *option) {
		settings.retry = policy
	}
}

// WithHook attaches request observability hooks.
func WithHook(hook observability.Hook) Option {
	return func(settings *option) {
		settings.hook = hook
	}
}

// WithLogger configures structured request logging.
func WithLogger(logger logging.Logger) Option {
	return func(settings *option) {
		settings.logger = logger
	}
}

// WithTransport reuses an existing core transport.
func WithTransport(client *transport.Client) Option {
	return func(settings *option) {
		settings.transport = client
	}
}

// WithCredentialProvider configures externally managed server credentials.
func WithCredentialProvider(identity string, provider auth.Provider) Option {
	return func(settings *option) {
		settings.credentialIdentity = identity
		settings.credentialProvider = provider
	}
}

// WithCredentialManager reuses an existing credential manager.
func WithCredentialManager(manager *auth.Manager) Option {
	return func(settings *option) {
		settings.credentialManager = manager
	}
}

// WithOAuthComponent configures webpage OAuth for an account authorized by a
// WeChat Open Platform component.
func WithOAuthComponent(componentAppID string, credential auth.Provider) Option {
	return func(settings *option) {
		settings.oauthOptions = append(
			settings.oauthOptions,
			oauth.WithComponent(componentAppID, credential),
		)
	}
}

// WithObserver configures a context-propagating observer for each HTTP attempt.
// When a shared transport is supplied, configure its observer on that transport.
func WithObserver(observer observability.Observer) Option {
	return func(settings *option) {
		settings.observer = observer
	}
}
