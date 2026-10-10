package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type State string

const (
	StateNew          State = "new"
	StateConnecting   State = "connecting"
	StateSubscribing  State = "subscribing"
	StateReady        State = "ready"
	StateReconnecting State = "reconnecting"
	StateReplaced     State = "replaced"
	StateClosed       State = "closed"
)

// Notification carries a state transition, a callback, or a connection error.
// Exactly one consumer should drain Notifications and dispatch business work.
type Notification struct {
	State    State
	Frame    Frame
	Message  *Message
	Event    *EventMessage
	Err      error
	Attempt  int
	Callback *ReceivedCallback
}
type keyLock struct {
	token chan struct{}
	refs  int
}

// Client owns one bot connection and never replays outbound messages.
// It cannot be reused after Close or after its lifecycle context ends.
type Client struct {
	config                    Config
	options                   options
	mu                        sync.Mutex
	state                     State
	started, closed, overflow bool
	terminal                  error
	cancel                    context.CancelFunc
	session                   *session
	done                      chan struct{}
	notices                   chan Notification
	keysMu                    sync.Mutex
	keys                      map[string]*keyLock
}

func newClient(config Config, opts ...option) (*Client, error) {
	o := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	u, err := url.Parse(o.url)
	if config.BotID == "" || config.Secret == "" || err != nil || u.Host == "" || (u.Scheme != "ws" && u.Scheme != "wss") || u.User != nil || o.httpClient == nil || o.logger == nil || o.requestTimeout <= 0 || o.heartbeatInterval <= 0 || o.heartbeatTimeout <= 0 || o.notificationBuffer < 1 || o.maxFrameBytes < 1 || o.downloadLimit < 1 || o.uploadLimit < 1 || o.chunkSize < 1 || o.chunkSize > 512<<10 || o.reconnect.InitialDelay <= 0 || o.reconnect.MaxDelay < o.reconnect.InitialDelay || o.reconnect.MaxAttempts < -1 {
		return nil, operationError("new", "", ErrInvalidArgument)
	}
	if o.dialer == nil {
		o.dialer = webSocketDialer{httpClient: o.httpClient, readLimit: o.maxFrameBytes}
	}
	return &Client{config: config, options: o, state: StateNew, done: make(chan struct{}), notices: make(chan Notification, o.notificationBuffer), keys: make(map[string]*keyLock)}, nil
}
func (c *Client) State() State { c.mu.Lock(); defer c.mu.Unlock(); return c.state }

// Err returns the terminal lifecycle error. Transient reconnect failures are notifications.
func (c *Client) Err() error                         { c.mu.Lock(); defer c.mu.Unlock(); return c.terminal }
func (c *Client) Notifications() <-chan Notification { return c.notices }

// Connect waits for the first successful subscription. ctx controls the entire
// connection lifecycle, including subsequent reconnects, until Close.
func (c *Client) Connect(ctx context.Context) error {
	if ctx == nil {
		return operationError("connect", "", ErrInvalidArgument)
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return operationError("connect", "", ErrClosed)
	}
	if c.started {
		ready := c.state == StateReady
		c.mu.Unlock()
		if ready {
			return nil
		}
		return operationError("connect", "", ErrNotReady)
	}
	c.started = true
	life, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.mu.Unlock()
	first := make(chan error, 1)
	go c.run(life, first)
	select {
	case err := <-first:
		return err
	case <-ctx.Done():
		return operationError("connect", "", ctx.Err())
	case <-c.done:
		if err := c.Err(); err != nil {
			return err
		}
		return operationError("connect", "", ErrClosed)
	}
}

