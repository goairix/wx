package websocket

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

func readyLifecycleClient(t *testing.T, first Connection, next DialerFunc, retries int) (*Client, *atomic.Int32) {
	t.Helper()
	calls := new(atomic.Int32)
	c, err := New(Config{BotID: "bot", Secret: "secret", HeartbeatInterval: time.Hour, Reconnect: initialPolicy(retries), Dialer: DialerFunc(func(ctx context.Context, url string) (Connection, error) {
		if calls.Add(1) == 1 {
			return first, nil
		}
		return next(ctx, url)
	})})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, calls
}

func subscribeLifecycleClient(t *testing.T, c *Client, first *testConnection) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- c.Connect(context.Background()) }()
	auth := first.next(t)
	first.ack(auth.Headers.ReqID, 0, nil)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("initial subscription did not finish")
	}
}

func lifecycleError(t *testing.T, c *Client) error {
	t.Helper()
	select {
	case <-c.done:
		return c.Err()
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("lifecycle kept retrying: %s", c.State())
		return nil
	}
}

func assertConnectionFailure(t *testing.T, err error, phase ConnectionPhase, retryable bool, cause error) {
	t.Helper()
	var classified *ConnectionError
	if !errors.As(err, &classified) || classified.Phase != phase || classified.Retryable != retryable || cause != nil && !errors.Is(err, cause) {
		t.Fatalf("lost connection classification/chain: class=%+v err=%v", classified, err)
	}
}

func TestReadyReconnectPermanentDialFailureTerminates(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"certificate", errors.Join(io.EOF, &x509.UnknownAuthorityError{})},
		{"forbidden", &coreerrors.Error{Operation: "dial", HTTPStatus: 403, RequestID: "handshake", Err: io.EOF}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := newTestConnection()
			c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return nil, tc.err }, -1)
			subscribeLifecycleClient(t, c, first)
			first.drop()
			err := lifecycleError(t, c)
			assertConnectionFailure(t, err, PhaseDial, false, tc.err)
			if calls.Load() != 2 || errors.Is(err, ErrRetryExhausted) {
				t.Fatalf("permanent error consumed retry budget: calls=%d err=%v", calls.Load(), err)
			}
			if tc.name == "forbidden" {
				var apiErr *coreerrors.Error
				if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 403 || apiErr.RequestID != "handshake" || apiErr.Operation != "dial" {
					t.Fatalf("HTTP metadata lost: %+v", apiErr)
				}
			}
			found := false
			for n := range c.Notifications() {
				if errors.Is(n.Err, tc.err) {
					assertConnectionFailure(t, n.Err, PhaseDial, false, tc.err)
					found = true
				}
			}
			if !found {
				t.Fatal("permanent failure notification missing")
			}
		})
	}
}

func TestReadyInvalidJSONTerminatesWithReceivePhase(t *testing.T) {
	first := newTestConnection()
	c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return nil, io.EOF }, -1)
	subscribeLifecycleClient(t, c, first)
	first.push(`{"cmd":`)
	err := lifecycleError(t, c)
	assertConnectionFailure(t, err, PhaseReceive, false, ErrProtocol)
	var syntaxErr *json.SyntaxError
	if calls.Load() != 1 || !errors.As(err, &syntaxErr) || errors.Is(err, ErrRetryExhausted) {
		t.Fatalf("protocol failure retried or lost JSON chain: calls=%d err=%v", calls.Load(), err)
	}
}

