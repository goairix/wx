package websocket

import (
	"context"
	"errors"

	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type LimitScope string

const (
	ScopeConversationSend LimitScope = "conversation_send"
	ScopeBotUpload        LimitScope = "bot_upload"
)

type RateLimits struct{ PerMinute, PerHour int }

func ConversationSendLimits() RateLimits { return RateLimits{PerMinute: 30, PerHour: 1000} }
func BotUploadLimits() RateLimits        { return RateLimits{PerMinute: 30, PerHour: 1000} }

// LimitRequest describes an admission attempt, without message content or secrets.
// StreamStart distinguishes a new stream from a refresh; the platform does not
// explicitly document how each refresh consumes its message quota.
type LimitRequest struct {
	Operation                                                    string
	Scope                                                        LimitScope
	BotID, ConversationID, ConversationType, RequestID, StreamID string
	StreamStart                                                  bool
}

// Limiter must support concurrent calls and honor context cancellation. It runs
// outside the write lock and never handles protocol heartbeats/subscriptions.
// Share an implementation across clients for quotas that survive reconnection.
type Limiter interface {
	Allow(context.Context, LimitRequest) error
}
type LimiterFunc func(context.Context, LimitRequest) error

func (f LimiterFunc) Allow(ctx context.Context, request LimitRequest) error { return f(ctx, request) }

type limitTargetKey struct{}

func (c *Client) admit(sendCtx, ackCtx context.Context, s *session, command, reqID string, p *sendPolicy) error {
	if p != nil {
		if err := p.check(); err != nil {
			return err
		}
	}
	if sendCtx.Err() != nil {
		return sendContextCause(sendCtx, ackCtx, s, p)
	}
	if settings.IsNil(c.options.limiter) {
		return nil
	}
	r := LimitRequest{Operation: command, RequestID: reqID, BotID: c.config.BotID}
	switch command {
	case CmdUploadInit:
		r.Scope = ScopeBotUpload
	case CmdSend:
		destination, _ := ackCtx.Value(limitTargetKey{}).(LimitRequest)
		r.ConversationID = destination.ConversationID
		r.ConversationType = destination.ConversationType
		r.Scope = ScopeConversationSend
	case CmdRespond, CmdWelcome, CmdUpdate:
		r.Scope = ScopeConversationSend
		if p != nil {
			p.state.mu.Lock()
			r.ConversationID = p.state.chatID
			r.ConversationType = p.state.chatType
			r.StreamID = p.streamID
			if r.StreamID != "" {
				stream := p.state.streams[r.StreamID]
				r.StreamStart = stream == nil || stream.started.IsZero()
			}
			p.state.mu.Unlock()
		}
	default:
		return nil
	}
	if r.Scope == ScopeConversationSend && r.ConversationID == "" {
		return ErrInvalidArgument
	}
	err := c.options.limiter.Allow(sendCtx, r)
	if sendCtx.Err() != nil {
		return errors.Join(err, sendContextCause(sendCtx, ackCtx, s, p))
	}
	if err != nil {
		return err
	}
	if p != nil {
		return p.check()
	}
	return nil
}
