package official

import (
	"fmt"
	"strings"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/official/oauth"
	"github.com/goairix/wx/v2/official/user"
)

const defaultBaseURL = "https://api.weixin.qq.com"

// Client is the v2 official account client.
type Client struct {
	config    Config
	transport *transport.Client
	auth      *auth.Manager
	oauth     *oauth.Client
	users     *user.Client
}

// NewClient constructs a context-aware official account client.
func NewClient(config Config, opts ...Option) (*Client, error) {
	optsState := &option{}
	for _, opt := range opts {
		if opt != nil {
			opt(optsState)
		}
	}
	if strings.TrimSpace(config.AppID) == "" {
		return nil, fmt.Errorf("official: AppID is required")
	}
	if strings.TrimSpace(config.AppSecret) == "" &&
		optsState.credentialProvider == nil &&
		optsState.credentialManager == nil {
		return nil, fmt.Errorf("official: AppSecret is required without a credential provider")
	}
	baseURL := optsState.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	tr := optsState.transport
	if tr == nil {
		tr = transport.New(optsState.httpClient, baseURL, optsState.retry)
		tr.Hook = optsState.hook
	}
	cache := optsState.coreCache
	if cache == nil {
		cache = corecache.NewMemory()
	}
	client := &Client{config: config, transport: tr}
	client.auth = optsState.credentialManager
	if client.auth == nil {
		provider := optsState.credentialProvider
		if provider == nil {
			provider = auth.ProviderFunc(client.fetchToken)
		}
		identity := optsState.credentialIdentity
		if identity == "" {
			identity = config.AppID
		}
		client.auth = auth.NewManager("official", identity, cache, provider)
	}
	client.oauth = oauth.NewClient(tr, config.AppID, config.AppSecret)
	client.users = user.NewClient(tr, client.auth)
	return client, nil
}

// OAuth returns the OAuth domain client.
func (c *Client) OAuth() *oauth.Client {
	return c.oauth
}

// Users returns the user management domain client.
func (c *Client) Users() *user.Client {
	return c.users
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return c.config
}
