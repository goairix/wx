package miniapp

import (
	"context"

	"github.com/goairix/wx/v2/core/auth"
	wxauth "github.com/goairix/wx/v2/miniapp/auth"
)

func (c *Client) fetchToken(ctx context.Context) (auth.Credential, error) {
	token, expires, err := wxauth.FetchToken(ctx, c.transport, wxauth.Config{
		AppID:     c.config.AppID,
		AppSecret: c.config.AppSecret,
	})
	if err != nil {
		return auth.Credential{}, err
	}
	return auth.Credential{AccessToken: token, ExpiresAt: expires}, nil
}
