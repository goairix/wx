package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/goairix/wx/v2/core/logging"
)

const DefaultWebSocketURL = "wss://openws.work.weixin.qq.com"

// Config contains bot credentials, independent of enterprise application tokens.
type Config struct {
	BotID                               string
	Secret                              string
	URL                                 string
	Dialer                              Dialer
	HeartbeatInterval, HeartbeatTimeout time.Duration
	Reconnect                           *ReconnectPolicy
	// InitialConnect optionally retries transient failures before first readiness.
	// nil preserves one initial attempt; its budget is independent of Reconnect.
	InitialConnect *ReconnectPolicy
	// Limiter is optional; nil leaves platform quota enforcement to the server.
	Limiter            Limiter
	NotificationBuffer int
	MaxFrameBytes      int64
	ChunkSize          int
}

// Connection reads/writes complete JSON messages. Methods must honor context;
// Close must promptly interrupt Read and Write and may run concurrently with them.
type Connection interface {
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
	Close() error
}

// Dialer establishes a connection; cancellation must interrupt dialing.
type Dialer interface {
	Dial(context.Context, string) (Connection, error)
}
type DialerFunc func(context.Context, string) (Connection, error)

func (f DialerFunc) Dial(ctx context.Context, url string) (Connection, error) { return f(ctx, url) }

// ReconnectPolicy configures either Config.InitialConnect or Config.Reconnect.
// MaxAttempts counts retries after the first attempt: -1 is unlimited, 0 disables.
type ReconnectPolicy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	MaxAttempts  int
}
type option func(*options)
type options struct {
	url                                                 string
	dialer                                              Dialer
	httpClient                                          *http.Client
	logger                                              logging.Logger
	heartbeatInterval, heartbeatTimeout, requestTimeout time.Duration
	reconnect                                           ReconnectPolicy
	initialConnect                                      *ReconnectPolicy
	limiter                                             Limiter
	notificationBuffer                                  int
	maxFrameBytes                                       int64
	downloadLimit, uploadLimit                          int64
	chunkSize                                           int
}

func defaultOptions() options {
	return options{url: DefaultWebSocketURL, httpClient: &http.Client{Timeout: 60 * time.Second}, logger: logging.Nop(), heartbeatInterval: 30 * time.Second, heartbeatTimeout: 10 * time.Second, requestTimeout: 10 * time.Second, reconnect: ReconnectPolicy{InitialDelay: time.Second, MaxDelay: 30 * time.Second, MaxAttempts: 10}, notificationBuffer: 1024, maxFrameBytes: 4 << 20, downloadLimit: 20 << 20, uploadLimit: 20 << 20, chunkSize: 512 << 10}
}
func withWebSocketURL(url string) option   { return func(o *options) { o.url = url } }
func withDialer(d Dialer) option           { return func(o *options) { o.dialer = d } }
func withHTTPClient(h *http.Client) option { return func(o *options) { o.httpClient = h } }
func withLogger(l logging.Logger) option   { return func(o *options) { o.logger = l } }

// withHeartbeat sets the interval and the maximum wait for each ping ACK.
func withHeartbeat(interval, timeout time.Duration) option {
	return func(o *options) { o.heartbeatInterval = interval; o.heartbeatTimeout = timeout }
}
func withRequestTimeout(timeout time.Duration) option {
	return func(o *options) { o.requestTimeout = timeout }
}
func withReconnectPolicy(p ReconnectPolicy) option { return func(o *options) { o.reconnect = p } }
func withNotificationBuffer(size int) option       { return func(o *options) { o.notificationBuffer = size } }
func withMaxFrameBytes(size int64) option          { return func(o *options) { o.maxFrameBytes = size } }
func withMediaLimits(upload, download int64) option {
	return func(o *options) { o.uploadLimit = upload; o.downloadLimit = download }
}
func withChunkSize(size int) option { return func(o *options) { o.chunkSize = size } }
