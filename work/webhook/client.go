// Package webhook contains typed enterprise WeChat callback handling.
package webhook

import (
	"context"
	"net/http"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Client stores enterprise callback credentials.
type Client struct {
	corpID         string
	token          string
	encodingAESKey string
}

// NewClient constructs an enterprise callback adapter.
func NewClient(corpID, token, encodingAESKey string) *Client {
	return &Client{
		corpID:         corpID,
		token:          token,
		encodingAESKey: encodingAESKey,
	}
}

// Handler processes a typed enterprise callback event.
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

type handlerOptions struct {
	errorResponse corewebhook.ErrorResponse
}

// Option configures the enterprise callback adapter.
type Option func(*handlerOptions)

// WithErrorResponse configures callback error responses.
func WithErrorResponse(policy corewebhook.ErrorResponse) Option {
	return func(options *handlerOptions) {
		options.errorResponse = policy
	}
}

// Handler returns an HTTP adapter that verifies and decodes callbacks.
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
			event.Raw = append([]byte(nil), payload.Raw...)
			event.Format = payload.Format
			return next.Handle(ctx, event)
		}),
		options...,
	)
}

// RawHandler returns an HTTP adapter that retains the generic core payload.
func (c *Client) RawHandler(
	next corewebhook.Handler,
	options ...Option,
) http.Handler {
	settings := new(handlerOptions)
	for _, configure := range options {
		if configure != nil {
			configure(settings)
		}
	}
	coreOptions := make([]corewebhook.Option, 0, 1)
	if settings.errorResponse != nil {
		coreOptions = append(
			coreOptions,
			corewebhook.WithErrorResponse(settings.errorResponse),
		)
	}
	return corewebhook.NewCallbackHandler(
		corewebhook.CallbackConfig{
			ReceiverID:     c.corpID,
			Token:          c.token,
			EncodingAESKey: c.encodingAESKey,
		},
		next,
		coreOptions...,
	)
}
