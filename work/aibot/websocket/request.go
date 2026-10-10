package websocket

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/logging"
)

type session struct {
	owner          *Client
	conn           Connection
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	pending        map[string]chan Frame
	failure        error
	failOnce       sync.Once
	writer         chan struct{}
	callbacks      chan Notification
	callbackMu     sync.Mutex
	callbacksReady bool
	replaced       bool
}

func newSession(ctx context.Context, conn Connection, owner *Client) *session {
	life, cancel := context.WithCancel(ctx)
	return &session{owner: owner, conn: conn, ctx: life, cancel: cancel, pending: make(map[string]chan Frame), writer: make(chan struct{}, 1), callbacks: make(chan Notification, owner.options.notificationBuffer)}
}

// queueCallback stages callbacks until subscription readiness. Once activated,
// callbacks publish directly, before the reader can observe a subsequent EOF.
func (s *session) queueCallback(notification Notification) bool {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	if s.callbacksReady {
		return s.owner.notify(notification)
	}
	select {
	case s.callbacks <- notification:
		return true
	default:
		s.fail(ErrNotificationOverflow)
		return false
	}
}
func (s *session) fail(err error) {
	s.failOnce.Do(func() {
		if err == nil {
			err = ErrDisconnected
		}
		s.mu.Lock()
		s.failure = err
		s.mu.Unlock()
		s.cancel()
		_ = s.conn.Close()
	})
}
func (s *session) cause() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure != nil {
		return s.failure
	}
	return s.ctx.Err()
}

var requestCounter atomic.Uint64

// A canceled operation can invalidate a session whose lifecycle is still live.
// Keep this provenance separate from cancellation of the lifecycle itself.
type canceledRequestCause struct{ cause error }

func (e *canceledRequestCause) Error() string { return "aibot in-flight request canceled" }
func (e *canceledRequestCause) Unwrap() error { return e.cause }

func requestFailureCause(ctx context.Context, err error) error {
	if ctx.Err() == context.Canceled {
		return &canceledRequestCause{cause: errors.Join(ctx.Err(), err)}
	}
	return err
}

func newRequestID() string {
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err == nil {
		return "wx_" + hex.EncodeToString(nonce[:])
	}
	return fmt.Sprintf("wx_%d_%d", time.Now().UnixNano(), requestCounter.Add(1))
}
func (c *Client) request(ctx context.Context, command, reqID string, body any) (Frame, error) {
	if ctx == nil {
		return Frame{}, operationError(command, reqID, ErrInvalidArgument)
	}
	if reqID == "" {
		reqID = newRequestID()
	}
	if len(reqID) > 256 || !utf8.ValidString(reqID) {
		return Frame{}, operationError(command, reqID, ErrInvalidArgument)
	}
	bound, err := c.bindSession(ctx)
	if err != nil {
		return Frame{}, operationError(command, reqID, err)
	}
	s := bound.Value(sessionContextKey{}).(*session)
	deadline, cancel := context.WithTimeout(bound, c.options.requestTimeout)
	defer cancel()
	policy := policyFromContext(deadline)
	sendCtx, sendCancel := makeSendContext(deadline, s, policy)
	defer sendCancel()
	controlled := context.WithValue(deadline, requestControlKey{}, &requestControl{sendContext: sendCtx, policy: policy})
	frame, err := c.withKey(sendCtx, "request:"+reqID, func() (Frame, error) {
		if admissionErr := c.admit(sendCtx, deadline, s, command, reqID, policy); admissionErr != nil {
			return Frame{}, operationError(command, reqID, admissionErr)
		}
		return s.request(controlled, command, reqID, body)
	})
	var queueError *coreerrors.Error
	if errors.As(err, &queueError) && queueError.Operation == "work.aibot.websocket.queue" {
		if sendCtx.Err() != nil {
			err = sendContextCause(sendCtx, deadline, s, policy)
		}
		return frame, operationError(command, reqID, err)
	}
	return frame, err
}

// A queued request remains bound to the session it was started on.
// Context values supplied by the caller remain inherited unchanged.
type sessionContextKey struct{}

func (c *Client) bindSession(ctx context.Context) (context.Context, error) {
	c.mu.Lock()
	current, state, closed := c.session, c.state, c.closed
	c.mu.Unlock()
	if closed {
		return nil, ErrClosed
	}
	if expected, ok := ctx.Value(sessionContextKey{}).(*session); ok && expected != current {
		return nil, ErrDisconnected
	}
	if current == nil || state != StateReady {
		return nil, ErrNotReady
	}
	if current.ctx.Err() != nil {
		return nil, current.cause()
	}
	return context.WithValue(ctx, sessionContextKey{}, current), nil
}

