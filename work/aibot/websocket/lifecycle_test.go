package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/logging"
)

func waitState(t *testing.T, c *Client, want State) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if c.State() == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("state %s, want %s (err %v)", c.State(), want, c.Err())
		case <-ticker.C:
		}
	}
}
func TestRepliesSerializeUntilACK(t *testing.T) {
	c, s := newTestClient(t)
	one := make(chan error, 1)
	two := make(chan error, 1)
	go func() { _, err := c.replyMarkdown(context.Background(), "r", "first"); one <- err }()
	first := s.next(t)
	go func() { _, err := c.replyMarkdown(context.Background(), "r", "second"); two <- err }()
	select {
	case b := <-s.writes:
		t.Fatalf("second sent before firstACK %s", b)
	case <-time.After(20 * time.Millisecond):
	}
	s.ack(first.Headers.ReqID, 0, nil)
	second := s.next(t)
	var body MessageBody
	if err := json.Unmarshal(second.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Markdown.Content != "second" {
		t.Fatal(body)
	}
	s.ack(second.Headers.ReqID, 0, nil)
	if err := <-one; err != nil {
		t.Fatal(err)
	}
	if err := <-two; err != nil {
		t.Fatal(err)
	}
}
func TestCanceledQueuedRequestPreservesOperationAndRequestID(t *testing.T) {
	c, s := newTestClient(t)
	done := make(chan error, 1)
	go func() { _, err := c.replyMarkdown(context.Background(), "r", "first"); done <- err }()
	s.next(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := c.replyMarkdown(ctx, "r", "second")
	var apiErr *coreerrors.Error
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &apiErr) || apiErr.RequestID != "r" || apiErr.Operation != "work.aibot.websocket.aibot_respond_msg" {
		t.Fatalf("lost context: %+v", apiErr)
	}
	s.ack("r", 0, nil)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	select {
	case b := <-s.writes:
		t.Fatalf("canceled request sent %s", b)
	default:
	}
}
func TestACKTimeoutDisconnectsBeforeReusedID(t *testing.T) {
	c, s := newTestClient(t, withRequestTimeout(20*time.Millisecond), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, MaxAttempts: 0}))
	result := make(chan error, 1)
	go func() { _, err := c.replyMarkdown(context.Background(), "r", "first"); result <- err }()
	s.next(t)
	err := <-result
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	select {
	case <-s.closed:
	case <-time.After(time.Second):
		t.Fatal("late ACK ambiguity not isolated")
	}
	_, err = c.replyMarkdown(context.Background(), "r", "second")
	if err == nil {
		t.Fatal("sent after timeout")
	}
	select {
	case b := <-s.writes:
		t.Fatalf("replayed %s", b)
	default:
	}
}
func TestReconnectResubscribesWithoutReplay(t *testing.T) {
	first, second := newTestConnection(), newTestConnection()
	var calls atomic.Int32
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) {
		if calls.Add(1) == 1 {
			return first, nil
		}
		return second, nil
	})), withHeartbeat(time.Hour, time.Second), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, MaxAttempts: 2}))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	connected := make(chan error, 1)
	go func() { connected <- c.Connect(context.Background()) }()
	f := first.next(t)
	first.ack(f.Headers.ReqID, 0, nil)
	if err := <-connected; err != nil {
		t.Fatal(err)
	}
	pending := make(chan error, 1)
	go func() { _, err := c.replyMarkdown(context.Background(), "uncertain", "message"); pending <- err }()
	first.next(t)
	first.drop()
	if err := <-pending; !errors.Is(err, ErrDisconnected) {
		t.Fatal(err)
	}
	auth := second.next(t)
	if auth.Command != CmdSubscribe {
		t.Fatalf("replay instead of subscribe: %+v", auth)
	}
	second.ack(auth.Headers.ReqID, 0, nil)
	waitState(t, c, StateReady)
	select {
	case b := <-second.writes:
		t.Fatalf("replayed deliveryunknown %s", b)
	case <-time.After(20 * time.Millisecond):
	}
}
func TestReplacementStopsReconnectAndPreservesEvent(t *testing.T) {
	c, s := newTestClient(t)
	s.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"replace"},"body":{"msgtype":"event","event":{"eventtype":"disconnected_event"}}}`)
	waitState(t, c, StateReplaced)
	if !errors.Is(c.Err(), ErrReplaced) {
		t.Fatalf("replacementcause %v", c.Err())
	}
	found := false
	for n := range c.Notifications() {
		if n.Event != nil && n.Event.Event.EventType == "disconnected_event" && n.Frame.Headers.ReqID == "replace" {
			found = true
		}
	}
	if !found {
		t.Fatal("replacement event lost")
	}
	select {
	case b := <-s.writes:
		t.Fatalf("reconnected after replaced %s", b)
	default:
	}
}
func TestHeartbeatNeedsMatchingACK(t *testing.T) {
	c, s := newTestClient(t, withHeartbeat(5*time.Millisecond, 15*time.Millisecond), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, MaxAttempts: 0}))
	ping := s.next(t)
	if ping.Command != CmdPing || len(ping.Body) != 0 {
		t.Fatal(ping)
	}
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"traffic"},"body":{"msgtype":"text","text":{"content":"alive"}}}`)
	s.ack("unrelated", 0, nil)
	waitState(t, c, StateClosed)
	if !errors.Is(c.Err(), context.DeadlineExceeded) {
		t.Fatalf("heartbeat timeout %v", c.Err())
	}
}
func TestNotificationOverflowTerminatesExplicitly(t *testing.T) {
	c, s := newTestClient(t, withNotificationBuffer(4)) // three lifecycle notifications already queued
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"1"},"body":{"msgtype":"text","text":{"content":"one"}}}`)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"2"},"body":{"msgtype":"text","text":{"content":"two"}}}`)
	waitState(t, c, StateClosed)
	if !errors.Is(c.Err(), ErrNotificationOverflow) {
		t.Fatal(c.Err())
	}
}
func TestErrorCodeAndLogRedaction(t *testing.T) {
	var mu sync.Mutex
	var logs []string
	logger := logging.LoggerFunc(func(_ context.Context, _ logging.Level, event string, attrs ...logging.Attr) {
		mu.Lock()
		defer mu.Unlock()
		logs = append(logs, fmt.Sprint(event, attrs))
	})
	c, s := newTestClient(t, withLogger(logger))
	done := make(chan error, 1)
	go func() { _, err := c.replyMarkdown(context.Background(), "r", "SECRET_CONTENT"); done <- err }()
	s.next(t)
	s.ack("r", 40014, nil)
	err := <-done
	var e *coreerrors.Error
	if !errors.As(err, &e) || e.Code != "40014" || e.RequestID != "r" {
		t.Fatal(err)
	}
	_, _ = c.downloadMedia(context.Background(), ":signed?SECRET_DOWNLOAD", "SECRET_AES")
	_ = c.Close()
	mu.Lock()
	defer mu.Unlock()
	for _, entry := range logs {
		for _, sensitive := range []string{"secret", "SECRET_CONTENT", "SECRET_DOWNLOAD", "SECRET_AES"} {
			if strings.Contains(entry, sensitive) {
				t.Fatalf("sensitive log: %s", entry)
			}
		}
	}
}

