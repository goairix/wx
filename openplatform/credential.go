package openplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

type authorizerCredential struct {
	client                *Client
	appID                 string
	gate                  chan struct{}
	currentRefreshToken   string
	persistedRefreshToken string
	pendingRefreshToken   string
	pendingDelete         bool
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
	return c.authorizerCredential(appID, refreshToken).manager
}

func (c *Client) authorizerCredential(appID, refreshToken string) *authorizerCredential {
	identity := credentialIdentity("authorizer", c.config.AppID, appID)

	c.mu.Lock()
	defer c.mu.Unlock()
	if credential := c.credentials[identity]; credential != nil {
		return credential
	}

	credential := &authorizerCredential{
		client:                c,
		gate:                  make(chan struct{}, 1),
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
	return credential
}

func (c *authorizerCredential) refresh(ctx context.Context) (auth.Credential, error) {
	if err := c.acquire(ctx); err != nil {
		return auth.Credential{}, err
	}
	defer c.release()
	if err := c.reconcileRepository(ctx); err != nil {
		return auth.Credential{}, err
	}
	if c.pendingDelete {
		if err := c.deleteRefreshToken(ctx); err != nil {
			return auth.Credential{}, err
		}
	}
	if err := c.persistPendingRefreshToken(ctx); err != nil {
		return auth.Credential{}, err
	}

	refreshToken := c.currentRefreshToken
	if refreshToken == "" {
		return auth.Credential{}, fmt.Errorf("openplatform authorizer: account is not authorized")
	}

	componentCredential, err := c.client.component.Token(ctx)
	if err != nil {
		return auth.Credential{}, err
	}

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

	if refreshToken != c.currentRefreshToken {
		c.currentRefreshToken = refreshToken
	}
	if c.client.refreshTokens == nil {
		c.persistedRefreshToken = refreshToken
		c.pendingRefreshToken = ""
	} else if refreshToken != c.persistedRefreshToken {
		c.pendingRefreshToken = refreshToken
	}

	return c.persistPendingRefreshToken(ctx)
}

// Helpers below require the credential gate; storage calls and refreshes share the same lock.
// reconcileRepository discards pending writes superseded by a later repository
// state. A save may have committed even if its acknowledgement returned an error.
func (c *authorizerCredential) reconcileRepository(ctx context.Context) error {
	repository := c.client.refreshRepository
	if repository == nil {
		return nil
	}
	token, err := repository.LoadRefreshToken(ctx, c.appID)
	if err != nil {
		return err
	}
	changed := token != c.persistedRefreshToken
	if c.pendingDelete && (changed || token == "") {
		c.pendingDelete = false
	}
	if c.pendingRefreshToken != "" && (changed || token == c.pendingRefreshToken) {
		c.pendingRefreshToken = ""
	}
	c.persistedRefreshToken = token
	if !c.pendingDelete && c.pendingRefreshToken == "" {
		c.currentRefreshToken = token
	}
	return nil
}

// rememberPersistedToken records the baseline for an explicit mutation so a
// failed write can later be distinguished from a newer authorization elsewhere.
func (c *authorizerCredential) rememberPersistedToken(ctx context.Context) error {
	if repository := c.client.refreshRepository; repository != nil {
		token, err := repository.LoadRefreshToken(ctx, c.appID)
		if err != nil {
			return err
		}
		c.persistedRefreshToken = token
	}
	return nil
}

func (c *authorizerCredential) persistPendingRefreshToken(ctx context.Context) error {
	if c.client.refreshTokens == nil || c.pendingRefreshToken == "" {
		return nil
	}
	if err := c.client.refreshTokens.SaveRefreshToken(ctx, c.appID, c.pendingRefreshToken); err != nil {
		return err
	}
	c.persistedRefreshToken = c.pendingRefreshToken
	c.pendingRefreshToken = ""
	return nil
}

func (c *authorizerCredential) deleteRefreshToken(ctx context.Context) error {
	if repository := c.client.refreshRepository; repository != nil {
		if err := repository.DeleteRefreshToken(ctx, c.appID); err != nil {
			return err
		}
	}
	c.pendingDelete = false
	c.persistedRefreshToken = ""
	return nil
}

// UpdateAuthorizer installs a new authorization for existing and future child
// clients. A failed save remains pending and is retried before refreshing.
func (c *Client) UpdateAuthorizer(ctx context.Context, appID, refreshToken string) error {
	if strings.TrimSpace(appID) == "" || strings.TrimSpace(refreshToken) == "" {
		return fmt.Errorf("openplatform: authorizer AppID and refresh token are required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := c.authorizerCredential(appID, "")
	if err := credential.acquire(ctx); err != nil {
		return err
	}
	if err := credential.rememberPersistedToken(ctx); err != nil {
		credential.release()
		return err
	}
	credential.currentRefreshToken = refreshToken
	credential.pendingDelete = false
	credential.pendingRefreshToken = ""
	if c.refreshTokens != nil {
		credential.pendingRefreshToken = refreshToken
	} else {
		credential.persistedRefreshToken = refreshToken
	}
	err := credential.persistPendingRefreshToken(ctx)
	credential.release()
	// The manager coordinates provider execution, so never hold the credential gate
	// while invalidating: a provider may already be waiting for that mutex.
	return errors.Join(err, credential.manager.Invalidate(ctx, ""))
}

// RevokeAuthorizer removes authorization and invalidates existing child clients.
// Failed repository deletion remains pending and prevents further refreshes.
// With a save-only store, callers must also delete their persisted record.
func (c *Client) RevokeAuthorizer(ctx context.Context, appID string) error {
	if strings.TrimSpace(appID) == "" {
		return fmt.Errorf("openplatform: authorizer AppID is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	credential := c.authorizerCredential(appID, "")
	if err := credential.acquire(ctx); err != nil {
		return err
	}
	if err := credential.rememberPersistedToken(ctx); err != nil {
		credential.release()
		return err
	}
	credential.currentRefreshToken = ""
	credential.pendingRefreshToken = ""
	credential.pendingDelete = true
	err := credential.deleteRefreshToken(ctx)
	credential.release()
	return errors.Join(err, credential.manager.Invalidate(ctx, ""))
}

// acquire serializes credential state without borrowing another request's
// storage or network deadline while waiting for its refresh to finish.
func (c *authorizerCredential) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.gate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			c.release()
			return err
		}
		return nil
	}
}

func (c *authorizerCredential) release() { <-c.gate }