// withKey serializes operations without preventing queued callers from canceling.
// Entries are removed once both the holder and all waiters have returned.
func (c *Client) withKey(ctx context.Context, key string, fn func() (Frame, error)) (Frame, error) {
	if ctx == nil {
		return Frame{}, operationError("queue", "", ErrInvalidArgument)
	}
	s, _ := ctx.Value(sessionContextKey{}).(*session)
	if s == nil {
		c.mu.Lock()
		s = c.session
		c.mu.Unlock()
	}
	var disconnected <-chan struct{}
	if s != nil {
		disconnected = s.ctx.Done()
	}
	c.keysMu.Lock()
	k := c.keys[key]
	if k == nil {
		k = &keyLock{token: make(chan struct{}, 1)}
		c.keys[key] = k
	}
	k.refs++
	c.keysMu.Unlock()
	defer func() {
		c.keysMu.Lock()
		k.refs--
		if k.refs == 0 {
			delete(c.keys, key)
		}
		c.keysMu.Unlock()
	}()
	// Lifecycle cancellation must release callers waiting behind another reply.
	select {
	case k.token <- struct{}{}:
		defer func() { <-k.token }()
	case <-ctx.Done():
		return Frame{}, operationError("queue", "", ctx.Err())
	case <-c.done:
		return Frame{}, operationError("queue", "", ErrClosed)
	case <-disconnected:
		return Frame{}, operationError("queue", "", s.cause())
	}
	if err := ctx.Err(); err != nil {
		return Frame{}, operationError("queue", "", err)
	}
	if s != nil && s.ctx.Err() != nil {
		return Frame{}, operationError("queue", "", s.cause())
	}
	return fn()
}
func (s *session) request(ctx context.Context, command, reqID string, body any) (Frame, error) {
	sendCtx := ctx
	var policy *sendPolicy
	if control, _ := ctx.Value(requestControlKey{}).(*requestControl); control != nil {
		sendCtx = control.sendContext
		policy = control.policy
	}
	if policy != nil {
		if policyErr := policy.check(); policyErr != nil {
			return Frame{}, operationError(command, reqID, policyErr)
		}
	}
	if sendCtx.Err() != nil {
		return Frame{}, operationError(command, reqID, sendContextCause(sendCtx, ctx, s, policy))
	}
	var rawBody json.RawMessage
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return Frame{}, operationError(command, reqID, err)
		}
		rawBody = b
	}
	wire, err := json.Marshal(Frame{Command: command, Headers: Headers{ReqID: reqID}, Body: rawBody})
	if err != nil {
		return Frame{}, operationError(command, reqID, err)
	}
	pending := make(chan Frame, 1)
	s.mu.Lock()
	if s.failure != nil || s.ctx.Err() != nil {
		cause := s.failure
		if cause == nil {
			cause = s.ctx.Err()
		}
		s.mu.Unlock()
		return Frame{}, operationError(command, reqID, cause)
	}
	if s.pending[reqID] != nil {
		s.mu.Unlock()
		return Frame{}, operationError(command, reqID, ErrProtocol)
	}
	s.pending[reqID] = pending
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.pending[reqID] == pending {
			delete(s.pending, reqID)
		}
		s.mu.Unlock()
	}()
	select {
	case s.writer <- struct{}{}:
	case <-sendCtx.Done():
		return Frame{}, operationError(command, reqID, sendContextCause(sendCtx, ctx, s, policy))
	case <-s.ctx.Done():
		return Frame{}, operationError(command, reqID, s.cause())
	}
	if sendCtx.Err() != nil {
		<-s.writer
		return Frame{}, operationError(command, reqID, sendContextCause(sendCtx, ctx, s, policy))
	}
	if s.ctx.Err() != nil {
		<-s.writer
		return Frame{}, operationError(command, reqID, s.cause())
	}
	if policy != nil {
		if policyErr := policy.check(); policyErr != nil {
			<-s.writer
			return Frame{}, operationError(command, reqID, policyErr)
		}
	}
	if command == CmdPing {
		// A ping cannot be overdue while still queued behind another valid
		// write. Bound its own send and ACK only after acquiring the writer.
		var pingCancel context.CancelFunc
		ctx, pingCancel = context.WithTimeout(ctx, s.owner.options.heartbeatTimeout)
		sendCtx = ctx
		defer pingCancel()
	}
	var writeCtx context.Context
	var writeCancel context.CancelFunc
	if policy != nil {
		policy.beforeWrite(time.Now())
		// The first stream acquires its ten-minute window only now. Recompute
		// the Write deadline while keeping the original context for its ACK.
		writeCtx, writeCancel = context.WithDeadline(sendCtx, policy.deadline())
	} else {
		writeCtx, writeCancel = context.WithCancel(sendCtx)
	}
	stop := context.AfterFunc(s.ctx, writeCancel)
	err = s.conn.Write(writeCtx, wire)
	stop()
	writeCancel()
	<-s.writer
	if err != nil {
		s.fail(requestFailureCause(ctx, errors.Join(ErrDisconnected, err)))
		return Frame{}, operationError(command, reqID, errors.Join(s.cause(), err))
	}
	s.owner.options.logger.Log(ctx, logging.LevelDebug, "work.aibot.request", logging.String("operation", command))
	select {
	case ack := <-pending:
		if ack.ErrCode == nil {
			return Frame{}, operationError(command, reqID, ErrProtocol)
		}
		if *ack.ErrCode != 0 {
			// Server text is retained in the returned error but is never logged.
			return ack, &coreerrors.Error{Platform: "work", Operation: "work.aibot.websocket." + command, RequestID: reqID, Code: strconv.Itoa(*ack.ErrCode), Message: ack.ErrMsg}
		}
		if policy != nil {
			policy.afterACK()
		}
		return ack, nil
	case <-ctx.Done():
		// No protocol sequence number exists for reused reply req_id. Stop this
		// connection so a delayed ACK cannot acknowledge the next reply.
		// A subscription context inherits session cancellation. Keep its actual
		// transport failure instead of replacing it with context.Canceled.
		if s.ctx.Err() != nil {
			return Frame{}, operationError(command, reqID, s.cause())
		}
		s.fail(requestFailureCause(ctx, ctx.Err()))
		return Frame{}, operationError(command, reqID, ctx.Err())
	case <-s.ctx.Done():
		return Frame{}, operationError(command, reqID, s.cause())
	}
}
