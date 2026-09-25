package openplatform

import (
	"context"
	"net/http"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/logging"
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
	httpClient        *http.Client
	baseURL           string
	workBaseURL       string
	cache             corecache.Cache
	retry             transport.RetryPolicy
	hook              observability.Hook
	observer          observability.Observer
	logger            logging.Logger
	refreshTokens     RefreshTokenStore
	refreshRepository RefreshTokenRepository
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

// WithWorkBaseURL overrides the enterprise WeChat API endpoint.
func WithWorkBaseURL(baseURL string) Option {
	return func(settings *option) { settings.workBaseURL = baseURL }
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

// WithLogger configures structured request logging.
func WithLogger(logger logging.Logger) Option {
	return func(settings *option) {
		settings.logger = logger
	}
}

// RefreshTokenStore persists a rotated authorizer refresh token.
type RefreshTokenStore interface {
	SaveRefreshToken(ctx context.Context, authorizerAppID, refreshToken string) error
}

// RefreshTokenRepository stores the authoritative authorization state. An empty
// loaded token means the account is not authorized. Implementations must scope
// records to the component when multiple components share a backend.
type RefreshTokenRepository interface {
	RefreshTokenStore
	LoadRefreshToken(ctx context.Context, authorizerAppID string) (string, error)
	DeleteRefreshToken(ctx context.Context, authorizerAppID string) error
}

// WithRefreshTokenRepository loads the current token before each authorizer
// refresh. Seed authorization using UpdateAuthorizer before using child clients.
func WithRefreshTokenRepository(repository RefreshTokenRepository) Option {
	return func(settings *option) {
		settings.refreshRepository = repository
		settings.refreshTokens = repository
	}
}

// RefreshTokenStoreFunc adapts a function to RefreshTokenStore.
type RefreshTokenStoreFunc func(
	ctx context.Context,
	authorizerAppID string,
	refreshToken string,
) error

// SaveRefreshToken calls the adapted function.
func (f RefreshTokenStoreFunc) SaveRefreshToken(
	ctx context.Context,
	authorizerAppID string,
	refreshToken string,
) error {
	return f(ctx, authorizerAppID, refreshToken)
}

// WithRefreshTokenStore persists refresh tokens rotated by WeChat.
func WithRefreshTokenStore(store RefreshTokenStore) Option {
	return func(settings *option) {
		settings.refreshTokens = store
		settings.refreshRepository = nil
	}
}

// WithObserver configures a context-propagating observer for each HTTP attempt.
// When a shared transport is supplied, configure its observer on that transport.
func WithObserver(observer observability.Observer) Option {
	return func(settings *option) {
		settings.observer = observer
	}
}
