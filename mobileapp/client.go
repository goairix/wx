package mobileapp

import (
	"fmt"
	"strings"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/mobileapp/oauth"
)

const defaultBaseURL = "https://api.weixin.qq.com"

type Client struct {
	config    Config
	transport *transport.Client
	oauth     *oauth.Client
}

func NewClient(config Config, opts ...Option) (*Client, error) {
	if strings.TrimSpace(config.AppID) == "" || strings.TrimSpace(config.AppSecret) == "" {
		return nil, fmt.Errorf("mobileapp: AppID and AppSecret are required")
	}
	s := &option{}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	base := s.baseURL
	if base == "" {
		base = defaultBaseURL
	}
	tr := transport.New(
		s.httpClient,
		base,
		s.retry,
		transport.WithHook(s.hook),
		transport.WithLogger(s.logger),
	)
	c := s.cache
	if c == nil {
		c = corecache.NewMemory()
	}
	oauthClient := oauth.New(tr, oauth.Config{
		AppID:     config.AppID,
		AppSecret: config.AppSecret,
	}, c)
	return &Client{
		config:    config,
		transport: tr,
		oauth:     oauthClient,
	}, nil
}

func (c *Client) Config() Config {
	return c.config
}

func (c *Client) OAuth() *oauth.Client {
	return c.oauth
}
