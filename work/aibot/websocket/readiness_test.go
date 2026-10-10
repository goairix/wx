package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestCallbacksImmediatelyAfterSubscribeACKCanReply(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	for _, reconnect := range []bool{false, true} {
		for _, kind := range []string{"message", "event"} {
			name := kind
			if reconnect {
				name += "/reconnect"
			} else {
				name += "/initial"
			}
			t.Run(name, func(t *testing.T) {
				first, second := newTestConnection(), newTestConnection()
				var calls atomic.Int32
				c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) {
					if calls.Add(1) == 1 {
						return first, nil
					}
					return second, nil
				})), withHeartbeat(time.Hour, time.Second), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, MaxAttempts: 1}))
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				defer func() { _ = c.Close() }()
				type replyResult struct {
					err         error
					readyNotice bool
					state       State
				}
				replies := make(chan replyResult, 1)
				consumerDone := make(chan struct{})
				go func() {
					defer close(consumerDone)
					ready := false
					for n := range c.Notifications() {
						if n.State == StateSubscribing {
							ready = false
						}
						if n.State == StateReady {
							ready = true
						}
						if n.Message == nil && n.Event == nil {
							continue
						}
						result := replyResult{readyNotice: ready, state: c.State()}
						if n.Message != nil {
							_, result.err = c.replyMarkdown(ctx, n.Frame.Headers.ReqID, "received")
						} else {
							_, result.err = c.replyWelcome(ctx, n.Frame.Headers.ReqID, MessageBody{MsgType: "text", Text: &TextContent{Content: "welcome"}})
						}
						replies <- result
					}
				}()
				connected := make(chan error, 1)
				go func() { connected <- c.Connect(ctx) }()
				auth := first.next(t)
				connection := first
				if reconnect {
					first.ack(auth.Headers.ReqID, 0, nil)
					if connectErr := <-connected; connectErr != nil {
						t.Fatal(connectErr)
					}
					first.drop()
					auth = second.next(t)
					connection = second
				}
				if auth.Command != CmdSubscribe {
					t.Fatalf("expected subscribe, got %+v", auth)
				}
				connection.ack(auth.Headers.ReqID, 0, nil)
				if kind == "message" {
					connection.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"first-callback"},"body":{"msgid":"first","msgtype":"text","text":{"content":"hello"}}}`)
				} else {
					connection.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"first-callback"},"body":{"msgid":"first","msgtype":"event","event":{"eventtype":"enter_chat"}}}`)
				}
				if !reconnect {
					if connectErr := <-connected; connectErr != nil {
						t.Fatal(connectErr)
					}
				}
				select {
				case result := <-replies:
					t.Fatalf("callback rejected before ready: state=%s, ready_notice=%v, err=%v", result.state, result.readyNotice, result.err)
				case bytes := <-connection.writes:
					var frame Frame
					if decodeErr := json.Unmarshal(bytes, &frame); decodeErr != nil {
						t.Fatal(decodeErr)
					}
					expected := CmdRespond
					if kind == "event" {
						expected = CmdWelcome
					}
					if frame.Command != expected || frame.Headers.ReqID != "first-callback" {
						t.Fatalf("reply frame %+v", frame)
					}
					connection.ack(frame.Headers.ReqID, 0, nil)
				case <-time.After(time.Second):
					t.Fatal("no callback reply")
				}
				select {
				case result := <-replies:
					if result.err != nil || !result.readyNotice || result.state != StateReady {
						t.Fatalf("callback was not ready: %+v", result)
					}
				case <-time.After(time.Second):
					t.Fatal("reply not completed")
				}
				_ = c.Close()
				<-consumerDone
			})
		}
	}
}

func TestRejectedSubscribeDoesNotPublishBusinessCallbacks(t *testing.T) {
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	auth := s.next(t)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"early"},"body":{"msgtype":"text","text":{"content":"early"}}}`)
	s.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"early-event"},"body":{"msgtype":"event","event":{"eventtype":"enter_chat"}}}`)
	s.ack(auth.Headers.ReqID, 401, nil)
	if connectErr := <-result; connectErr == nil {
		t.Fatal("subscribe accepted")
	}
	for n := range c.Notifications() {
		if n.Message != nil || n.Event != nil {
			t.Fatalf("unauthenticated business callback published: %+v", n)
		}
	}
}

