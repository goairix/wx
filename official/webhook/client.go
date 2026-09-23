// Package webhook provides typed official account callback handling.
package webhook

import (
	"context"
	"net/http"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Event contains common official account message and event fields.
type Event struct {
	ToUserName   string  `xml:"ToUserName" json:"ToUserName"`
	FromUserName string  `xml:"FromUserName" json:"FromUserName"`
	CreateTime   int64   `xml:"CreateTime" json:"CreateTime"`
	MessageType  string  `xml:"MsgType" json:"MsgType"`
	Event        string  `xml:"Event" json:"Event"`
	EventKey     string  `xml:"EventKey" json:"EventKey"`
	Content      string  `xml:"Content" json:"Content"`
	Ticket       string  `xml:"Ticket" json:"Ticket"`
	Latitude     float64 `xml:"Latitude" json:"Latitude"`
	Longitude    float64 `xml:"Longitude" json:"Longitude"`
	Precision    float64 `xml:"Precision" json:"Precision"`
}

// Handler processes a typed official account event.
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

// Client stores official account callback credentials.
type Client struct {
	config corewebhook.CallbackConfig
}

// NewClient constructs an official account callback adapter.
func NewClient(appID, token, encodingAESKey string) *Client {
	return &Client{config: corewebhook.CallbackConfig{
		ReceiverID:     appID,
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