func TestReadyTransientFailuresRetryWithClassifiedNotifications(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
	}{
		{"reset", syscall.ECONNRESET},
		{"temporary DNS", &net.DNSError{Err: "server misbehaving", Name: "example.invalid", IsTemporary: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second := newTestConnection(), newTestConnection()
			var retries atomic.Int32
			c, _ := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) {
				if retries.Add(1) == 1 {
					return nil, tc.cause
				}
				return second, nil
			}, 2)
			subscribeLifecycleClient(t, c, first)
			first.drop()
			auth := second.next(t)
			second.ack(auth.Headers.ReqID, 0, nil)
			waitState(t, c, StateReady)
			_ = c.Close()
			seenReceive, seenDial := false, false
			for n := range c.Notifications() {
				if errors.Is(n.Err, io.EOF) {
					assertConnectionFailure(t, n.Err, PhaseReceive, true, io.EOF)
					seenReceive = true
				}
				if errors.Is(n.Err, tc.cause) {
					assertConnectionFailure(t, n.Err, PhaseDial, true, tc.cause)
					seenDial = true
				}
			}
			if !seenReceive || !seenDial || retries.Load() != 2 {
				t.Fatalf("transient notifications missing: receive=%v dial=%v retries=%d", seenReceive, seenDial, retries.Load())
			}
		})
	}
}

func TestReadyReconnectExhaustionRetainsLastClassifiedCause(t *testing.T) {
	first := newTestConnection()
	last := &coreerrors.Error{Operation: "dial", HTTPStatus: 503, Err: syscall.ECONNREFUSED}
	c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return nil, last }, 2)
	subscribeLifecycleClient(t, c, first)
	first.drop()
	err := lifecycleError(t, c)
	if !errors.Is(err, ErrRetryExhausted) || calls.Load() != 3 {
		t.Fatalf("missing exhaustion or changed budget: calls=%d err=%v", calls.Load(), err)
	}
	assertConnectionFailure(t, err, PhaseDial, true, last)
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatal("last network cause lost")
	}
	var apiErr *coreerrors.Error
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 503 {
		t.Fatalf("last HTTP metadata lost: %+v", apiErr)
	}
}

func TestReadyReconnectSubscriptionFailuresAreClassified(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{true: "rejected", false: "transport exhausted"}[reject], func(t *testing.T) {
			first, second := newTestConnection(), newTestConnection()
			c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return second, nil }, 1)
			subscribeLifecycleClient(t, c, first)
			first.drop()
			auth := second.next(t)
			if reject {
				second.ack(auth.Headers.ReqID, 401, nil)
			} else {
				second.drop()
			}
			err := lifecycleError(t, c)
			assertConnectionFailure(t, err, PhaseSubscribe, !reject, nil)
			if calls.Load() != 2 || errors.Is(err, ErrRetryExhausted) == reject {
				t.Fatalf("wrong subscription recovery: calls=%d err=%v", calls.Load(), err)
			}
			if reject {
				var apiErr *coreerrors.Error
				if !errors.As(err, &apiErr) || apiErr.Code != "401" || apiErr.RequestID != auth.Headers.ReqID || apiErr.Operation != "work.aibot.websocket."+CmdSubscribe {
					t.Fatalf("subscription error metadata lost: %+v", apiErr)
				}
			} else if !errors.Is(err, io.EOF) {
				t.Fatal("subscription transport cause lost")
			}
		})
	}
}

func TestReadyReconnectTemporarySubscriptionFailureRecovers(t *testing.T) {
	first, second, third := newTestConnection(), newTestConnection(), newTestConnection()
	var retries atomic.Int32
	c, _ := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) {
		if retries.Add(1) == 1 {
			return second, nil
		}
		return third, nil
	}, 2)
	subscribeLifecycleClient(t, c, first)
	first.drop()
	second.next(t)
	second.drop()
	auth := third.next(t)
	third.ack(auth.Headers.ReqID, 0, nil)
	waitState(t, c, StateReady)
	_ = c.Close()
	found := false
	for n := range c.Notifications() {
		var classified *ConnectionError
		if errors.As(n.Err, &classified) && classified.Phase == PhaseSubscribe {
			assertConnectionFailure(t, n.Err, PhaseSubscribe, true, io.EOF)
			found = true
		}
	}
	if !found || retries.Load() != 2 {
		t.Fatalf("temporary subscription not recovered: notification=%v retries=%d", found, retries.Load())
	}
}

