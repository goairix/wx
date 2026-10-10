package websocket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

type deadlineCaptureConnection struct {
	*testConnection
	deadlines chan time.Time
}

func (c *deadlineCaptureConnection) Write(ctx context.Context, data []byte) error {
	deadline, _ := ctx.Deadline()
	c.deadlines <- deadline
	return c.testConnection.Write(ctx, data)
}

func TestFirstStreamWriteUsesStreamDeadlineWithLongRequestBudget(t *testing.T) {
	conn := &deadlineCaptureConnection{testConnection: newTestConnection(), deadlines: make(chan time.Time, 4)}
	c, err := New(Config{BotID: "bot", Secret: "secret", HeartbeatInterval: time.Hour, Dialer: DialerFunc(func(context.Context, string) (Connection, error) { return conn, nil })}, aibot.WithRequestTimeout(20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	done := make(chan error, 1)
	go func() { done <- c.Connect(context.Background()) }()
	auth := conn.next(t)
	<-conn.deadlines
	conn.ack(auth.Headers.ReqID, 0, nil)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	target := callbackTarget(t, c, conn.testConnection, "")
	go func() {
		_, replyErr := c.Reply(context.Background(), target, streamReply("stream", false))
		done <- replyErr
	}()
	frame := conn.next(t)
	deadline := <-conn.deadlines
	target.state.mu.Lock()
	started := target.state.streams["stream"].started
	target.state.mu.Unlock()
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if want := started.Add(10 * time.Minute); !deadline.Equal(want) {
		t.Fatalf("first stream Write deadline %v, want %v", deadline, want)
	}
}

func callbackTarget(t *testing.T, c *Client, conn *testConnection, event string) *ReceivedCallback {
	t.Helper()
	if event == "" {
		conn.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"callback"},"body":{"msgtype":"text","chattype":"single","from":{"userid":"user"},"text":{"content":"hello"}}}`)
	} else {
		conn.push(`{"cmd":"aibot_event_callback","headers":{"req_id":"callback"},"body":{"msgtype":"event","chattype":"group","chatid":"group","event":{"eventtype":"` + event + `","template_card_event":{"task_id":"task"}}}}`)
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case n := <-c.Notifications():
			if n.Callback != nil {
				return n.Callback
			}
		case <-timer.C:
			t.Fatal("missing callback")
			return nil
		}
	}
}
func shortReplyDeadline(target *ReceivedCallback, deadline time.Time) {
	target.state.mu.Lock()
	target.state.deadline = deadline
	target.state.mu.Unlock()
}
func markdownReply() *aibot.Reply {
	return &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "reply"}}}
}
func streamReply(id string, finish bool) *aibot.Reply {
	return &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "stream", Stream: &aibot.Stream{ID: id, Content: "full snapshot", Finish: finish}}}
}

func TestReplyDeadlineSnapshotAndNotificationDelay(t *testing.T) {
	for _, event := range []string{"", "enter_chat", "template_card_event", "feedback_event"} {
		t.Run(event, func(t *testing.T) {
			c, conn := newTestClient(t)
			before := time.Now()
			target := callbackTarget(t, c, conn, event)
			after := time.Now()
			if target.ReceivedAt().Before(before) || target.ReceivedAt().After(after) {
				t.Fatal("invalid receipt time")
			}
			want := 24 * time.Hour
			if event != "" {
				want = 5 * time.Second
			}
			if event == "feedback_event" {
				if !target.ReplyDeadline().IsZero() {
					t.Fatal("unsupported event has deadline")
				}
				return
			}
			if target.ReplyDeadline().Sub(target.ReceivedAt()) != want {
				t.Fatal("invalid reply window")
			}
			original := target.ReplyDeadline()
			target.Callback = &aibot.Callback{}
			target.Frame = Frame{}
			if target.ReplyDeadline() != original {
				t.Fatal("public metadata changed deadline")
			}
		})
	}
	c, conn := newTestClient(t)
	target := callbackTarget(t, c, conn, "")
	shortReplyDeadline(target, time.Now().Add(-time.Second))
	if _, err := c.Reply(context.Background(), target, markdownReply()); !errors.Is(err, aibot.ErrReplyExpired) {
		t.Fatalf("expired target accepted: %v", err)
	}
	if c.State() != StateReady {
		t.Fatal("expiry closed connection")
	}
	select {
	case <-conn.writes:
		t.Fatal("expired reply sent")
	default:
	}
}

func TestReplyExpiryWhileWaitingForWriterDoesNotClose(t *testing.T) {
	c, conn := newTestClient(t)
	target := callbackTarget(t, c, conn, "")
	shortReplyDeadline(target, time.Now().Add(40*time.Millisecond))
	target.session.writer <- struct{}{}
	_, err := c.Reply(context.Background(), target, markdownReply())
	<-target.session.writer
	if !errors.Is(err, aibot.ErrReplyExpired) || c.State() != StateReady {
		t.Fatalf("err %v state %s", err, c.State())
	}
	select {
	case <-conn.writes:
		t.Fatal("expired queued reply sent")
	default:
	}
}
func TestOnTimeReplyACKCanArriveAfterProtocolDeadline(t *testing.T) {
	c, conn := newTestClient(t)
	target := callbackTarget(t, c, conn, "")
	shortReplyDeadline(target, time.Now().Add(60*time.Millisecond))
	done := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, markdownReply()); done <- err }()
	frame := conn.next(t)
	time.Sleep(90 * time.Millisecond)
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-done; err != nil || c.State() != StateReady {
		t.Fatalf("on-time reply lost ACK: %v state %s", err, c.State())
	}
}
func TestQueuedSameRequestReplyExpiresWithoutDisconnectingHolder(t *testing.T) {
	c, conn := newTestClient(t)
	target := callbackTarget(t, c, conn, "")
	shortReplyDeadline(target, time.Now().Add(50*time.Millisecond))
	first := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, markdownReply()); first <- err }()
	frame := conn.next(t)
	_, err := c.Reply(context.Background(), target, markdownReply())
	if !errors.Is(err, aibot.ErrReplyExpired) {
		t.Fatal(err)
	}
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err = <-first; err != nil || c.State() != StateReady {
		t.Fatalf("holder canceled: %v", err)
	}
	select {
	case <-conn.writes:
		t.Fatal("expired waiter sent")
	default:
	}
}

func TestStreamStateSharedAcrossCallbackCopies(t *testing.T) {
	c, conn := newTestClient(t)
	target := callbackTarget(t, c, conn, "")
	copyTarget := *target
	done := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, streamReply("stream", true)); done <- err }()
	frame := conn.next(t)
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := c.Reply(context.Background(), &copyTarget, streamReply("stream", false)); !errors.Is(err, aibot.ErrStreamFinished) {
		t.Fatalf("copied target revived flow: %v", err)
	}
	go func() { _, err := c.Reply(context.Background(), target, streamReply("another", false)); done <- err }()
	frame = conn.next(t)
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-done; err != nil {
		t.Fatal("finish prohibited independent stream")
	}
}
func TestStreamDeadlineStartsAtWriteAndRejectedFinishCanRetry(t *testing.T) {
	c, conn := newTestClient(t)
	target := callbackTarget(t, c, conn, "")
	before := time.Now()
	done := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, streamReply("stream", true)); done <- err }()
	frame := conn.next(t)
	target.state.mu.Lock()
	started := target.state.streams["stream"].started
	target.state.mu.Unlock()
	if started.Before(before) || started.After(time.Now()) {
		t.Fatal("stream not timed from Write")
	}
	conn.ack(frame.Headers.ReqID, 400, nil)
	if err := <-done; err == nil {
		t.Fatal("rejected finish accepted")
	}
	go func() { _, err := c.Reply(context.Background(), target, streamReply("stream", false)); done <- err }()
	frame = conn.next(t)
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-done; err != nil {
		t.Fatal("rejected finish latched state")
	}
	target.state.mu.Lock()
	target.state.streams["stream"].started = time.Now().Add(-10 * time.Minute)
	target.state.mu.Unlock()
	if _, err := c.Reply(context.Background(), target, streamReply("stream", true)); !errors.Is(err, aibot.ErrStreamExpired) {
		t.Fatalf("overdue stream sent: %v", err)
	}
	if c.State() != StateReady {
		t.Fatal("stream expiration closed connection")
	}
}
