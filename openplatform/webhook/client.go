// Package webhook provides typed Open Platform callback handling.
package webhook

import (
	"context"
	"net/http"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Event contains component verification and authorization callback fields.
type Event struct {
	AppID                        string `xml:"AppId" json:"AppId"`
	CreateTime                   int64  `xml:"CreateTime" json:"CreateTime"`
	InfoType                     string `xml:"InfoType" json:"InfoType"`
	ComponentVerifyTicket        string `xml:"ComponentVerifyTicket" json:"ComponentVerifyTicket"`
	AuthorizerAppID              string `xml:"AuthorizerAppid" json:"AuthorizerAppid"`
	AuthorizationCode            string `xml:"AuthorizationCode" json:"AuthorizationCode"`
	AuthorizationCodeExpiredTime int64  `xml:"AuthorizationCodeExpiredTime" json:"AuthorizationCodeExpiredTime"`
	PreAuthCode                  string `xml:"PreAuthCode" json:"PreAuthCode"`
}

// Handler processes a typed Open Platform event.
type Handler interface {
	Handle(context.Context, Event) (corewebhook.Response, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(context.Context, Event) (corewebhook.Response, error)

// Handle calls f with the decoded event.
func (f HandlerFunc) Handle(
	ctx context.Context,
	event Event,
) (corewebhook.Response, error) {
	return f(ctx, event)
}

// Option configures the callback HTTP adapter.
type Option = corewebhook.Option

// WithErrorResponse configures callback error responses.
func WithErrorResponse(policy corewebhook.ErrorResponse) Option {
	return corewebhook.WithErrorResponse(policy)
}

// Client stores component callback credentials.
type Client struct {
	config corewebhook.CallbackConfig
}

// NewClient constructs an Open Platform callback adapter.
func NewClient(componentAppID, token, encodingAESKey string) *Client {
	return &Client{config: corewebhook.CallbackConfig{
		ReceiverID:     componentAppID,
		Token:          token,
		EncodingAESKey: encodingAESKey,
	}}
}

// Handler returns an HTTP handler that verifies and decodes callbacks.
func (c *Client) Handler(next Handler, options ...Option) http.Handler {
	return c.RawHandler(
		corewebhook.HandlerFunc(func(
			ctx context.Context,
			payload corewebhook.Payload,
		) (corewebhook.Response, error) {
			var event Event
			if err := corewebhook.Decode(payload, &event); err != nil {
				return corewebhook.Response{}, err
			}
			return next.Handle(ctx, event)
		}),
		options...,
	)
}

// RawHandler returns an HTTP handler with the generic core payload.
func (c *Client) RawHandler(
	next corewebhook.Handler,
	options ...Option,
) http.Handler {
	return corewebhook.NewCallbackHandler(c.config, next, options...)
}
