// Package component manages Open Platform component credentials and tickets.
package component

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

const verifyTicketTTL = 12 * time.Hour

// Config contains component credentials used by the component token endpoint.
type Config struct {
	AppID     string
	AppSecret string
}

// Client manages the verify ticket and component access token.
type Client struct {
	config    Config
	transport *transport.Client
	cache     cache.Cache
	ticketKey string
	manager   *auth.Manager
}

// NewClient constructs a component credential client.
func NewClient(
	config Config,
	transportClient *transport.Client,
	store cache.Cache,
	credentialKey string,
	ticketKey string,
) *Client {
	client := &Client{
		config:    config,
		transport: transportClient,
		cache:     store,
		ticketKey: ticketKey,
	}
	client.manager = auth.NewManager(
		"openplatform-component",
		credentialKey,
		store,
		auth.ProviderFunc(client.fetchToken),
	)
	return client
}

// SetVerifyTicket stores a component_verify_ticket received from WeChat.
func (c *Client) SetVerifyTicket(ctx context.Context, ticket string) error {
	if ticket == "" {
		return fmt.Errorf("openplatform component: verify ticket is required")
	}
	return c.cache.Put(ctx, c.ticketKey, ticket, verifyTicketTTL)
}

// VerifyTicket returns the most recently received component verify ticket.
func (c *Client) VerifyTicket(ctx context.Context) (string, bool, error) {
	return c.cache.Get(ctx, c.ticketKey)
}

// Token returns a cached or freshly issued component credential.
func (c *Client) Token(ctx context.Context) (auth.Credential, error) {
	return c.manager.Token(ctx)
}

func (c *Client) fetchToken(ctx context.Context) (auth.Credential, error) {
	ticket, ok, err := c.VerifyTicket(ctx)
	if err != nil {
		return auth.Credential{}, err
	}
	if !ok || ticket == "" {
		return auth.Credential{}, fmt.Errorf("openplatform component: verify ticket is unavailable")
	}

	var response struct {
		api.ErrorFields
		AccessToken string `json:"component_access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: "openplatform.component.token",
		Platform:  "openplatform",
		Method:    http.MethodPost,
		Path:      "cgi-bin/component/api_component_token",
		Body: struct {
			AppID     string `json:"component_appid"`
			AppSecret string `json:"component_appsecret"`
			Ticket    string `json:"component_verify_ticket"`
		}{
			AppID:     c.config.AppID,
			AppSecret: c.config.AppSecret,
			Ticket:    ticket,
		},
		Result: &response,
		Meta:   meta,
	})
	if err != nil {
		return auth.Credential{}, err
	}
	if err := api.Error("openplatform.component.token", response.ErrorFields, meta); err != nil {
		return auth.Credential{}, err
	}
	if response.AccessToken == "" {
		return auth.Credential{}, fmt.Errorf("openplatform component: token response is empty")
	}
	return auth.Credential{
		AccessToken: response.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
	}, nil
}
