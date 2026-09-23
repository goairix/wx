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
	client       *Client
	appID        string
	mu           sync.RWMutex
	refreshToken string
	manager      *auth.Manager
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
	identity := credentialIdentity("authorizer", c.config.AppID, appID, refreshToken)

	c.mu.Lock()
	defer c.mu.Unlock()
	if credential := c.credentials[identity]; credential != nil {
		return credential.manager
	}

	credential := &authorizerCredential{
		client:       c,
		appID:        appID,
		refreshToken: refreshToken,
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
	componentCredential, err := c.client.component.Token(ctx)
	if err != nil {
		return auth.Credential{}, err
	}

	c.mu.RLock()
	refreshToken := c.refreshToken
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
	if err := c.saveRotatedRefreshToken(ctx, response.RefreshToken); err != nil {
		return auth.Credential{}, err
	}
	return auth.Credential{
		AccessToken: response.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
	}, nil
}

func (c *authorizerCredential) saveRotatedRefreshToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	c.mu.Lock()
	if refreshToken == c.refreshToken {
		c.mu.Unlock()
		return nil
	}
	c.refreshToken = refreshToken
	c.mu.Unlock()

	if c.client.refreshTokens == nil {
		return nil
	}
	return c.client.refreshTokens.SaveRefreshToken(ctx, c.appID, refreshToken)
}
