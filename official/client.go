package official

import (
	"fmt"
	"strings"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/official/article"
	"github.com/goairix/wx/v2/official/authorizer"
	"github.com/goairix/wx/v2/official/internal/api"
	"github.com/goairix/wx/v2/official/jssdk"
	"github.com/goairix/wx/v2/official/menu"
	"github.com/goairix/wx/v2/official/message"
	"github.com/goairix/wx/v2/official/oauth"
	"github.com/goairix/wx/v2/official/qrcode"
	"github.com/goairix/wx/v2/official/user"
	officialwebhook "github.com/goairix/wx/v2/official/webhook"
)

const defaultBaseURL = "https://api.weixin.qq.com"

// Client is the v2 official account client.
type Client struct {
	config     Config
	transport  *transport.Client
	auth       *auth.Manager
	article    *article.Client
	authorizer *authorizer.Client
	menu       *menu.Client
	messages   *message.Client
	oauth      *oauth.Client
	qrCode     *qrcode.Client
	jsSDK      *jssdk.Client
	users      *user.Client
	webhook    *officialwebhook.Client
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
		tr = transport.New(
			optsState.httpClient,
			baseURL,
			optsState.retry,
			transport.WithHook(optsState.hook),
			transport.WithObserver(optsState.observer),
			transport.WithLogger(optsState.logger),
		)
	}
	credentialCache := optsState.coreCache
	if credentialCache == nil {
		credentialCache = corecache.NewMemory()
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
		client.auth = auth.NewManager("official", identity, credentialCache, provider)
	}
	executor := api.New(tr, client.auth)
	client.article = article.NewClient()
	client.authorizer = authorizer.NewClient(executor)
	client.menu = menu.NewClient(executor)
	client.messages = message.NewClient(executor)
	client.oauth = oauth.NewClient(
		tr,
		config.AppID,
		config.AppSecret,
		optsState.oauthOptions...,
	)
	client.qrCode = qrcode.NewClient(executor)
	client.jsSDK = jssdk.NewClient(executor, config.AppID, credentialCache)
	client.users = user.NewClient(tr, client.auth)
	client.webhook = officialwebhook.NewClient(
		config.AppID,
		config.Token,
		config.EncodingAESKey,
	)
	return client, nil
}

// Article returns the article domain client.
func (c *Client) Article() *article.Client {
	return c.article
}

// Authorizer returns Open Platform binding operations.
func (c *Client) Authorizer() *authorizer.Client {
	return c.authorizer
}

// Menu returns the custom menu domain client.
func (c *Client) Menu() *menu.Client {
	return c.menu
}

// TemplateMessages returns the template message domain client.
func (c *Client) TemplateMessages() *message.Client {
	return c.messages
}

// OAuth returns the OAuth domain client.
func (c *Client) OAuth() *oauth.Client {
	return c.oauth
}

// QRCode returns the parameterized QR code domain client.
func (c *Client) QRCode() *qrcode.Client {
	return c.qrCode
}

// JSSDK returns the JS SDK configuration builder.
func (c *Client) JSSDK() *jssdk.Client {
	return c.jsSDK
}

// Users returns the user management domain client.
func (c *Client) Users() *user.Client {
	return c.users
}

// Webhook returns the typed callback adapter.
func (c *Client) Webhook() *officialwebhook.Client {
	return c.webhook
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return c.config
}
