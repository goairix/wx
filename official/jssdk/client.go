// Package jssdk builds signed official account JS SDK configurations.
package jssdk

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/random"
	"github.com/goairix/wx/v2/official/internal/api"
)

// Client builds JS SDK signatures and manages jsapi tickets.
type Client struct {
	appID   string
	tickets *auth.Manager
	now     func() time.Time
	nonce   func(int) (string, error)
}

// NewClient constructs a JS SDK client.
func NewClient(executor *api.Client, appID string, store cache.Cache) *Client {
	return NewWithCaller(executor, appID, store)
}

// NewWithCaller constructs a domain client with an authenticated caller.
func NewWithCaller(executor Caller, appID string, store cache.Cache) *Client {
	client := &Client{
		appID: appID,
		now:   time.Now,
		nonce: random.String,
	}
	client.tickets = auth.NewManager(
		"official.jssdk",
		appID+":jsapi",
		store,
		auth.ProviderFunc(func(ctx context.Context) (auth.Credential, error) {
			var result struct {
				Ticket    string `json:"ticket"`
				ExpiresIn int64  `json:"expires_in"`
			}
			err := executor.Get(
				ctx,
				"official.jssdk.ticket",
				"cgi-bin/ticket/getticket",
				mapValues("type", "jsapi"),
				&result,
			)
			if err != nil {
				return auth.Credential{}, err
			}
			return auth.Credential{
				AccessToken: result.Ticket,
				ExpiresAt: time.Now().Add(
					time.Duration(result.ExpiresIn) * time.Second,
				),
			}, nil
		}),
	)
	return client
}

// BuildConfig returns a signed configuration for one exact page URL.
func (c *Client) BuildConfig(
	ctx context.Context,
	pageURL string,
	jsAPIList []string,
	debug bool,
	beta bool,
) (map[string]interface{}, error) {
	ticket, err := c.tickets.Token(ctx)
	if err != nil {
		return nil, err
	}
	nonce, err := c.nonce(10)
	if err != nil {
		return nil, err
	}
	timestamp := c.now().Unix()
	return map[string]interface{}{
		"debug":     debug,
		"beta":      beta,
		"jsApiList": jsAPIList,
		"appId":     c.appID,
		"nonceStr":  nonce,
		"timestamp": timestamp,
		"url":       pageURL,
		"signature": signature(ticket.AccessToken, nonce, timestamp, pageURL),
	}, nil
}

func signature(ticket, nonce string, timestamp int64, pageURL string) string {
	value := fmt.Sprintf(
		"jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s",
		ticket,
		nonce,
		timestamp,
		pageURL,
	)
	digest := sha1.Sum([]byte(value))
	return hex.EncodeToString(digest[:])
}
