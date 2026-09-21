package health_card

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

func (client *Client) do(path string, req interface{}, result interface{}) error {
	commonIn := CommonIn{
		AppToken:   client.appToken,
		RequestID:  client.requestID(),
		HospitalID: client.hospitalID,
		Timestamp:  fmt.Sprintf("%d", client.now().Unix()),
		ChannelNum: client.channelNum,
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