// Close permanently ends the connection and promptly releases pending requests.
func (c *Client) Close() error {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		if c.session != nil {
			c.session.fail(ErrClosed)
		}
		if c.cancel != nil {
			c.cancel()
		}
	}
	started := c.started
	if !started {
		c.state = StateClosed
		select {
		case <-c.done:
		default:
			close(c.notices)
			close(c.done)
		}
	}
	c.mu.Unlock()
	if started {
		<-c.done
	}
	return nil
}
func (c *Client) notify(n Notification) bool {
	select {
	case c.notices <- n:
		return true
	default:
	}
	c.mu.Lock()
	c.overflow = true
	s := c.session
	c.mu.Unlock()
	if s != nil {
		s.fail(ErrNotificationOverflow)
	}
	return false
}
func (c *Client) transition(ctx context.Context, state State, attempt int, err error) bool {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()
	// Error strings, frames, credentials and network addresses are deliberately absent.
	c.options.logger.Log(ctx, logging.LevelInfo, "work.aibot.state", logging.String("state", string(state)), logging.Int("attempt", attempt))
	return c.notify(Notification{State: state, Attempt: attempt, Err: err})
}
func (c *Client) run(ctx context.Context, first chan<- error) {
	var terminal error
	firstSent := false
	defer func() {
		c.mu.Lock()
		c.terminal = terminal
		c.closed = true
		c.session = nil
		c.mu.Unlock()
		if !firstSent {
			first <- terminal
		}
		if !errors.Is(terminal, ErrReplaced) {
			c.transition(ctx, StateClosed, 0, terminal)
		}
		close(c.notices)
		close(c.done)
	}()
	authenticated := false
	phase := PhaseDial
	attempt := 0
	delay := c.options.reconnect.InitialDelay
	if c.options.initialConnect != nil {
		delay = c.options.initialConnect.InitialDelay
	}
	for {
		if err := ctx.Err(); err != nil {
			c.mu.Lock()
			closed := c.closed
			c.mu.Unlock()
			if closed {
				terminal = connectionError(phase, operationError("connect", "", ErrClosed))
			} else {
				terminal = connectionError(phase, operationError("connect", "", err))
			}
			return
		}
		c.mu.Lock()
		overflow := c.overflow
		c.mu.Unlock()
		if overflow {
			terminal = connectionError(phase, operationError("receive", "", ErrNotificationOverflow))
			return
		}
		if attempt > 0 {
			policy := c.options.reconnect
			if !authenticated && c.options.initialConnect != nil {
				policy = *c.options.initialConnect
			}
			if !c.transition(ctx, StateReconnecting, attempt, nil) {
				terminal = connectionError(phase, operationError("receive", "", ErrNotificationOverflow))
				return
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				continue
			case <-timer.C:
			}
			if delay < policy.MaxDelay {
				if delay >= policy.MaxDelay-delay {
					delay = policy.MaxDelay
				} else {
					delay *= 2
				}
			}
		} else {
			if !c.transition(ctx, StateConnecting, 0, nil) {
				terminal = connectionError(phase, operationError("receive", "", ErrNotificationOverflow))
				return
			}
		}
		dialCtx, dialCancel := context.WithTimeout(ctx, c.options.requestTimeout)
		conn, err := c.options.dialer.Dial(dialCtx, c.options.url)
		dialCancel()
		if err != nil && !settings.IsNil(conn) {
			_ = conn.Close()
		}
		if err == nil && settings.IsNil(conn) {
			err = ErrProtocol
		}
		phase = PhaseDial
		if err == nil {
			phase = PhaseSubscribe
			s := newSession(ctx, conn, c)
			c.mu.Lock()
			c.session = s
			closed := c.closed
			c.mu.Unlock()
			if closed {
				s.fail(ErrClosed)
			}
			readerDone := make(chan struct{})
			go func() { defer close(readerDone); s.readLoop() }()
			c.transition(ctx, StateSubscribing, attempt, nil)
			authCtx, authCancel := context.WithTimeout(s.ctx, c.options.requestTimeout)
			_, err = s.request(authCtx, CmdSubscribe, newRequestID(), map[string]string{"bot_id": c.config.BotID, "secret": c.config.Secret})
			authCancel()
			if err == nil && s.ctx.Err() != nil {
				err = s.cause()
			}
			if err == nil {
				if !c.transition(ctx, StateReady, 0, nil) {
					err = ErrNotificationOverflow
				} else if s.ctx.Err() != nil {
					err = s.cause()
				}
			}
			if err == nil {
				err = s.activateCallbacks()
			}
			if err == nil {
				phase = PhaseReceive
				authenticated = true
				attempt = 0
				delay = c.options.reconnect.InitialDelay
				if !firstSent {
					first <- nil
					firstSent = true
				}
				heartbeatDone := make(chan struct{})
				go func() { defer close(heartbeatDone); s.heartbeat() }()
				<-s.ctx.Done()
				err = s.cause()
				<-heartbeatDone
			}
			s.fail(err)
			<-readerDone
			c.mu.Lock()
			if c.session == s {
				c.session = nil
			}
			c.mu.Unlock()
			if s.replaced {
				terminal = connectionError(PhaseReceive, operationError("receive", "", ErrReplaced))
				c.mu.Lock()
				c.terminal = terminal
				c.mu.Unlock()
				c.transition(ctx, StateReplaced, 0, terminal)
				return
			}
			if errors.Is(err, ErrNotificationOverflow) {
				terminal = connectionError(phase, operationError("receive", "", err))
				return
			}
		}
		if ctx.Err() != nil {
			continue
		}
		classified := connectionError(phase, err)
		policy, operation := &c.options.reconnect, "reconnect"
		if !authenticated {
			policy, operation = c.options.initialConnect, "connect"
		}
		if policy == nil || !retryableConnectionError(err) {
			terminal = classified
			return
		}
		attempt++
		if authenticated {
			c.notify(Notification{Err: classified})
		}
		if policy.MaxAttempts >= 0 && attempt > policy.MaxAttempts {
			terminal = operationError(operation, "", errors.Join(ErrRetryExhausted, classified))
			return
		}
		if !authenticated {
			c.notify(Notification{Err: classified})
		}
	}
}
func (s *session) readLoop() {
	for {
		b, err := s.conn.Read(s.ctx)
		receivedAt := time.Now()
		if err != nil {
			s.fail(errors.Join(ErrDisconnected, err))
			return
		}
		if int64(len(b)) > s.owner.options.maxFrameBytes {
			s.fail(ErrProtocol)
			return
		}
		var frame Frame
		if err := json.Unmarshal(b, &frame); err != nil {
			s.fail(errors.Join(ErrProtocol, err))
			return
		}
		frame.Raw = append(json.RawMessage(nil), b...)
		switch frame.Command {
		case CmdMessageCallback:
			var m Message
			if err := json.Unmarshal(frame.Body, &m); err != nil {
				s.fail(errors.Join(ErrProtocol, err))
				return
			}
			target, err := s.received(frame, receivedAt)
			if err != nil {
				s.fail(err)
				return
			}
			if !s.queueCallback(Notification{Frame: frame, Message: &m, Callback: target}) {
				return
			}
		case CmdEventCallback:
			var e EventMessage
			if err := json.Unmarshal(frame.Body, &e); err != nil {
				s.fail(errors.Join(ErrProtocol, err))
				return
			}
			if e.Event.EventType == "disconnected_event" {
				s.replaced = true
				s.owner.notify(Notification{Frame: frame, Event: &e, Err: connectionError(PhaseReceive, operationError("receive", frame.Headers.ReqID, ErrReplaced))})
				s.fail(ErrReplaced)
				return
			}
			target, err := s.received(frame, receivedAt)
			if err != nil {
				s.fail(err)
				return
			}
			if !s.queueCallback(Notification{Frame: frame, Event: &e, Callback: target}) {
				return
			}
		default:
			if frame.ErrCode == nil {
				continue
			}
			s.mu.Lock()
			pending := s.pending[frame.Headers.ReqID]
			if pending != nil {
				delete(s.pending, frame.Headers.ReqID)
			}
			s.mu.Unlock()
			if pending != nil {
				pending <- frame
			}
		}
	}
}

// activateCallbacks flushes staged callbacks after the Ready notification.
// Holding callbackMu prevents later frames from overtaking this bounded flush.
// Publication never waits for business consumption.
func (s *session) activateCallbacks() error {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	for {
		if s.ctx.Err() != nil {
			return s.cause()
		}
		select {
		case notification := <-s.callbacks:
			if !s.owner.notify(notification) {
				return s.cause()
			}
		default:
			s.callbacksReady = true
			return nil
		}
	}
}

func (s *session) heartbeat() {
	ticker := time.NewTicker(s.owner.options.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			_, err := s.request(s.ctx, CmdPing, newRequestID(), nil)
			if err != nil {
				s.fail(err)
				return
			}
		}
	}
}
