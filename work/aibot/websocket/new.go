package websocket

import (
	"github.com/goairix/wx/v2/work/aibot"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

func New(cfg Config, opts ...aibot.Option) (*Client, error) {
	shared, err := settings.Resolve(opts...)
	if err != nil {
		return nil, aibot.WrapError("websocket.new", "", aibot.ErrInvalidArgument)
	}
	o := defaultOptions()
	o.httpClient = shared.HTTPClient
	o.logger = shared.Logger
	o.requestTimeout = shared.RequestTimeout
	o.downloadLimit = shared.MaxDownloadBytes
	if cfg.URL != "" {
		o.url = cfg.URL
	}
	if cfg.Dialer != nil {
		if settings.IsNil(cfg.Dialer) {
			return nil, aibot.WrapError("websocket.new", "", aibot.ErrInvalidArgument)
		}
		o.dialer = cfg.Dialer
	}
	if cfg.HeartbeatInterval != 0 {
		o.heartbeatInterval = cfg.HeartbeatInterval
	}
	if cfg.HeartbeatTimeout != 0 {
		o.heartbeatTimeout = cfg.HeartbeatTimeout
	}
	if cfg.Reconnect != nil {
		o.reconnect = *cfg.Reconnect
	}
	if cfg.InitialConnect != nil {
		p := *cfg.InitialConnect
		if !validReconnectPolicy(p) {
			return nil, aibot.WrapError("websocket.new", "", aibot.ErrInvalidArgument)
		}
		o.initialConnect = &p
	}
	if cfg.Limiter != nil {
		if settings.IsNil(cfg.Limiter) {
			return nil, aibot.WrapError("websocket.new", "", aibot.ErrInvalidArgument)
		}
		o.limiter = cfg.Limiter
	}
	if cfg.NotificationBuffer != 0 {
		o.notificationBuffer = cfg.NotificationBuffer
	}
	if cfg.MaxFrameBytes != 0 {
		o.maxFrameBytes = cfg.MaxFrameBytes
	}
	if cfg.ChunkSize != 0 {
		o.chunkSize = cfg.ChunkSize
	}
	return newClient(cfg, func(target *options) { *target = o })
}
