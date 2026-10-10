package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

func TestRawCardNumbersSurviveEveryPublicSendPath(t *testing.T) {
	for _, mode := range []string{"send", "reply", "update"} {
		t.Run(mode, func(t *testing.T) {
			c, conn := newTestClient(t)
			card := aibot.RawTemplateCard(`{"card_type":"button_interaction","task_id":"input-task","extension":{"opaque_id":9007199254740993,"nested":[9007199254740995,0,false],"huge":1e500}}`)
			m := aibot.OutgoingMessage{MsgType: "template_card", TemplateCard: card}
			var target *ReceivedCallback
			if mode != "send" {
				if mode == "update" {
					conn.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"callback"},"body":{"msgtype":"event","event":{"eventtype":"template_card_event","template_card_event":{"task_id":"original-task"}}}}`)
				} else {
					conn.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"callback"},"body":{"msgtype":"text"}}`)
				}
				for n := range c.Notifications() {
					if n.Callback != nil {
						target = n.Callback
						break
					}
				}
			}
			done := make(chan error, 1)
			go func() {
				var err error
				if mode == "send" {
					_, err = c.Send(context.Background(), "chat", m)
				} else {
					_, err = c.Reply(context.Background(), target, &aibot.Reply{OutgoingMessage: m})
				}
				done <- err
			}()
			var frame Frame
			select {
			case b := <-conn.writes:
				if err := json.Unmarshal(b, &frame); err != nil {
					t.Fatal(err)
				}
			case err := <-done:
				t.Fatalf("valid raw card rejected: %v", err)
			case <-time.After(time.Second):
				t.Fatal("no card written")
			}
			var body struct {
				TemplateCard struct {
					TaskID    string          `json:"task_id"`
					Extension json.RawMessage `json:"extension"`
				} `json:"template_card"`
			}
			if err := json.Unmarshal(frame.Body, &body); err != nil {
				t.Fatal(err)
			}
			var extension map[string]json.RawMessage
			if err := json.Unmarshal(body.TemplateCard.Extension, &extension); err != nil {
				t.Fatal(err)
			}
			if string(extension["opaque_id"]) != "9007199254740993" || string(extension["nested"]) != "[9007199254740995,0,false]" || string(extension["huge"]) != "1e500" {
				t.Fatalf("raw card changed: %s", frame.Body)
			}
			wantTask := "input-task"
			if mode == "update" {
				wantTask = "original-task"
			}
			if body.TemplateCard.TaskID != wantTask {
				t.Fatalf("task_id = %q", body.TemplateCard.TaskID)
			}
			conn.ack(frame.Headers.ReqID, 0, nil)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

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
