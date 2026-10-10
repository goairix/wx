package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

type blockedSendConnection struct {
	*testConnection
	started chan struct{}
	release chan struct{}
}

func (c *blockedSendConnection) Write(ctx context.Context, data []byte) error {
	var frame Frame
	if err := json.Unmarshal(data, &frame); err != nil {
		return err
	}
	if frame.Command == CmdSend {
		close(c.started)
		select {
		case <-c.release:
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return context.Canceled
		}
	}
	return c.testConnection.Write(ctx, data)
}

func TestHeartbeatDoesNotExpireWhileQueuedBehindValidWrite(t *testing.T) {
	conn := &blockedSendConnection{testConnection: newTestConnection(), started: make(chan struct{}), release: make(chan struct{})}
	c, err := New(Config{BotID: "bot", Secret: "secret", Dialer: DialerFunc(func(context.Context, string) (Connection, error) { return conn, nil }), HeartbeatInterval: 20 * time.Millisecond, HeartbeatTimeout: 40 * time.Millisecond, Reconnect: &ReconnectPolicy{InitialDelay: time.Second, MaxDelay: time.Second, MaxAttempts: 0}}, aibot.WithRequestTimeout(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	connected := make(chan error, 1)
	go func() { connected <- c.Connect(context.Background()) }()
	subscribe := conn.next(t)
	conn.ack(subscribe.Headers.ReqID, 0, nil)
	if err := <-connected; err != nil {
		t.Fatal(err)
	}
	sent := make(chan error, 1)
	go func() {
		_, sendErr := c.Send(context.Background(), "test-chat", aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "test"}})
		sent <- sendErr
	}()
	select {
	case <-conn.started:
	case <-time.After(time.Second):
		t.Fatal("send did not start")
	}
	// The send is within its one-second budget; no ping can reach the wire yet.
	select {
	case sendErr := <-sent:
		t.Fatalf("heartbeat canceled valid queued write: %v", sendErr)
	case <-time.After(150 * time.Millisecond):
	}
	if c.State() != StateReady {
		t.Fatalf("unsent ping closed session: %s %v", c.State(), c.Err())
	}
	close(conn.release)
	frame := conn.next(t)
	if frame.Command != CmdSend {
		t.Fatalf("expected send, got %s", frame.Command)
	}
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	ping := conn.next(t)
	if ping.Command != CmdPing {
		t.Fatalf("expected ping, got %s", ping.Command)
	}
	conn.ack(ping.Headers.ReqID, 0, nil)
}
