// Package healthcard provides a context-aware Tencent electronic health card client.
package healthcard

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/healthcard/antifraud"
	"github.com/goairix/wx/v2/healthcard/card"
	"github.com/goairix/wx/v2/healthcard/device"
	"github.com/goairix/wx/v2/healthcard/notification"
	"github.com/goairix/wx/v2/healthcard/patient"
	"github.com/goairix/wx/v2/healthcard/usage"
	"github.com/goairix/wx/v2/healthcard/verification"
)

const defaultBaseURL = "https://p-healthopen.tengmed.com"
const getAppTokenPath = "/rest/auth/HealthCard/HealthOpenAuth/AuthObj/getAppToken"

// Client is the root client for the Tencent electronic health card platform.
type Client struct {
	config       Config
	transport    *transport.Client
	auth         *auth.Manager
	channelNum   int
	now          func() time.Time
	requestID    func() string
	card         *card.Client
	patient      *patient.Client
	verification *verification.Client
	usage        *usage.Client
	device       *device.Client
	notification *notification.Client
	antifraud    *antifraud.Client
}

// NewClient constructs a health card client without making network calls.
func NewClient(config Config, options ...Option) (*Client, error) {
	if strings.TrimSpace(config.AppID) == "" {
		return nil, fmt.Errorf("healthcard: AppID is required")
	}
	if strings.TrimSpace(config.AppSecret) == "" {
		return nil, fmt.Errorf("healthcard: AppSecret is required")
	}
	if strings.TrimSpace(config.HospitalID) == "" {
		return nil, fmt.Errorf("healthcard: HospitalID is required")
	}

	settings := new(option)
	for _, configure := range options {
		if configure != nil {
			configure(settings)
		}
	}
	if settings.relatedAppID != "" {
		config.RelatedAppID = settings.relatedAppID
	}

	transportClient := settings.transport
	if transportClient == nil {
		baseURL := settings.baseURL
		if baseURL == "" {
			baseURL = defaultBaseURL
		}
		transportClient = transport.New(
			settings.httpClient,
			baseURL,
			settings.retry,
			transport.WithHook(settings.hook),
			transport.WithLogger(settings.logger),
		)
	}

	client := &Client{
		config:     config,
		transport:  transportClient,
		channelNum: settings.channelNum,
		now:        settings.now,
		requestID:  settings.requestID,
	}
	if client.now == nil {
		client.now = time.Now
	}
	if client.requestID == nil {
		client.requestID = newRequestID
	}

	credentialCache := settings.cache
	if credentialCache == nil {
		credentialCache = corecache.NewMemory()
	}
	client.auth = settings.credentialManager
	if client.auth == nil {
		provider := settings.credentialProvider
		if provider == nil && settings.initialAppToken != "" {
			provider = seededTokenProvider(settings.initialAppToken)
		}
		if provider == nil {
			provider = auth.ProviderFunc(client.fetchAppToken)
		}
		client.auth = auth.NewManager(
			"healthcard",
			credentialIdentity(config.AppID, config.AppSecret),
			credentialCache,
			provider,
		)
	}

	client.card = card.New(client)
	client.patient = patient.New(client)
	client.verification = verification.New(client)
	client.usage = usage.New(client)
	client.device = device.New(client)
	client.notification = notification.New(client)
	client.antifraud = antifraud.New(client)
	return client, nil
}

func credentialIdentity(appID, appSecret string) string {
	digest := sha256.Sum256([]byte(appID + "\x00" + appSecret))
	return appID + ":" + hex.EncodeToString(digest[:])
}

func seededTokenProvider(appToken string) auth.Provider {
	return auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{
			AccessToken: appToken,
			ExpiresAt:   time.Now().AddDate(100, 0, 0),
		}, nil
	})
}

func (client *Client) fetchAppToken(ctx context.Context) (auth.Credential, error) {
	var result AppTokenResponse
	err := client.do(
		ctx,
		getAppTokenPath,
		struct {
			AppID string `json:"appId"`
		}{AppID: client.config.AppID},
		&result,
		false,
		"",
		false,
	)
	if err != nil {
		return auth.Credential{}, err
	}
	if result.AppToken == "" {
		return auth.Credential{}, &wxerrors.Error{
			Platform:  "healthcard",
			Operation: getAppTokenPath,
			Message:   "app token response is empty",
		}
	}

	credential := auth.Credential{AccessToken: result.AppToken}
	if result.ExpiresIn > 0 {
		credential.ExpiresAt = time.Now().Add(
			time.Duration(result.ExpiresIn) * time.Second,
		)
	}
	return credential, nil
}

// AppToken returns a cached appToken or obtains a fresh one.
func (client *Client) AppToken(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	credential, err := client.auth.Token(ctx)
	if err != nil {
		return "", client.wrapError(getAppTokenPath, err)
	}
	if credential.AccessToken == "" {
		return "", &wxerrors.Error{
			Platform:  "healthcard",
			Operation: getAppTokenPath,
			Message:   "app token response is empty",
		}
	}
	return credential.AccessToken, nil
}

// Config returns a copy of the client configuration.
func (client *Client) Config() Config {
	return client.config
}

// Card returns health card registration, query, and QR code operations.
func (client *Client) Card() *card.Client {
	return client.card
}

// Patient returns patient record operations.
func (client *Client) Patient() *patient.Client {
	return client.patient
}

// Verification returns identity verification operations.
func (client *Client) Verification() *verification.Client {
	return client.verification
}

// Usage returns health card usage reporting operations.
func (client *Client) Usage() *usage.Client {
	return client.usage
}

// Device returns self-service authorization device operations.
func (client *Client) Device() *device.Client {
	return client.device
}

// Notification returns platform notification operations.
func (client *Client) Notification() *notification.Client {
	return client.notification
}

// AntiFraud returns appointment anti-fraud operations.
func (client *Client) AntiFraud() *antifraud.Client {
	return client.antifraud
}

func newRequestID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return strings.ToUpper(hex.EncodeToString([]byte(time.Now().String())))
	}
	return strings.ToUpper(hex.EncodeToString(data))
}
