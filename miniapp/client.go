package miniapp

import (
	"fmt"
	"strings"

	authpkg "github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/miniapp/auth"
	"github.com/goairix/wx/v2/miniapp/encryptor"
	"github.com/goairix/wx/v2/miniapp/message"
	"github.com/goairix/wx/v2/miniapp/qrcode"
	"github.com/goairix/wx/v2/miniapp/security"
	"github.com/goairix/wx/v2/miniapp/user"
	"github.com/goairix/wx/v2/miniapp/wxacode"
)

const defaultBaseURL = "https://api.weixin.qq.com"

type Client struct {
	config    Config
	transport *transport.Client
	token     *authpkg.Manager
	auth      *auth.Auth
	users     *user.Client
	messages  *message.Client
	qr        *qrcode.Client
	codes     *wxacode.Client
	security  *security.Client
	encryptor *encryptor.Encryptor
}

// NewClient constructs a context-aware miniapp client.
func NewClient(config Config, opts ...Option) (*Client, error) {
	if strings.TrimSpace(config.AppID) == "" || strings.TrimSpace(config.AppSecret) == "" {
		return nil, fmt.Errorf("miniapp: AppID and AppSecret are required")
	}
	state := &option{}
	for _, opt := range opts {
		if opt != nil {
			opt(state)
		}
	}
	base := state.baseURL
	if base == "" {
		base = defaultBaseURL
	}
	tr := transport.New(state.httpClient, base, state.retry)
	tr.Hook = state.hook
	c := &Client{config: config, transport: tr}
	var cc corecache.Cache = state.cache
	if cc == nil {
		cc = corecache.NewMemory()
	}
	c.token = authpkg.NewManager("miniapp", config.AppID, cc, authpkg.ProviderFunc(c.fetchToken))
	c.auth = auth.New(tr, auth.Config{AppID: config.AppID, AppSecret: config.AppSecret})
	c.users = user.New(tr, c.token)
	c.messages = message.New(tr, c.token)
	c.qr = qrcode.New(tr, c.token)
	c.codes = wxacode.New(tr, c.token)
	c.security = security.New(tr, c.token)
	c.encryptor = encryptor.New()
	return c, nil
}
func (c *Client) Config() Config             { return c.config }
func (c *Client) Auth() *auth.Auth           { return c.auth }
func (c *Client) User() *user.Client         { return c.users }
func (c *Client) Users() *user.Client        { return c.users }
func (c *Client) Message() *message.Client   { return c.messages }
func (c *Client) Messages() *message.Client  { return c.messages }
func (c *Client) QRCode() *qrcode.Client     { return c.qr }
func (c *Client) QrCode() *qrcode.Client     { return c.qr }
func (c *Client) WXACode() *wxacode.Client   { return c.codes }
func (c *Client) WxaCode() *wxacode.Client   { return c.codes }
func (c *Client) Security() *security.Client { return c.security }

func (c *Client) Encryptor() *encryptor.Encryptor { return c.encryptor }
