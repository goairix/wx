// Package openplatform provides a context-aware WeChat Open Platform client.
package openplatform

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/miniapp"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/openplatform/authorizer"
	"github.com/goairix/wx/v2/openplatform/code"
	"github.com/goairix/wx/v2/openplatform/component"
	"github.com/goairix/wx/v2/openplatform/template"
	openplatformwebhook "github.com/goairix/wx/v2/openplatform/webhook"
	workauthorizer "github.com/goairix/wx/v2/work/authorizer"
)

const defaultBaseURL = "https://api.weixin.qq.com"

// Client is a WeChat Open Platform component client.
type Client struct {
	config        Config
	transport     *transport.Client
	cache         cache.Cache
	component     *component.Client
	authorizers   *authorizer.Client
	code          *code.Client
	templates     *template.Client
	mu            sync.Mutex
	credentials   map[string]*authorizerCredential
	refreshTokens RefreshTokenStore
	webhook       *openplatformwebhook.Client
}

// NewClient constructs an Open Platform client without making network calls.
func NewClient(config Config, options ...Option) (*Client, error) {
	if strings.TrimSpace(config.AppID) == "" {
		return nil, fmt.Errorf("openplatform: AppID is required")
	}
	if strings.TrimSpace(config.AppSecret) == "" {
		return nil, fmt.Errorf("openplatform: AppSecret is required")
	}

	settings := new(option)
	for _, configure := range options {
		if configure != nil {
			configure(settings)
		}
	}
	baseURL := settings.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	transportClient := transport.New(
		settings.httpClient,
		baseURL,
		settings.retry,
		transport.WithHook(settings.hook),
		transport.WithLogger(settings.logger),
	)
	store := settings.cache
	if store == nil {
		store = cache.NewMemory()
	}

	client := &Client{
		config:        config,
		transport:     transportClient,
		cache:         store,
		credentials:   make(map[string]*authorizerCredential),
		refreshTokens: settings.refreshTokens,
	}
	componentIdentity := credentialIdentity("component", config.AppID)
	ticketIdentity := credentialIdentity("verify-ticket", config.AppID)
	client.component = component.NewClient(
		component.Config{
			AppID:     config.AppID,
			AppSecret: config.AppSecret,
		},
		transportClient,
		store,
		componentIdentity,
		ticketIdentity,
	)
	client.authorizers = authorizer.NewClient(
		config.AppID,
		transportClient,
		client.component,
	)
	client.code = code.NewClient(transportClient, client.authorizerManager)
	client.templates = template.NewClient(transportClient, client.component)
	client.webhook = openplatformwebhook.NewClient(
		config.AppID,
		config.Token,
		config.EncodingAESKey,
	)
	return client, nil
}

// Config returns a copy of the component configuration.
func (c *Client) Config() Config {
	return c.config
}

// Component returns component credential and ticket operations.
func (c *Client) Component() *component.Client {
	return c.component
}

// Authorizers returns account authorization operations.
func (c *Client) Authorizers() *authorizer.Client {
	return c.authorizers
}

// Code returns authorized miniapp code operations.
func (c *Client) Code() *code.Client {
	return c.code
}

// Templates returns component code template operations.
func (c *Client) Templates() *template.Client {
	return c.templates
}

// Webhook returns the typed component callback adapter.
func (c *Client) Webhook() *openplatformwebhook.Client {
	return c.webhook
}

// AcceptVerifyTicket stores a verified ticket event for this component.
func (c *Client) AcceptVerifyTicket(ctx context.Context, componentAppID, ticket string) error {
	if componentAppID != c.config.AppID {
		return fmt.Errorf("openplatform: ticket component AppID does not match config")
	}
	return c.component.SetVerifyTicket(ctx, ticket)
}

// AuthorizedOfficial constructs an official account client that obtains its
// server credential through this Open Platform component.
func (c *Client) AuthorizedOfficial(appID, refreshToken string) (*official.Client, error) {
	manager := c.authorizerManager(appID, refreshToken)
	return official.NewClient(
		official.Config{
			AppID:          appID,
			Token:          c.config.Token,
			EncodingAESKey: c.config.EncodingAESKey,
		},
		official.WithTransport(c.transport),
		official.WithCache(c.cache),
		official.WithCredentialManager(manager),
	)
}

// AuthorizedMiniApp constructs a miniapp client that obtains its server
// credential through this Open Platform component.
func (c *Client) AuthorizedMiniApp(appID, refreshToken string) (*miniapp.Client, error) {
	manager := c.authorizerManager(appID, refreshToken)
	return miniapp.NewClient(
		miniapp.Config{
			AppID:          appID,
			Token:          c.config.Token,
			EncodingAESKey: c.config.EncodingAESKey,
		},
		miniapp.WithTransport(c.transport),
		miniapp.WithCache(c.cache),
		miniapp.WithCredentialManager(manager),
	)
}

// WorkAuthorizer constructs the enterprise authorization domain on the same
// transport. Its methods still require enterprise suite credentials.
func (c *Client) WorkAuthorizer() *workauthorizer.Client {
	return workauthorizer.NewWithTransport(c.transport)
}

func mapValues(key, value string) url.Values {
	return url.Values{key: {value}}
}
