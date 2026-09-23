package openplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

type authorizerCredential struct {
	client                *Client
	appID                 string
	mu                    sync.RWMutex
	currentRefreshToken   string
	persistedRefreshToken string
	pendingRefreshToken   string
	manager               *auth.Manager
}

func credentialIdentity(kind string, values ...string) string {
	digest := sha256.New()
	for _, value := range values {
		_, _ = digest.Write([]byte(value))
		_, _ = digest.Write([]byte{0})
	}
	return kind + ":" + hex.EncodeToString(digest.Sum(nil))
}

func (c *Client) authorizerManager(appID, refreshToken string) *auth.Manager {
	identity := credentialIdentity("authorizer", c.config.AppID, appID)

	c.mu.Lock()
	defer c.mu.Unlock()
	if credential := c.credentials[identity]; credential != nil {
		return credential.manager
	}

	credential := &authorizerCredential{
		client:                c,
		appID:                 appID,
		currentRefreshToken:   refreshToken,
		persistedRefreshToken: refreshToken,
	}
	credential.manager = auth.NewManager(
		"openplatform-authorizer",
		identity,
		c.cache,
		auth.ProviderFunc(credential.refresh),
	)
	c.credentials[identity] = credential
	return credential.manager
}

func (c *authorizerCredential) refresh(ctx context.Context) (auth.Credential, error) {
	if err := c.persistPendingRefreshToken(ctx); err != nil {
		return auth.Credential{}, err
	}

	componentCredential, err := c.client.component.Token(ctx)
	if err != nil {
		return auth.Credential{}, err
	}

	c.mu.RLock()
	refreshToken := c.currentRefreshToken
	c.mu.RUnlock()

	var response struct {
		api.ErrorFields
		AccessToken  string `json:"authorizer_access_token"`
		RefreshToken string `json:"authorizer_refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	meta := new(request.ResponseMeta)
	err = c.client.transport.Do(ctx, request.Request{
		Operation: "openplatform.authorizer.token",
		Platform:  "openplatform",
		Method:    http.MethodPost,
		Path:      "cgi-bin/component/api_authorizer_token",
		Query:     mapValues("component_access_token", componentCredential.AccessToken),
		Body: struct {
			ComponentAppID  string `json:"component_appid"`
			AuthorizerAppID string `json:"authorizer_appid"`
			RefreshToken    string `json:"authorizer_refresh_token"`
		}{
			ComponentAppID:  c.client.config.AppID,
			AuthorizerAppID: c.appID,
			RefreshToken:    refreshToken,
		},
		Result: &response,
		Meta:   meta,
	})
	if err != nil {
		return auth.Credential{}, err
	}
	if err := api.Error("openplatform.authorizer.token", response.ErrorFields, meta); err != nil {
		return auth.Credential{}, err
	}
	if response.AccessToken == "" {
		return auth.Credential{}, fmt.Errorf("openplatform authorizer: token response is empty")
	}
	if err := c.acceptRefreshToken(ctx, response.RefreshToken); err != nil {
		return auth.Credential{}, err
	}
	return auth.Credential{
		AccessToken: response.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
	}, nil
}

func (c *authorizerCredential) acceptRefreshToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return c.persistPendingRefreshToken(ctx)
	}

	c.mu.Lock()
	if refreshToken != c.currentRefreshToken {
		c.currentRefreshToken = refreshToken
	}
	if c.client.refreshTokens == nil {
		c.persistedRefreshToken = refreshToken
		c.pendingRefreshToken = ""
	} else if refreshToken != c.persistedRefreshToken {
		c.pendingRefreshToken = refreshToken
	}
	c.mu.Unlock()

	return c.persistPendingRefreshToken(ctx)
}

func (c *authorizerCredential) persistPendingRefreshToken(ctx context.Context) error {
	store := c.client.refreshTokens
	if store == nil {
		return nil
	}

	for {
		c.mu.RLock()
		pending := c.pendingRefreshToken
		c.mu.RUnlock()
		if pending == "" {
			return nil
		}

		if err := store.SaveRefreshToken(ctx, c.appID, pending); err != nil {
			return err
		}

		c.mu.Lock()
		if c.pendingRefreshToken == pending {
			c.persistedRefreshToken = pending
			c.pendingRefreshToken = ""
		}
		morePending := c.pendingRefreshToken != ""
		c.mu.Unlock()
		if !morePending {
			return nil
		}
	}
}
