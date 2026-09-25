package work

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/work/accountid"
	workauth "github.com/goairix/wx/v2/work/auth"
	"github.com/goairix/wx/v2/work/authorizer"
	"github.com/goairix/wx/v2/work/contact"
	"github.com/goairix/wx/v2/work/customer"
	"github.com/goairix/wx/v2/work/internal/api"
	"github.com/goairix/wx/v2/work/kf"
	"github.com/goairix/wx/v2/work/media"
	"github.com/goairix/wx/v2/work/message"
	"github.com/goairix/wx/v2/work/miniapp"
	workwebhook "github.com/goairix/wx/v2/work/webhook"
)

const defaultBaseURL = "https://qyapi.weixin.qq.com"

// Client is a context-aware enterprise WeChat client.
type Client struct {
	config     Config
	transport  *transport.Client
	auth       *auth.Manager
	login      *workauth.Client
	contact    *contact.Client
	customer   *customer.Client
	message    *message.Client
	kefu       *kf.Client
	media      *media.Client
	accountID  *accountid.Client
	miniApp    *miniapp.Client
	authorizer *authorizer.Client
	webhook    *workwebhook.Client
}

// NewClient constructs an enterprise WeChat client.
func NewClient(config Config, options ...Option) (*Client, error) {
	if strings.TrimSpace(config.CorpID) == "" || strings.TrimSpace(config.CorpSecret) == "" {
		return nil, fmt.Errorf("work: CorpID and CorpSecret are required")
	}

	settings := new(option)
	for _, configure := range options {
		if configure != nil {
			configure(settings)
		}
	}
	baseURL := settings.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	transportClient := transport.New(
		settings.httpClient,
		baseURL,
		settings.retry,
		transport.WithHook(settings.hook),
		transport.WithObserver(settings.observer),
		transport.WithLogger(settings.logger),
	)
	credentialCache := settings.coreCache
	if credentialCache == nil {
		credentialCache = corecache.NewMemory()
	}

	client := &Client{
		config:    config,
		transport: transportClient,
	}
	client.auth = auth.NewManager(
		"work",
		credentialIdentity(config.CorpID, config.CorpSecret),
		credentialCache,
		auth.ProviderFunc(client.fetchToken),
	)
	executor := api.New(transportClient, client.auth)
	client.login = workauth.NewClient(executor, config.CorpID, config.AgentID)
	client.contact = contact.NewClient(executor, config.EncodingAESKey)
	client.customer = customer.NewClient(executor)
	client.message = message.NewClient(executor, config.AgentID)
	client.kefu = kf.NewClient(executor)
	client.media = media.NewClient(executor)
	client.accountID = accountid.NewClient(executor)
	client.miniApp = miniapp.NewClient(executor)
	client.authorizer = authorizer.NewClient(transportClient)
	client.webhook = workwebhook.NewClient(config.CorpID, config.Token, config.EncodingAESKey)
	return client, nil
}

func credentialIdentity(corpID, corpSecret string) string {
	digest := sha256.Sum256([]byte(corpID + "\x00" + corpSecret))
	return corpID + ":" + hex.EncodeToString(digest[:])
}

func (c *Client) fetchToken(ctx context.Context) (auth.Credential, error) {
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	meta := new(request.ResponseMeta)
	err := c.transport.Do(ctx, request.Request{
		Operation: "work.auth.token",
		Platform:  "work",
		Method:    http.MethodGet,
		Path:      "cgi-bin/gettoken",
		Query: url.Values{
			"corpid":     []string{c.config.CorpID},
			"corpsecret": []string{c.config.CorpSecret},
		},
		Result: &result,
		Meta:   meta,
	})
	if err != nil {
		return auth.Credential{}, err
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return auth.Credential{}, &wxerrors.Error{
			Platform:   "work",
			Operation:  "work.auth.token",
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(result.ErrCode),
			Message:    result.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return auth.Credential{
		AccessToken: result.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return c.config
}

func (c *Client) Auth() *workauth.Client {
	return c.login
}

func (c *Client) Contact() *contact.Client {
	return c.contact
}

func (c *Client) Customer() *customer.Client {
	return c.customer
}

func (c *Client) Message() *message.Client {
	return c.message
}

func (c *Client) Kefu() *kf.Client {
	return c.kefu
}

func (c *Client) Media() *media.Client {
	return c.media
}

func (c *Client) AccountID() *accountid.Client {
	return c.accountID
}

func (c *Client) MiniApp() *miniapp.Client {
	return c.miniApp
}

func (c *Client) Authorizer() *authorizer.Client {
	return c.authorizer
}

func (c *Client) Webhook() *workwebhook.Client {
	return c.webhook
}
