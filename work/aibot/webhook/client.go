// Package webhook implements encrypted intelligent-bot HTTP callbacks.
package webhook

import (
	corehook "github.com/goairix/wx/v2/core/webhook"
	"github.com/goairix/wx/v2/work/aibot"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type Config struct {
	Token, EncodingAESKey                                  string
	MaxRequestBytes, MaxPlainReplyBytes, MaxWireReplyBytes int64
	ErrorResponse                                          corehook.ErrorResponse
}
type Client struct {
	config Config
	shared settings.Settings
	cipher *corehook.MessageCipher
}

func New(cfg Config, opts ...aibot.Option) (*Client, error) {
	shared, err := settings.Resolve(opts...)
	if err != nil {
		return nil, aibot.WrapError("webhook.new", "", aibot.ErrInvalidArgument)
	}
	if cfg.MaxRequestBytes == 0 {
		cfg.MaxRequestBytes = 1 << 20
	}
	if cfg.MaxPlainReplyBytes == 0 {
		cfg.MaxPlainReplyBytes = 20 << 20
	}
	if cfg.MaxWireReplyBytes == 0 {
		cfg.MaxWireReplyBytes = 28 << 20
	}
	if cfg.Token == "" || cfg.MaxRequestBytes < 1 || cfg.MaxRequestBytes > 1<<40 || cfg.MaxPlainReplyBytes < 1 || cfg.MaxPlainReplyBytes > 1<<32-1 || cfg.MaxWireReplyBytes < wireBudget(cfg.MaxPlainReplyBytes) {
		return nil, aibot.WrapError("webhook.new", "", aibot.ErrInvalidArgument)
	}
	cipher, err := corehook.NewMessageCipher(corehook.CipherConfig{EncodingAESKey: cfg.EncodingAESKey, AllowEmptyReceiver: true})
	if err != nil {
		return nil, aibot.WrapError("webhook.new", "", err)
	}
	return &Client{config: cfg, shared: shared, cipher: cipher}, nil
}
func wireBudget(n int64) int64 {
	if n < 1 || n > 1<<32-1 {
		return 1<<63 - 1
	}
	return ((n+20+32+2)/3)*4 + 4096
}
