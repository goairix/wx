package aibot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	coreerrors "github.com/goairix/wx/v2/core/errors"
)

func TestDefaultDialerAuthenticatesAndReceivesRealWebSocket(t *testing.T) {
	received := make(chan Frame, 4)
	serverErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := websocket.Accept(w, r, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		defer func() { _ = socket.CloseNow() }()
		for {
			_, data, err := socket.Read(r.Context())
			if err != nil {
				return
			}
			var frame Frame
			if err := json.Unmarshal(data, &frame); err != nil {
				serverErrors <- err
				return
			}
			received <- frame
			ack, _ := json.Marshal(map[string]any{"headers": map[string]string{"req_id": frame.Headers.ReqID}, "errcode": 0, "errmsg": "ok"})
			if err := socket.Write(r.Context(), websocket.MessageText, ack); err != nil {
				serverErrors <- err
				return
			}
			if frame.Command == CmdSubscribe {
				if err := socket.Write(r.Context(), websocket.MessageText, []byte(`{"cmd":"aibot_msg_callback","headers":{"req_id":"wire-callback"},"body":{"msgid":"wire-message","msgtype":"text","text":{"content":"real wire"}}}`)); err != nil {
					serverErrors <- err
					return
				}
			}
		}
	}))
	defer server.Close()
	c, err := New(Config{BotID: "wire-bot", Secret: "wire-secret"}, WithWebSocketURL("ws"+strings.TrimPrefix(server.URL, "http")), WithHTTPClient(server.Client()), WithHeartbeat(time.Hour, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	auth := <-received
	if auth.Command != CmdSubscribe {
		t.Fatal(auth)
	}
	for {
		select {
		case notice := <-c.Notifications():
			if notice.Message != nil {
				if notice.Message.Text.Content != "real wire" {
					t.Fatal(notice)
				}
				goto reply
			}
		case err := <-serverErrors:
			t.Fatal(err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
reply:
	if _, err := c.ReplyMarkdown(ctx, "wire-callback", "response"); err != nil {
		t.Fatal(err)
	}
	message := <-received
	if message.Command != CmdRespond || message.Headers.ReqID != "wire-callback" {
		t.Fatal(message)
	}
}
func TestDialRespectsRequestTimeout(t *testing.T) {
	c, err := New(Config{BotID: "bot", Secret: "secret"}, WithRequestTimeout(15*time.Millisecond), WithDialer(DialerFunc(func(ctx context.Context, _ string) (Connection, error) { <-ctx.Done(); return nil, ctx.Err() })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result := make(chan error, 1)
	go func() { result <- c.Connect(context.Background()) }()
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("dial not bounded by timeout")
	}
}
func TestCloseCancelsSubscribeAndCannotRestart(t *testing.T) {
	s := newTestConnection()
	c, err := New(Config{BotID: "bot", Secret: "secret"}, WithDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.Connect(context.Background()) }()
	s.next(t)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := c.Connect(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
func TestSubscribeRejectionPreservesRemoteError(t *testing.T) {
	s := newTestConnection()
	c, err := New(Config{BotID: "bot", Secret: "secret"}, WithDialer(DialerFunc(func(context.Context, string) (Connection, error) { return s, nil })))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	done := make(chan error, 1)
	go func() { done <- c.Connect(context.Background()) }()
	frame := s.next(t)
	s.ack(frame.Headers.ReqID, 401, nil)
	err = <-done
	var e *coreerrors.Error
	if !errors.As(err, &e) || e.Code != "401" || e.RequestID != frame.Headers.ReqID || e.Operation != "aibot.aibot_subscribe" {
		t.Fatalf("%+v", e)
	}
	waitState(t, c, StateClosed)
}
func TestConfigRejectsInvalidDependenciesAndLimits(t *testing.T) {
	cases := []Option{WithWebSocketURL("https://example.com"), WithHTTPClient(nil), WithLogger(nil), WithChunkSize(512<<10 + 1), WithMediaLimits(0, 1), WithHeartbeat(0, time.Second), WithRequestTimeout(0), WithNotificationBuffer(0), WithMaxFrameBytes(0), WithReconnectPolicy(ReconnectPolicy{InitialDelay: time.Second, MaxDelay: time.Millisecond})}
	for _, option := range cases {
		if _, err := New(Config{BotID: "bot", Secret: "secret"}, option); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("invalidconfiguration accepted: %v", err)
		}
	}
	if _, err := New(Config{}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatal(err)
	}
}
