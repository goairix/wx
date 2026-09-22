package miniapp

import (
	"net/http"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/transport"
)

// Config configures a WeChat miniapp client.
type Config struct {
	AppID          string
	AppSecret      string
	Token          string
	EncodingAESKey string
}

type option struct {
	httpClient *http.Client
	baseURL    string
	cache      corecache.Cache
	retry      transport.RetryPolicy
	hook       observability.Hook
}

// Option customizes a miniapp client.
type Option func(*option)

func WithHTTPClient(c *http.Client) Option {
	return func(o *option) {
		o.httpClient = c
	}
}
func WithBaseURL(v string) Option {
	return func(o *option) {
		o.baseURL = v
	}
}
func WithCache(c corecache.Cache) Option {
	return func(o *option) {
		o.cache = c
	}
}
func WithRetryPolicy(p transport.RetryPolicy) Option {
	return func(o *option) {
		o.retry = p
	}
}
func WithHook(h observability.Hook) Option {
	return func(o *option) {
		o.hook = h
	}
}
