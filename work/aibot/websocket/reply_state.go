package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

// A private pointer makes state shared even when ReceivedCallback is copied.
type replyState struct {
	mu                   sync.Mutex
	receivedAt, deadline time.Time
	chatID, chatType     string
	streams              map[string]*streamState
}
type streamState struct {
	started  time.Time
	finished bool
}
type sendPolicy struct {
	state    *replyState
	streamID string
	finish   bool
}
type sendPolicyKey struct{}
type requestControlKey struct{}
type requestControl struct {
	sendContext context.Context
	policy      *sendPolicy
}

func (p *sendPolicy) deadline() time.Time {
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	deadline := p.state.deadline
	if stream := p.state.streams[p.streamID]; p.streamID != "" && stream != nil && !stream.started.IsZero() {
		end := stream.started.Add(10 * time.Minute)
		if end.Before(deadline) {
			deadline = end
		}
	}
	return deadline
}
func (p *sendPolicy) check() error {
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	now := time.Now()
	if !p.state.deadline.IsZero() && !now.Before(p.state.deadline) {
		return aibot.ErrReplyExpired
	}
	if stream := p.state.streams[p.streamID]; p.streamID != "" && stream != nil {
		if stream.finished {
			return aibot.ErrStreamFinished
		}
		if !stream.started.IsZero() && !now.Before(stream.started.Add(10*time.Minute)) {
			return aibot.ErrStreamExpired
		}
	}
	return nil
}
func (p *sendPolicy) beforeWrite(now time.Time) {
	if p.streamID == "" {
		return
	}
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	stream := p.state.streams[p.streamID]
	if stream == nil {
		stream = &streamState{}
		p.state.streams[p.streamID] = stream
	}
	if stream.started.IsZero() {
		stream.started = now
	}
}
func (p *sendPolicy) afterACK() {
	if p.streamID == "" || !p.finish {
		return
	}
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	p.state.streams[p.streamID].finished = true
}
func policyFromContext(ctx context.Context) *sendPolicy {
	p, _ := ctx.Value(sendPolicyKey{}).(*sendPolicy)
	return p
}

// This context bounds sending only. ACK waiting retains the original budget.
func makeSendContext(ctx context.Context, s *session, p *sendPolicy) (context.Context, func()) {
	life, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(s.ctx, cancel)
	var deadlineCancel context.CancelFunc
	if p != nil && !p.deadline().IsZero() {
		life, deadlineCancel = context.WithDeadline(life, p.deadline())
	}
	return life, func() {
		if deadlineCancel != nil {
			deadlineCancel()
		}
		stop()
		cancel()
	}
}
func sendContextCause(sendCtx, ackCtx context.Context, s *session, p *sendPolicy) error {
	// Subscription contexts inherit s.ctx cancellation, so preserve the
	// established session cause before inspecting the child context error.
	if s.ctx.Err() != nil {
		return s.cause()
	}
	if err := ackCtx.Err(); err != nil {
		return err
	}
	if p != nil {
		if err := p.check(); err != nil {
			return err
		}
	}
	return sendCtx.Err()
}