func TestReadyReconnectZeroBudgetIdentifiesReceiveExhaustion(t *testing.T) {
	first := newTestConnection()
	c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return nil, io.EOF }, 0)
	subscribeLifecycleClient(t, c, first)
	first.drop()
	err := lifecycleError(t, c)
	assertConnectionFailure(t, err, PhaseReceive, true, io.EOF)
	if !errors.Is(err, ErrRetryExhausted) || calls.Load() != 1 {
		t.Fatalf("zero budget contract violated: calls=%d err=%v", calls.Load(), err)
	}
}

func TestReplacementNotificationRetainsReceiveClassification(t *testing.T) {
	c, conn := newTestClient(t)
	conn.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"replaced"},"body":{"msgtype":"event","event":{"eventtype":"disconnected_event"}}}`)
	err := lifecycleError(t, c)
	assertConnectionFailure(t, err, PhaseReceive, false, ErrReplaced)
	for n := range c.Notifications() {
		if n.Event != nil {
			assertConnectionFailure(t, n.Err, PhaseReceive, false, ErrReplaced)
			var apiErr *coreerrors.Error
			if !errors.As(n.Err, &apiErr) || apiErr.RequestID != "replaced" {
				t.Fatal("replacement req_id lost")
			}
			return
		}
	}
	t.Fatal("replacement notification missing")
}

func TestCanceledBusinessRequestReconnectsWithoutReplaying(t *testing.T) {
	for _, cancelDuringWrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "awaiting ACK", true: "during write"}[cancelDuringWrite], func(t *testing.T) {
			first, second := newTestConnection(), newTestConnection()
			var conn Connection = first
			writing := make(chan struct{})
			if cancelDuringWrite {
				conn = &writeFailureConnection{testConnection: first, writing: writing, waitForCancel: true}
			}
			c, calls := readyLifecycleClient(t, conn, func(context.Context, string) (Connection, error) { return second, nil }, 1)
			subscribeLifecycleClient(t, c, first)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := c.Send(ctx, "chat", aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "uncertain delivery"}})
				done <- err
			}()
			if cancelDuringWrite {
				select {
				case <-writing:
				case <-time.After(time.Second):
					t.Fatal("request did not start writing")
				}
			} else {
				first.next(t)
			}
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatalf("caller cancellation lost: %v", err)
			}
			auth := second.next(t)
			if auth.Command != CmdSubscribe {
				t.Fatalf("delivery-ambiguous request replayed: %+v", auth)
			}
			second.ack(auth.Headers.ReqID, 0, nil)
			waitState(t, c, StateReady)
			_ = c.Close()
			found := false
			for n := range c.Notifications() {
				if errors.Is(n.Err, context.Canceled) {
					assertConnectionFailure(t, n.Err, PhaseReceive, true, context.Canceled)
					found = true
				}
			}
			if !found || calls.Load() != 2 || len(second.writes) != 0 {
				t.Fatalf("canceled operation recovery: found=%v calls=%d writes=%d", found, calls.Load(), len(second.writes))
			}
		})
	}
}

func TestWebSocketCloseClassifications(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		retry bool
	}{
		{"normal", coderws.CloseError{Code: coderws.StatusNormalClosure}, true},
		{"going away", coderws.CloseError{Code: coderws.StatusGoingAway}, true},
		{"restart", coderws.CloseError{Code: coderws.StatusServiceRestart}, true},
		{"try later pointer", &coderws.CloseError{Code: coderws.StatusTryAgainLater}, true},
		{"protocol", errors.Join(io.EOF, coderws.CloseError{Code: coderws.StatusProtocolError}), false},
		{"policy", errors.Join(io.EOF, &coderws.CloseError{Code: coderws.StatusPolicyViolation}), false},
		{"restart and TLS", errors.Join(coderws.CloseError{Code: coderws.StatusServiceRestart}, &x509.UnknownAuthorityError{}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryableConnectionError(tc.err); got != tc.retry {
				t.Fatalf("close classification retry=%v, want=%v", got, tc.retry)
			}
		})
	}
}

func TestLifecycleCancellationAfterReadyDoesNotReconnect(t *testing.T) {
	first := newTestConnection()
	c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return nil, io.EOF }, -1)
	subscribeLifecycleClient(t, c, first)
	c.cancel()
	err := lifecycleError(t, c)
	assertConnectionFailure(t, err, PhaseReceive, false, context.Canceled)
	if calls.Load() != 1 || errors.Is(err, ErrRetryExhausted) {
		t.Fatalf("lifecycle cancellation retried: calls=%d err=%v", calls.Load(), err)
	}
}

func TestCanceledRequestCannotOverridePermanentFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	requestCause := requestFailureCause(ctx, io.EOF)
	if !errors.Is(requestCause, context.Canceled) || !errors.Is(requestCause, io.EOF) {
		t.Fatal("canceled request lost cancellation or underlying transport cause")
	}
	for _, permanent := range []error{ErrProtocol, ErrClosed, ErrReplaced, &x509.UnknownAuthorityError{}, &coreerrors.Error{HTTPStatus: 403}} {
		if retryableConnectionError(errors.Join(requestCause, permanent)) {
			t.Fatalf("canceled request overrode permanent failure: %T", permanent)
		}
	}
}

// Model a transport that closes its socket on Write cancellation: Read returns
// first, then the canceled Write returns after the SDK has invalidated session.
type readFirstCanceledWriteConnection struct {
	*testConnection
	writing, readFailure chan struct{}
}

func (c *readFirstCanceledWriteConnection) Read(ctx context.Context) ([]byte, error) {
	select {
	case data := <-c.reads:
		return data, nil
	case <-c.readFailure:
		return nil, net.ErrClosed
	case <-c.closed:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *readFirstCanceledWriteConnection) Write(ctx context.Context, data []byte) error {
	var frame Frame
	if err := json.Unmarshal(data, &frame); err != nil {
		return err
	}
	if frame.Command == CmdSubscribe {
		return c.testConnection.Write(ctx, data)
	}
	close(c.writing)
	<-ctx.Done()
	close(c.readFailure)
	<-c.closed
	return ctx.Err()
}

func TestCanceledWriteReadFailureWinsStillReconnectsWithoutReplay(t *testing.T) {
	first := &readFirstCanceledWriteConnection{testConnection: newTestConnection(), writing: make(chan struct{}), readFailure: make(chan struct{})}
	second := newTestConnection()
	c, calls := readyLifecycleClient(t, first, func(context.Context, string) (Connection, error) { return second, nil }, 1)
	subscribeLifecycleClient(t, c, first.testConnection)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := c.Send(ctx, "chat", aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "uncertain delivery"}})
		done <- err
	}()
	select {
	case <-first.writing:
	case <-time.After(time.Second):
		t.Fatal("request did not start writing")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, net.ErrClosed) {
			t.Fatalf("caller lost cancellation or first transport cause: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled request stranded")
	}
	auth := second.next(t)
	if auth.Command != CmdSubscribe {
		t.Fatal("canceled request replayed")
	}
	second.ack(auth.Headers.ReqID, 0, nil)
	waitState(t, c, StateReady)
	_ = c.Close()
	found := false
	for n := range c.Notifications() {
		if errors.Is(n.Err, net.ErrClosed) {
			assertConnectionFailure(t, n.Err, PhaseReceive, true, net.ErrClosed)
			found = true
		}
	}
	if !found || calls.Load() != 2 || len(second.writes) != 0 {
		t.Fatalf("read-before-write-return recovery: found=%v calls=%d writes=%d", found, calls.Load(), len(second.writes))
	}
}
