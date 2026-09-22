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

	"github.com/goairix/wx/health_card/anti_fraud"
	"github.com/goairix/wx/health_card/card"
	"github.com/goairix/wx/health_card/device"
	"github.com/goairix/wx/health_card/notification"
	"github.com/goairix/wx/health_card/patient"
	"github.com/goairix/wx/health_card/usage"
	"github.com/goairix/wx/health_card/verification"
	kernelContracts "github.com/goairix/wx/kernel/contracts"
	"github.com/goairix/wx/support/cache"
	"github.com/goairix/wx/support/lock"
)

const defaultBaseURL = "https://p-healthopen.tengmed.com"
const getAppTokenPath = "/rest/auth/HealthCard/HealthOpenAuth/AuthObj/getAppToken"

// Client 是腾讯电子健康卡开放平台的根客户端。
// 它负责公共参数组装、请求签名、appToken 管理、HTTP 调用和统一错误转换，
// 业务接口通过 Card、Patient 等领域入口访问。
type Client struct {
	appID               string
	appSecret           string
	initialAppToken     string
	hospitalID          string
	baseURL             string
	channelNum          int
	relateAppID         string
	relateOpenID        string
	httpClient          *http.Client
	now                 func() time.Time
	requestID           func() string
	tokenProvider       kernelContracts.AccessTokenProvider
	tokenCache          cache.Cache
	tokenCacheKeyPrefix string
	tokenLocker         lock.Locker
}

// Card 返回健康卡注册、查询和展码领域客户端。
func (client *Client) Card() *card.Client { return card.New(client) }

// Patient 返回建档和实名就诊人领域客户端。
func (client *Client) Patient() *patient.Client { return patient.New(client) }

// Verification 返回人脸及实人认证领域客户端。
func (client *Client) Verification() *verification.Client { return verification.New(client) }

// Usage 返回用卡数据上报领域客户端。
func (client *Client) Usage() *usage.Client { return usage.New(client) }

// Device 返回自助机扫码设备领域客户端。
func (client *Client) Device() *device.Client { return device.New(client) }

// Notification 返回平台通知领域客户端。
func (client *Client) Notification() *notification.Client { return notification.New(client) }

// AntiFraud 返回预约防黄牛领域客户端。
func (client *Client) AntiFraud() *anti_fraud.Client { return anti_fraud.New(client) }

// Option 用于配置根客户端。
type Option func(*Client)

// WithAccessTokenProvider 注入外部 appToken 提供器。设置后不再请求腾讯的凭证接口。
func WithAccessTokenProvider(provider kernelContracts.AccessTokenProvider) Option {
	return func(client *Client) { client.tokenProvider = provider }
}

// WithTokenProvider 是 WithAccessTokenProvider 的简写别名。
func WithTokenProvider(provider kernelContracts.AccessTokenProvider) Option {
	return WithAccessTokenProvider(provider)
}

// WithBaseURL 覆盖腾讯生产环境地址，通常用于测试环境或 httptest.Server。
func WithBaseURL(baseURL string) Option {
	return func(client *Client) { client.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient 设置底层 HTTP 客户端，可用于设置超时、代理和自定义传输层。
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// WithChannelNum 设置腾讯平台要求的渠道编号。
func WithChannelNum(channelNum int) Option {
	return func(client *Client) { client.channelNum = channelNum }
}

// WithAppToken 注入已经获取的 appToken。
// 未注入时，客户端会通过 getAppToken 接口自动获取并在内存中缓存。
func WithAppToken(appToken string) Option {
	return func(client *Client) { client.initialAppToken = appToken }
}

// WithRelatedAppID 覆盖关联的小程序或公众号 AppID。
func WithRelatedAppID(appID string) Option {
	return func(client *Client) { client.relateAppID = appID }
}

// WithRelatedOpenID 设置关联的微信用户 OpenID（可选）。
func WithRelatedOpenID(openID string) Option {
	return func(client *Client) { client.relateOpenID = openID }
}

// WithClock 替换生成请求时间戳的时钟，主要用于测试。
func WithClock(now func() time.Time) Option {
	return func(client *Client) {
		if now != nil {
			client.now = now
		}
	}
}

// WithRequestID 替换请求 ID 生成函数，主要用于测试和链路追踪。
func WithRequestID(requestID func() string) Option {
	return func(client *Client) {
		if requestID != nil {
			client.requestID = requestID
		}
	}
}

// New 创建腾讯电子健康卡根客户端。
// relateAppID 是实际承载健康卡流程的小程序或公众号 AppID，应与前端获取 wechatCode 的应用一致。
func New(appID, appSecret, hospitalID, relateAppID string, opts ...Option) *Client {
	client := &Client{
		appID:               appID,
		appSecret:           appSecret,
		hospitalID:          hospitalID,
		relateAppID:         relateAppID,
		baseURL:             defaultBaseURL,
		channelNum:          0,
		httpClient:          http.DefaultClient,
		now:                 time.Now,
		requestID:           newRequestID,
		tokenCacheKeyPrefix: cache.DefaultCacheKeyPrefix,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	if client.tokenCache == nil {
		client.tokenCache = cache.NewMemoryCache()
	}
	if client.tokenLocker == nil {
		client.tokenLocker = &lock.Mutex{}
	}
	if client.initialAppToken != "" {
		// WithAppToken 表示调用方已经确认该 token 可用，因此按不过期凭证预置到缓存。
		_ = client.writeCachedAppToken(client.initialAppToken, 0)
	}
	return client
}

// AppToken 返回当前可用的 appToken。
// 配置外部 TokenProvider 时由外部提供；否则在本地缓存有效时直接返回，
// 缓存失效后自动请求腾讯 getAppToken 接口。
func (client *Client) AppToken() (string, error) {
	if client.tokenProvider != nil {
		token, err := client.tokenProvider.GetAccessToken()
		if err != nil {
			return "", err
		}
		if token.AccessToken == "" {
			return "", fmt.Errorf("health card app token response is empty")
		}
		return token.AccessToken, nil
	}
	if token, ok := client.readCachedAppToken(); ok {
		return token, nil
	}
	client.tokenLocker.Lock()
	defer client.tokenLocker.Unlock()

	if token, ok := client.readCachedAppToken(); ok {
		return token, nil
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
	if err := client.writeCachedAppToken(result.AppToken, result.ExpiresIn); err != nil {
		return "", err
	}
	return result.AppToken, nil
}

func newRequestID() string {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return strings.ToUpper(hex.EncodeToString([]byte(time.Now().String())))
	}
	return strings.ToUpper(hex.EncodeToString(data))
}

func (client *Client) do(path string, req interface{}, result interface{}) error {
	appToken := ""
	if path != getAppTokenPath {
		var err error
		appToken, err = client.AppToken()
		if err != nil {
			return err
		}
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