func TestReplacementWhileSubscribingTerminatesWithoutReady(t *testing.T) {
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	s.next(t)
	s.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"replaced"},"body":{"msgtype":"event","event":{"eventtype":"disconnected_event"}}}`)
	select {
	case connectErr := <-result:
		if !errors.Is(connectErr, ErrReplaced) {
			t.Fatal(connectErr)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement blocked by readiness")
	}
	found := false
	for n := range c.Notifications() {
		if n.State == StateReady {
			t.Fatal("became ready without subscription ACK")
		}
		if n.Event != nil && n.Event.Event.EventType == "disconnected_event" {
			found = true
		}
	}
	if !found || c.State() != StateReplaced {
		t.Fatalf("replacement lost: state=%s event=%v", c.State(), found)
	}
}

func TestPreReadyCallbacksDoNotBlockSubscribeACK(t *testing.T) {
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	auth := s.next(t)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"early-message"},"body":{"msgtype":"text","text":{"content":"early"}}}`)
	s.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"early-event"},"body":{"msgtype":"event","event":{"eventtype":"enter_chat"}}}`)
	s.ack(auth.Headers.ReqID, 0, nil)
	select {
	case connectErr := <-result:
		if connectErr != nil {
			t.Fatal(connectErr)
		}
	case <-time.After(time.Second):
		t.Fatal("business callback blocked authentication ACK")
	}
	ready := false
	expected := []string{"early-message", "early-event"}
	for len(expected) > 0 {
		select {
		case n := <-c.Notifications():
			if n.State == StateReady {
				ready = true
			}
			if n.Message == nil && n.Event == nil {
				continue
			}
			if !ready || n.Frame.Headers.ReqID != expected[0] || len(n.Frame.Raw) == 0 {
				t.Fatalf("callback order/readiness incorrect: %+v", n)
			}
			expected = expected[1:]
		case <-time.After(time.Second):
			t.Fatal("staged callbacks not released")
		}
	}
}

func TestPreReadyCallbackQueueOverflowTerminates(t *testing.T) {
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withNotificationBuffer(4), withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	s.next(t)
	for i := 0; i < 5; i++ {
		s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"early"},"body":{"msgtype":"text","text":{"content":"early"}}}`)
	}
	select {
	case connectErr := <-result:
		if !errors.Is(connectErr, ErrNotificationOverflow) {
			t.Fatal(connectErr)
		}
	case <-time.After(time.Second):
		t.Fatal("unready callback queue overflow did not stop authentication")
	}
	for n := range c.Notifications() {
		if n.Message != nil {
			t.Fatal("callback published before ready")
		}
	}
}

func TestReadyNotificationOverflowDiscardsStagedCallbacks(t *testing.T) {
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withNotificationBuffer(2), withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	auth := s.next(t)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"early"},"body":{"msgtype":"text","text":{"content":"early"}}}`)
	s.ack(auth.Headers.ReqID, 0, nil)
	if connectErr := <-result; !errors.Is(connectErr, ErrNotificationOverflow) {
		t.Fatal(connectErr)
	}
	for n := range c.Notifications() {
		if n.Message != nil {
			t.Fatal("callback published despite failed ready notification")
		}
	}
}

type orderedEOFConnection struct{ *testConnection }

func (s orderedEOFConnection) Read(ctx context.Context) ([]byte, error) {
	select {
	case data := <-s.reads:
		if data == nil {
			return nil, io.EOF
		}
		return data, nil
	case <-s.closed:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func TestReadyCallbackImmediatelyBeforeEOFIsDelivered(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return orderedEOFConnection{s}, nil })), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Second, MaxDelay: time.Second, MaxAttempts: 0}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	auth := s.next(t)
	s.ack(auth.Headers.ReqID, 0, nil)
	if connectErr := <-result; connectErr != nil {
		t.Fatal(connectErr)
	}
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"last"},"body":{"msgtype":"text","text":{"content":"last message"}}}`)
	s.reads <- nil
	received := false
	for n := range c.Notifications() {
		if n.Message != nil && n.Frame.Headers.ReqID == "last" {
			received = true
		}
	}
	if !received {
		t.Fatal("authenticated callback discarded when EOF followed immediately")
	}
}
