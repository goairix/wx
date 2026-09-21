package health_card

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://p-healthopen.tengmed.com"

// Client calls the Tencent Electronic Health Card Open Platform.
type Client struct {
	appSecret  string
	appToken   string
	hospitalID string
	baseURL    string
	channelNum int
	httpClient *http.Client
	now        func() time.Time
	requestID  func() string
}

// Option customizes a Client.
type Option func(*Client)

// WithBaseURL overrides the production endpoint, primarily for testing.
func WithBaseURL(baseURL string) Option {
	return func(client *Client) { client.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient sets the HTTP client used for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// WithChannelNum sets the platform channel number.
func WithChannelNum(channelNum int) Option {
	return func(client *Client) { client.channelNum = channelNum }
}

// WithClock replaces the clock used to create request timestamps.
func WithClock(now func() time.Time) Option {
	return func(client *Client) {
		if now != nil {
			client.now = now
		}
	}
}

// WithRequestID replaces request ID generation.
func WithRequestID(requestID func() string) Option {
	return func(client *Client) {
		if requestID != nil {
			client.requestID = requestID
		}
	}
}

// New creates a Tencent Electronic Health Card client.
func New(appSecret, appToken, hospitalID string, opts ...Option) *Client {
	client := &Client{
		appSecret:  appSecret,
		appToken:   appToken,
		hospitalID: hospitalID,
		baseURL:    defaultBaseURL,
		channelNum: 0,
		httpClient: http.DefaultClient,
		now:        time.Now,
		requestID:  newRequestID,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

func newRequestID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return strings.ToUpper(hex.EncodeToString([]byte(time.Now().String())))
	}
	return strings.ToUpper(hex.EncodeToString(data))
}
