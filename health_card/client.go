package health_card

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://p-healthopen.tengmed.com"
const getAppTokenPath = "/rest/auth/HealthCard/HealthOpenAuth/AuthObj/getAppToken"

// Client calls the Tencent Electronic Health Card Open Platform.
type Client struct {
	appID        string
	appSecret    string
	appToken     string
	tokenUntil   time.Time
	hospitalID   string
	baseURL      string
	channelNum   int
	relateAppID  string
	relateOpenID string
	httpClient   *http.Client
	now          func() time.Time
	requestID    func() string
	tokenMu      sync.Mutex
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

// WithAppToken supplies an already fetched token. When omitted, the client
// obtains and caches a token through the platform getAppToken API.
func WithAppToken(appToken string) Option {
	return func(client *Client) { client.appToken = appToken }
}

// WithRelatedAppID sets the optional related mini-program or service account ID.
func WithRelatedAppID(appID string) Option {
	return func(client *Client) { client.relateAppID = appID }
}

// WithRelatedOpenID sets the optional related WeChat user open ID.
func WithRelatedOpenID(openID string) Option {
	return func(client *Client) { client.relateOpenID = openID }
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
func New(appID, appSecret, hospitalID string, opts ...Option) *Client {
	client := &Client{
		appID:      appID,
		appSecret:  appSecret,
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

// AppToken returns a cached platform token or fetches a new one when needed.
func (client *Client) AppToken() (string, error) {
	client.tokenMu.Lock()
	defer client.tokenMu.Unlock()

	if client.appToken != "" && (client.tokenUntil.IsZero() || client.now().Before(client.tokenUntil)) {
		return client.appToken, nil
	}

	var result AppTokenResponse
	if err := client.do(getAppTokenPath, struct {
		AppID string `json:"appId"`
	}{AppID: client.appID}, &result); err != nil {
		return "", err
	}
	if result.AppToken == "" {
		return "", fmt.Errorf("health card app token response is empty")
	}
	client.appToken = result.AppToken
	if result.ExpiresIn > 0 {
		refreshAfter := result.ExpiresIn - 60
		if refreshAfter <= 0 {
			refreshAfter = result.ExpiresIn
		}
		client.tokenUntil = client.now().Add(time.Duration(refreshAfter) * time.Second)
	}
	return client.appToken, nil
}

func newRequestID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return strings.ToUpper(hex.EncodeToString([]byte(time.Now().String())))
	}
	return strings.ToUpper(hex.EncodeToString(data))
}

func (client *Client) do(path string, req interface{}, result interface{}) error {
	if path != getAppTokenPath {
		if _, err := client.AppToken(); err != nil {
			return err
		}
	}
	appToken := client.appToken
	if path == getAppTokenPath {
		appToken = ""
	}
	commonIn := CommonIn{
		AppToken:     appToken,
		RequestID:    client.requestID(),
		HospitalID:   client.hospitalID,
		Timestamp:    fmt.Sprintf("%d", client.now().Unix()),
		ChannelNum:   client.channelNum,
		RelateAppID:  client.relateAppID,
		RelateOpenID: client.relateOpenID,
	}

	requestValues, err := structMap(req)
	if err != nil {
		return err
	}
	commonValues, err := structMap(commonIn)
	if err != nil {
		return err
	}
	signValues := make(map[string]interface{}, len(requestValues)+len(commonValues))
	for key, value := range commonValues {
		signValues[key] = value
	}
	for key, value := range requestValues {
		signValues[key] = value
	}
	commonIn.Sign = sign(signValues, client.appSecret)

	body := requestEnvelope{CommonIn: commonIn, Req: req}
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, client.baseURL+path, strings.NewReader(string(encoded)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json;charset=utf-8")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("health card http error: status_code=%d", response.StatusCode)
	}

	var envelope struct {
		CommonOut CommonOut       `json:"commonOut"`
		Rsp       json.RawMessage `json:"rsp"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return err
	}
	if envelope.CommonOut.ResultCode != 0 {
		return &APIError{
			RequestID: envelope.CommonOut.RequestID,
			Code:      envelope.CommonOut.ResultCode,
			Message:   envelope.CommonOut.ErrMsg,
		}
	}
	if result == nil || len(envelope.Rsp) == 0 || string(envelope.Rsp) == "null" {
		return nil
	}
	return json.Unmarshal(envelope.Rsp, result)
}

func structMap(value interface{}) (map[string]interface{}, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var values map[string]interface{}
	if err := json.Unmarshal(encoded, &values); err != nil {
		return nil, err
	}
	return values, nil
}
