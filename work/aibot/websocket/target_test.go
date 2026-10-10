package websocket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

func TestReplyTargetSnapshotAndSessionIsolation(t *testing.T) {
	c, s := newTestClient(t)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"original"},"body":{"msgtype":"text","text":{"content":"hi"}}}`)
	var target *ReceivedCallback
	for n := range c.Notifications() {
		if n.Callback != nil {
			target = n.Callback
			break
		}
	}
	target.Frame.Headers.ReqID = "mutated"
	target.Callback.Message.MsgType = "event"
	reply := &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "hello"}}}
	done := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, reply); done <- err }()
	f := s.next(t)
	if f.Headers.ReqID != "original" || f.Command != CmdRespond {
		t.Fatalf("%+v", f)
	}
	s.ack("original", 0, nil)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	other, _ := newTestClient(t)
	if _, err := other.Reply(context.Background(), target, reply); !errors.Is(err, aibot.ErrInvalidTarget) {
		t.Fatal(err)
	}
	if _, err := c.Reply(context.Background(), &ReceivedCallback{}, reply); !errors.Is(err, aibot.ErrInvalidTarget) {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Reply(context.Background(), target, reply); !errors.Is(err, aibot.ErrInvalidTarget) {
		t.Fatal(err)
	}
}
func TestTransportRejectsUnsupportedReplyBeforeWrite(t *testing.T) {
	c, s := newTestClient(t)
	s.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"r"},"body":{"msgtype":"text"}}`)
	var target *ReceivedCallback
	for n := range c.Notifications() {
		if n.Callback != nil {
			target = n.Callback
			break
		}
	}
	_, err := c.Reply(context.Background(), target, &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "stream_with_template_card", Stream: &aibot.Stream{ID: "s", Content: "text"}}})
	if !errors.Is(err, aibot.ErrUnsupported) {
		t.Fatal(err)
	}
	select {
	case <-s.writes:
		t.Fatal("unsupported reply written")
	default:
	}
}

func TestOldCallbackCannotReplyAfterReconnect(t *testing.T) {
	first, second := newTestConnection(), newTestConnection()
	calls := 0
	c, err := newClient(Config{BotID: "b", Secret: "s"}, withHeartbeat(time.Hour, time.Second), withReconnectPolicy(ReconnectPolicy{InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, MaxAttempts: 1}), withDialer(DialerFunc(func(context.Context, string) (Connection, error) {
		calls++
		if calls == 1 {
			return first, nil
		}
		return second, nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ready := make(chan error, 1)
	go func() { ready <- c.Connect(context.Background()) }()
	auth := first.next(t)
	first.ack(auth.Headers.ReqID, 0, nil)
	if connectErr := <-ready; connectErr != nil {
		t.Fatal(connectErr)
	}
	first.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"same"},"body":{"msgtype":"text"}}`)
	var old *ReceivedCallback
	for n := range c.Notifications() {
		if n.Callback != nil {
			old = n.Callback
			break
		}
	}
	first.drop()
	auth = second.next(t)
	second.ack(auth.Headers.ReqID, 0, nil)
	waitState(t, c, StateReady)
	_, err = c.Reply(context.Background(), old, &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "late"}}})
	if !errors.Is(err, aibot.ErrInvalidTarget) {
		t.Fatal(err)
	}
	select {
	case <-second.writes:
		t.Fatal("old reply written in new session")
	default:
	}
}