func TestDisconnectCancelsQueuedStreamBeforeLockRelease(t *testing.T) {
	c, s := newTestClient(t, withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Second, MaxDelay: time.Second, MaxAttempts: 1}))
	locked, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	go func() {
		_, _ = c.withKey(context.Background(), "stream:queued", func() (Frame, error) { close(locked); <-release; return Frame{}, nil })
	}()
	<-locked
	result := make(chan error, 1)
	go func() {
		_, err := c.replyStream(context.Background(), "queued-reply", Stream{ID: "queued", Content: "unsent"})
		result <- err
	}()
	deadline := time.After(time.Second)
	for {
		c.keysMu.Lock()
		refs := c.keys["stream:queued"].refs
		c.keysMu.Unlock()
		if refs == 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("reply not queued")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	s.drop()
	select {
	case err := <-result:
		if !errors.Is(err, ErrDisconnected) {
			t.Fatalf("queued error %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("disconnect stranded queued stream")
	}
}

type writeFailureConnection struct {
	*testConnection
	failure       error
	writing       chan struct{}
	waitForCancel bool
}

func (s *writeFailureConnection) Write(ctx context.Context, data []byte) error {
	var frame Frame
	if err := json.Unmarshal(data, &frame); err != nil {
		return err
	}
	if frame.Command == CmdSubscribe {
		return s.testConnection.Write(ctx, data)
	}
	if s.writing != nil {
		close(s.writing)
	}
	if s.waitForCancel {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.failure
}
func TestWriteFailurePreservesSessionAndUnderlyingCauses(t *testing.T) {
	for _, closeDuringWrite := range []bool{false, true} {
		t.Run(fmt.Sprint(closeDuringWrite), func(t *testing.T) {
			base := newTestConnection()
			writeErr := errors.New("write failed")
			conn := &writeFailureConnection{testConnection: base, failure: writeErr, writing: make(chan struct{}), waitForCancel: closeDuringWrite}
			c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return conn, nil })), withHeartbeat(time.Hour, time.Second), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Second, MaxDelay: time.Second, MaxAttempts: 0}))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			connected := make(chan error, 1)
			go func() { connected <- c.Connect(context.Background()) }()
			f := base.next(t)
			base.ack(f.Headers.ReqID, 0, nil)
			if connectedErr := <-connected; connectedErr != nil {
				t.Fatal(connectedErr)
			}
			result := make(chan error, 1)
			go func() {
				_, requestErr := c.replyMarkdown(context.Background(), "failed-write", "content")
				result <- requestErr
			}()
			<-conn.writing
			if closeDuringWrite {
				_ = c.Close()
			}
			err = <-result
			if closeDuringWrite {
				if !errors.Is(err, ErrClosed) || !errors.Is(err, context.Canceled) {
					t.Fatalf("lost close/write cause: %v", err)
				}
			} else {
				if !errors.Is(err, ErrDisconnected) || !errors.Is(err, writeErr) {
					t.Fatalf("lost disconnect/write cause: %v", err)
				}
			}
		})
	}
}

