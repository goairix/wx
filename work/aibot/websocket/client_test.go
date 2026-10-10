package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type testConnection struct {
	reads  chan []byte
	writes chan []byte
	closed chan struct{}
	once   sync.Once
}

func newTestConnection() *testConnection {
	return &testConnection{reads: make(chan []byte, 256), writes: make(chan []byte, 256), closed: make(chan struct{})}
}
func (s *testConnection) Read(ctx context.Context) ([]byte, error) {
	select {
	case b := <-s.reads:
		return b, nil
	case <-s.closed:
		return nil, io.EOF
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (s *testConnection) Write(ctx context.Context, b []byte) error {
	select {
	case <-s.closed:
		return io.EOF
	default:
	}
	select {
	case s.writes <- append([]byte(nil), b...):
		return nil
	case <-s.closed:
		return io.EOF
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *testConnection) Close() error { s.once.Do(func() { close(s.closed) }); return nil }
func (s *testConnection) next(t *testing.T) Frame {
	t.Helper()
	select {
	case b := <-s.writes:
		var f Frame
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatal(err)
		}
		return f
	case <-time.After(time.Second):
		t.Fatal("no outbound frame")
		return Frame{}
	}
}
func (s *testConnection) ack(reqID string, code int, body any) {
	b, _ := json.Marshal(map[string]any{"headers": map[string]string{"req_id": reqID}, "errcode": code, "errmsg": "test ACK", "body": body})
	s.reads <- b
}
func (s *testConnection) push(b string) { s.reads <- []byte(b) }
func (s *testConnection) drop()         { _ = s.Close() }
func newTestClient(t *testing.T, opts ...option) (*Client, *testConnection) {
	t.Helper()
	s := newTestConnection()
	opts = append([]option{withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })), withHeartbeat(time.Hour, time.Second)}, opts...)
	c, err := newClient(Config{BotID: "bot", Secret: "secret"}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(func() { _ = c.Close() })
	ready := make(chan error, 1)
	go func() { ready <- c.Connect(ctx) }()
	f := s.next(t)
	if f.Command != "aibot_subscribe" {
		t.Fatalf("subscribe: %+v", f)
	}
	s.ack(f.Headers.ReqID, 0, nil)
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	return c, s
}
func TestSubscribeRequiresACKAndCredentials(t *testing.T) {
	s := newTestConnection()
	c, err := newClient(Config{BotID: "bot-1", Secret: "sensitive"}, withDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	f := s.next(t)
	var body map[string]string
	if err := json.Unmarshal(f.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["bot_id"] != "bot-1" || body["secret"] != "sensitive" {
		t.Fatalf("credentials: %v", body)
	}
	select {
	case err := <-result:
		t.Fatalf("ready before ACK: %v", err)
	default:
	}
	if c.State() == StateReady {
		t.Fatal("ready before auth")
	}
	s.ack(f.Headers.ReqID, 0, nil)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if c.State() != StateReady {
		t.Fatal(c.State())
	}
}
func TestCloseUnblocksPendingRequest(t *testing.T) {
	c, s := newTestClient(t)
	done := make(chan error, 1)
	go func() {
		_, err := c.request(context.Background(), CmdRespond, "reply", map[string]string{"msgtype": "markdown"})
		done <- err
	}()
	s.next(t)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("pending request stranded")
	}
}
func TestCallbackWithPendingRequestIDIsNotACK(t *testing.T) {
	c, s := newTestClient(t)
	done := make(chan error, 1)
	go func() { _, err := c.request(context.Background(), CmdRespond, "same", nil); done <- err }()
	s.next(t)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"same"},"body":{"msgid":"m1","msgtype":"text","text":{"content":"hello"}}}`)
	var got bool
	for !got {
		select {
		case n := <-c.Notifications():
			if n.Message != nil {
				got = true
				if n.Message.Text.Content != "hello" || len(n.Frame.Raw) == 0 {
					t.Fatal(n)
				}
			}
		case <-time.After(time.Second):
			t.Fatal("callback missing")
		}
	}
	select {
	case err := <-done:
		t.Fatalf("push was ACK: %v", err)
	default:
	}
	s.ack("same", 0, nil)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
