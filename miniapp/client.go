package miniapp

import (
	"fmt"
	"strings"

	authpkg "github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/miniapp/auth"
	"github.com/goairix/wx/v2/miniapp/authorizer"
	"github.com/goairix/wx/v2/miniapp/encryptor"
	"github.com/goairix/wx/v2/miniapp/internal/api"
	"github.com/goairix/wx/v2/miniapp/message"
	"github.com/goairix/wx/v2/miniapp/qrcode"
	"github.com/goairix/wx/v2/miniapp/security"
	"github.com/goairix/wx/v2/miniapp/user"
	miniappwebhook "github.com/goairix/wx/v2/miniapp/webhook"
	"github.com/goairix/wx/v2/miniapp/wxacode"
)

const defaultBaseURL = "https://api.weixin.qq.com"

type Client struct {
	config     Config
	transport  *transport.Client
	token      *authpkg.Manager
	auth       *auth.Auth
	authorizer *authorizer.Client
	users      *user.Client
	messages   *message.Client
	qr         *qrcode.Client
	codes      *wxacode.Client
	security   *security.Client
	encryptor  *encryptor.Encryptor
	webhook    *miniappwebhook.Client
}

// NewClient constructs a context-aware miniapp client.
func NewClient(config Config, opts ...Option) (*Client, error) {
	state := &option{}
	for _, opt := range opts {
		if opt != nil {
			opt(state)
		}
	}
	if strings.TrimSpace(config.AppID) == "" {
		return nil, fmt.Errorf("miniapp: AppID is required")
	}
	if strings.TrimSpace(config.AppSecret) == "" && state.provider == nil && state.manager == nil {
		return nil, fmt.Errorf("miniapp: AppSecret is required without a credential provider")
	}
	base := state.baseURL
	if base == "" {
		base = defaultBaseURL
	}
	tr := state.transport
	if tr == nil {
		tr = transport.New(state.httpClient, base, state.retry)
		tr.Hook = state.hook
	}
	c := &Client{config: config, transport: tr}
	var cc corecache.Cache = state.cache
	if cc == nil {
		cc = corecache.NewMemory()
	}
	c.token = state.manager
	if c.token == nil {
		provider := state.provider
		if provider == nil {
			provider = authpkg.ProviderFunc(c.fetchToken)
		}
		identity := state.identity
		if identity == "" {
			identity = config.AppID
		}
		c.token = authpkg.NewManager("miniapp", identity, cc, provider)
	}
	c.auth = auth.New(tr, auth.Config{AppID: config.AppID, AppSecret: config.AppSecret})
	executor := api.New(tr, c.token)
	c.authorizer = authorizer.NewClient(executor)
	c.users = user.New(tr, c.token)
	c.messages = message.New(tr, c.token)
	c.qr = qrcode.New(tr, c.token)
	c.codes = wxacode.New(tr, c.token)
	c.security = security.New(tr, c.token)
	c.encryptor = encryptor.New()
	c.webhook = miniappwebhook.NewClient(
		config.AppID,
		config.Token,
		config.EncodingAESKey,
	)
	return c, nil
}

func (c *Client) Config() Config {
	return c.config
}

func (c *Client) Auth() *auth.Auth {
	return c.auth
}

// Authorizer returns operations available to an authorized miniapp.
func (c *Client) Authorizer() *authorizer.Client {
	return c.authorizer
}

func (c *Client) User() *user.Client {
	return c.users
}

func (c *Client) Users() *user.Client {
	return c.users
}

func (c *Client) Message() *message.Client {
	return c.messages
}

func (c *Client) Messages() *message.Client {
	return c.messages
}

func (c *Client) QRCode() *qrcode.Client {
	return c.qr
}

func (c *Client) QrCode() *qrcode.Client {
	return c.qr
}

func (c *Client) WXACode() *wxacode.Client {
	return c.codes
}

func (c *Client) WxaCode() *wxacode.Client {
	return c.codes
}

func (c *Client) Security() *security.Client {
	return c.security
}

func (c *Client) Encryptor() *encryptor.Encryptor {
	return c.encryptor
}

// Webhook returns the typed callback adapter.
func (c *Client) Webhook() *miniappwebhook.Client {
	return c.webhook
}