func TestReadyNotificationOverflowCannotReportSuccessfulConnect(t *testing.T) {
	for i := 0; i < 20; i++ {
		s := newTestConnection()
		c, err := newClient(Config{BotID: "bot", Secret: "secret"}, withNotificationBuffer(2), withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
		if err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		go func() { result <- c.Connect(context.Background()) }()
		auth := s.next(t)
		s.ack(auth.Headers.ReqID, 0, nil)
		err = <-result
		_ = c.Close()
		if !errors.Is(err, ErrNotificationOverflow) {
			t.Fatalf("connect reported success after own overflow: %v", err)
		}
	}
}

func TestReconnectNotificationOverflowStopsBeforeBackoff(t *testing.T) {
	c, s := newTestClient(t, withNotificationBuffer(4), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Hour, MaxDelay: time.Hour, MaxAttempts: 1}))
	s.drop()
	select {
	case <-c.done:
		if !errors.Is(c.Err(), ErrNotificationOverflow) {
			t.Fatal(c.Err())
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("overflow kept backoff alive: %s", c.State())
	}
}

func TestRejectedHeartbeatRetainsCodeAndRequestID(t *testing.T) {
	c, s := newTestClient(t, withHeartbeat(5*time.Millisecond, time.Second), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Second, MaxDelay: time.Second, MaxAttempts: 0}))
	ping := s.next(t)
	s.ack(ping.Headers.ReqID, 40014, nil)
	waitState(t, c, StateClosed)
	var apiErr *coreerrors.Error
	if !errors.As(c.Err(), &apiErr) || apiErr.Code != "40014" || apiErr.RequestID != ping.Headers.ReqID {
		t.Fatalf("wrapped ACK lost metadata: %+v", apiErr)
	}
}
