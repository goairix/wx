package openplatform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

func credentialIdentity(kind string, values ...string) string {
	digest := sha256.New()
	for _, value := range values {
		_, _ = digest.Write([]byte(value))
		_, _ = digest.Write([]byte{0})
	}
	return kind + ":" + hex.EncodeToString(digest.Sum(nil))
}

func (c *Client) authorizerProvider(appID, refreshToken string) auth.Provider {
	return auth.ProviderFunc(func(ctx context.Context) (auth.Credential, error) {
		componentCredential, err := c.component.Token(ctx)
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
		err = c.transport.Do(ctx, request.Request{
			Operation: "openplatform.authorizer.token",
			Platform:  "openplatform",
			Method:    http.MethodPost,
			Path:      "cgi-bin/component/api_authorizer_token",
			Query: mapValues(
				"component_access_token",
				componentCredential.AccessToken,
			),
			Body: struct {
				ComponentAppID  string `json:"component_appid"`
				AuthorizerAppID string `json:"authorizer_appid"`
				RefreshToken    string `json:"authorizer_refresh_token"`
			}{
				ComponentAppID:  c.config.AppID,
				AuthorizerAppID: appID,
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
		return auth.Credential{
			AccessToken: response.AccessToken,
			ExpiresAt:   time.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
		}, nil
	})
}

func (c *Client) authorizerManager(appID, refreshToken string) *auth.Manager {
	identity := credentialIdentity("authorizer", c.config.AppID, appID, refreshToken)
	return auth.NewManager(
		"openplatform-authorizer",
		identity,
		c.cache,
		c.authorizerProvider(appID, refreshToken),
	)
}
