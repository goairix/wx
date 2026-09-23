// Package miniapp provides enterprise mini program APIs.
package miniapp

import (
	"context"
	"net/url"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides enterprise mini program login APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client {
	return &Client{
		api: executor,
	}
}

// Session is the result of exchanging a mini program login code.
type Session struct {
	CorpID     string `json:"corpid"`
	UserID     string `json:"userid"`
	SessionKey string `json:"session_key"`
}

// Session exchanges a mini program login code for a session.
func (c *Client) Session(ctx context.Context, code string) (*Session, error) {
	result := new(Session)
	err := c.api.Get(
		ctx,
		"work.miniapp.session",
		"cgi-bin/miniprogram/jscode2session",
		url.Values{
			"js_code":    []string{code},
			"grant_type": []string{"authorization_code"},
		},
		result,
	)
	return result, err
}
