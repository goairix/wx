package websocket

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/url"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

func TestInitialConnectRetriesSubscriptionTransportFailure(t *testing.T) {
	for i := 0; i < 20; i++ {
		first, second := newTestConnection(), newTestConnection()
		var calls atomic.Int32
		c, err := New(Config{BotID: "b", Secret: "s", InitialConnect: initialPolicy(1), Dialer: DialerFunc(func(context.Context, string) (Connection, error) {
			if calls.Add(1) == 1 {
				return first, nil
			}
			return second, nil
		})}, aibot.WithRequestTimeout(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		go func() { result <- c.Connect(context.Background()) }()
		first.next(t)
		first.drop()
		select {
		case err = <-result:
			_ = c.Close()
			t.Fatalf("temporary subscription disconnect was terminal: %v", err)
		case bytes := <-second.writes:
			var frame Frame
			if err = json.Unmarshal(bytes, &frame); err != nil {
				t.Fatal(err)
			}
			second.ack(frame.Headers.ReqID, 0, nil)
		case <-time.After(time.Second):
			_ = c.Close()
			t.Fatal("subscription transport failure not retried")
		}
		if err = <-result; err != nil {
			t.Fatal(err)
		}
		_ = c.Close()
	}
}

func TestInitialConnectRetriesSubscriptionAttemptTimeout(t *testing.T) {
	first, second := newTestConnection(), newTestConnection()
	var calls atomic.Int32
	c, err := New(Config{BotID: "b", Secret: "s", InitialConnect: initialPolicy(1), Dialer: DialerFunc(func(context.Context, string) (Connection, error) {
		if calls.Add(1) == 1 {
			return first, nil
		}
		return second, nil
	})}, aibot.WithRequestTimeout(80*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	first.next(t)
	frame := second.next(t)
	second.ack(frame.Headers.ReqID, 0, nil)
	if err = <-result; err != nil || calls.Load() != 2 {
		t.Fatalf("subscription timeout not retried: %v", err)
	}
}

func initialPolicy(attempts int) *ReconnectPolicy {
	return &ReconnectPolicy{InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, MaxAttempts: attempts}
}

type failAfterWriteConnection struct {
	*testConnection
	fail func()
}

func (c failAfterWriteConnection) Write(context.Context, []byte) error { c.fail(); return nil }
func TestSubscribeWaitPreservesTransportCauseOnSessionCancellation(t *testing.T) {
	for i := 0; i < 100; i++ {
		owner, err := newClient(Config{BotID: "b", Secret: "s"})
		if err != nil {
			t.Fatal(err)
		}
		conn := failAfterWriteConnection{testConnection: newTestConnection()}
		s := newSession(context.Background(), conn, owner)
		conn.fail = func() { s.fail(io.EOF) }
		s.conn = conn
		ctx, cancel := context.WithTimeout(s.ctx, time.Second)
		_, err = s.request(ctx, CmdSubscribe, "subscribe", nil)
		cancel()
		if !errors.Is(err, io.EOF) {
			t.Fatalf("session cause lost to child cancellation: %v", err)
		}
	}
}

func TestSubscribeBeforeWritePreservesTransportCauseOnSessionCancellation(t *testing.T) {
	owner, err := newClient(Config{BotID: "b", Secret: "s"})
	if err != nil {
		t.Fatal(err)
	}
	s := newSession(context.Background(), newTestConnection(), owner)
	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()
	s.fail(io.EOF)
	_, err = s.request(ctx, CmdSubscribe, "subscribe", nil)
	if !errors.Is(err, io.EOF) || !retryableConnectionError(err) {
		t.Fatalf("pre-write failure lost transport cause: %v", err)
	}
}

func TestInitialConnectRetriesTemporaryDialFailure(t *testing.T) {
	conn := newTestConnection()
	var calls atomic.Int32
	c, err := New(Config{BotID: "bot", Secret: "secret", InitialConnect: initialPolicy(2), Dialer: DialerFunc(func(context.Context, string) (Connection, error) {
		if calls.Add(1) == 1 {
			return nil, &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
		}
		return conn, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	auth := conn.next(t)
	conn.ack(auth.Headers.ReqID, 0, nil)
	if err = <-result; err != nil || calls.Load() != 2 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
	found := false
	for len(c.notices) > 0 {
		n := <-c.notices
		if n.State == StateReconnecting && n.Attempt == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("missing initial retry notification")
	}
}

func TestInitialConnectClassifiesPermanentFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
		retry bool
	}{
		{"eof", io.EOF, true}, {"reset", syscall.ECONNRESET, true}, {"timeout", context.DeadlineExceeded, true},
		{"closed transport", net.ErrClosed, true},
		{"closed transport with protocol error", errors.Join(net.ErrClosed, ErrProtocol), false},
		{"closed transport with certificate error", errors.Join(net.ErrClosed, &x509.UnknownAuthorityError{}), false},
		{"closed transport with forbidden response", &coreerrors.Error{HTTPStatus: 403, Err: net.ErrClosed}, false},
		{"unknown", errors.New("custom dial failure"), false}, {"disconnected only", ErrDisconnected, false},
		{"joined protocol", errors.Join(io.EOF, ErrProtocol), false},
		{"temporary DNS", &net.DNSError{Err: "server misbehaving", Name: "example.invalid", IsTemporary: true}, true},
		{"permanent DNS", &net.DNSError{Err: "no such host", Name: "example.invalid", IsNotFound: true}, false},
		{"joined canceled", errors.Join(io.EOF, context.Canceled), false},
		{"certificate", &url.Error{Op: "Get", URL: "https://example.invalid", Err: x509.UnknownAuthorityError{}}, false},
		{"joined certificate", errors.Join(io.EOF, x509.UnknownAuthorityError{}), false},
		{"joined pointer authority", errors.Join(io.EOF, &x509.UnknownAuthorityError{}), false},
		{"joined pointer hostname", errors.Join(io.EOF, &x509.HostnameError{}), false},
		{"joined pointer certificate", errors.Join(io.EOF, &x509.CertificateInvalidError{}), false},
		{"forbidden", &coreerrors.Error{HTTPStatus: 403, Err: context.DeadlineExceeded}, false},
		{"not found", &coreerrors.Error{HTTPStatus: 404, Err: io.EOF}, false},
		{"server unavailable", &coreerrors.Error{HTTPStatus: 503, Err: errors.New("handshake")}, true},
		{"rate limit", &coreerrors.Error{HTTPStatus: 429, Err: errors.New("handshake")}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			c, err := New(Config{BotID: "b", Secret: "s", InitialConnect: initialPolicy(1), Dialer: DialerFunc(func(context.Context, string) (Connection, error) { calls.Add(1); return nil, tc.cause })})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			err = c.Connect(context.Background())
			var failure *ConnectionError
			if !errors.As(err, &failure) || failure.Retryable != tc.retry || failure.Phase != PhaseDial || !errors.Is(err, tc.cause) {
				t.Fatalf("lost classification: %+v %v", failure, err)
			}
			want := int32(1)
			if tc.retry {
				want = 2
				if !errors.Is(err, ErrRetryExhausted) {
					t.Fatal("missing exhausted error")
				}
			}
			if calls.Load() != want {
				t.Fatalf("calls %d, want %d", calls.Load(), want)
			}
		})
	}
}

func TestInitialConnectNilPolicyAndAuthRejectionDoNotRetry(t *testing.T) {
	var calls atomic.Int32
	c, err := New(Config{BotID: "b", Secret: "s", Dialer: DialerFunc(func(context.Context, string) (Connection, error) { calls.Add(1); return nil, io.EOF })})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Connect(context.Background()); err == nil || calls.Load() != 1 {
		t.Fatal("nil policy retried")
	}
	_ = c.Close()
	conn := newTestConnection()
	calls.Store(0)
	c, err = New(Config{BotID: "b", Secret: "s", InitialConnect: initialPolicy(-1), Dialer: DialerFunc(func(context.Context, string) (Connection, error) { calls.Add(1); return conn, nil })})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	auth := conn.next(t)
	conn.ack(auth.Headers.ReqID, 401, nil)
	err = <-result
	var failure *ConnectionError
	var apiErr *coreerrors.Error
	if calls.Load() != 1 || !errors.As(err, &failure) || failure.Retryable || failure.Phase != PhaseSubscribe || !errors.As(err, &apiErr) || apiErr.Code != "401" || apiErr.RequestID != auth.Headers.ReqID {
		t.Fatalf("bad auth rejection: %v %+v", err, failure)
	}
}

func TestInitialConnectCancelAndCloseInterruptBackoff(t *testing.T) {
	for _, closeClient := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "close"}[closeClient], func(t *testing.T) {
			attempted := make(chan struct{})
			var calls atomic.Int32
			c, err := New(Config{BotID: "b", Secret: "s", InitialConnect: &ReconnectPolicy{InitialDelay: time.Hour, MaxDelay: time.Hour, MaxAttempts: -1}, Dialer: DialerFunc(func(context.Context, string) (Connection, error) {
				if calls.Add(1) == 1 {
					close(attempted)
				}
				return nil, io.EOF
			})})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- c.Connect(ctx) }()
			<-attempted
			waitState(t, c, StateReconnecting)
			if closeClient {
				_ = c.Close()
			} else {
				cancel()
			}
			select {
			case err = <-result:
				if err == nil {
					t.Fatal("accepted canceled connection")
				}
			case <-time.After(time.Second):
				t.Fatal("backoff not interrupted")
			}
			if calls.Load() != 1 {
				t.Fatal("dial after lifecycle ended")
			}
		})
	}
}
