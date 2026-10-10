package websocket

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

type recordingLimiter struct {
	mu       sync.Mutex
	requests []LimitRequest
	err      error
}

func (r *recordingLimiter) Allow(ctx context.Context, request LimitRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, request)
	return r.err
}
func (r *recordingLimiter) all() []LimitRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]LimitRequest(nil), r.requests...)
}
func withTestLimiter(l Limiter) option { return func(o *options) { o.limiter = l } }

func TestLimiterCoversAllPublicMessagePaths(t *testing.T) {
	for _, mode := range []string{"send", "reply", "welcome", "update", "stream"} {
		t.Run(mode, func(t *testing.T) {
			limiter := new(recordingLimiter)
			c, conn := newTestClient(t, withTestLimiter(limiter))
			var target *ReceivedCallback
			reply := markdownReply()
			wantID, wantType, wantOp := "user", "single", CmdRespond
			switch mode {
			case "send":
				wantID, wantType, wantOp = "active", "group", CmdSend
			case "welcome":
				target = callbackTarget(t, c, conn, "enter_chat")
				reply = &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "text", Text: &aibot.TextContent{Content: "hi"}}}
				wantID, wantType, wantOp = "group", "group", CmdWelcome
			case "update":
				target = callbackTarget(t, c, conn, "template_card_event")
				reply = &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "template_card", TemplateCard: aibot.RawTemplateCard(`{"card_type":"text_notice"}`)}}
				wantID, wantType, wantOp = "group", "group", CmdUpdate
			default:
				target = callbackTarget(t, c, conn, "")
			}
			if target != nil {
				target.Callback = &aibot.Callback{}
				target.Frame = Frame{}
			}
			count := 1
			if mode == "stream" {
				count = 2
			}
			for i := 0; i < count; i++ {
				done := make(chan error, 1)
				go func() {
					var err error
					if mode == "send" {
						_, err = c.Send(context.Background(), "active", aibot.OutgoingMessage{MsgType: "markdown", ChatType: 2, Markdown: &aibot.TextContent{Content: "hi"}})
					} else {
						if mode == "stream" {
							reply = streamReply("flow", false)
						}
						_, err = c.Reply(context.Background(), target, reply)
					}
					done <- err
				}()
				frame := conn.next(t)
				conn.ack(frame.Headers.ReqID, 0, nil)
				if err := <-done; err != nil {
					t.Fatal(err)
				}
			}
			requests := limiter.all()
			if len(requests) != count {
				t.Fatalf("calls %d", len(requests))
			}
			for i, r := range requests {
				if r.Scope != ScopeConversationSend || r.BotID != "bot" || r.ConversationID != wantID || r.ConversationType != wantType || r.Operation != wantOp || r.RequestID == "" {
					t.Fatalf("metadata %+v", r)
				}
				if mode == "stream" && (r.StreamID != "flow" || r.StreamStart != (i == 0)) {
					t.Fatalf("stream metadata %+v", r)
				}
			}
		})
	}
}

func TestLimiterCountsUploadOnlyOnce(t *testing.T) {
	limiter := new(recordingLimiter)
	c, conn := newTestClient(t, withTestLimiter(limiter))
	done := make(chan error, 1)
	go func() {
		_, err := c.UploadMedia(context.Background(), "file", "test.txt", strings.NewReader("12345"))
		done <- err
	}()
	init := conn.next(t)
	conn.ack(init.Headers.ReqID, 0, map[string]string{"upload_id": "upload"})
	chunk := conn.next(t)
	conn.ack(chunk.Headers.ReqID, 0, nil)
	finish := conn.next(t)
	conn.ack(finish.Headers.ReqID, 0, map[string]string{"type": "file", "media_id": "media", "created_at": "123"})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	requests := limiter.all()
	if len(requests) != 1 || requests[0].Scope != ScopeBotUpload || requests[0].Operation != CmdUploadInit || requests[0].ConversationID != "" {
		t.Fatalf("upload counted per chunk: %+v", requests)
	}
	if ConversationSendLimits().PerMinute != 30 || ConversationSendLimits().PerHour != 1000 || BotUploadLimits() != ConversationSendLimits() {
		t.Fatal("wrong advertised limits")
	}
}

func TestLimiterRejectionDoesNotStartStreamOrCloseSession(t *testing.T) {
	rejected := errors.New("limiter rejected")
	limiter := &recordingLimiter{err: rejected}
	c, conn := newTestClient(t, withTestLimiter(limiter))
	target := callbackTarget(t, c, conn, "")
	if _, err := c.Reply(context.Background(), target, streamReply("flow", false)); !errors.Is(err, rejected) {
		t.Fatal(err)
	}
	target.state.mu.Lock()
	n := len(target.state.streams)
	target.state.mu.Unlock()
	if n != 0 || c.State() != StateReady {
		t.Fatal("rejection changed lifecycle")
	}
	select {
	case <-conn.writes:
		t.Fatal("rejection wrote frame")
	default:
	}
	limiter.mu.Lock()
	limiter.err = nil
	limiter.mu.Unlock()
	done := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, streamReply("flow", false)); done <- err }()
	frame := conn.next(t)
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !limiter.all()[1].StreamStart {
		t.Fatal("rejected stream appeared started")
	}
}

type limiterContextKey struct{}

func TestLimiterWaitingHonorsReplyDeadlineAndValues(t *testing.T) {
	entered := make(chan struct{})
	limiter := LimiterFunc(func(ctx context.Context, r LimitRequest) error {
		if ctx.Value(limiterContextKey{}) != "value" {
			t.Error("context value lost")
		}
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	})
	c, conn := newTestClient(t, withTestLimiter(limiter))
	target := callbackTarget(t, c, conn, "")
	shortReplyDeadline(target, time.Now().Add(40*time.Millisecond))
	_, err := c.Reply(context.WithValue(context.Background(), limiterContextKey{}, "value"), target, streamReply("flow", false))
	<-entered
	if !errors.Is(err, aibot.ErrReplyExpired) || c.State() != StateReady {
		t.Fatalf("limiter expiry %v %s", err, c.State())
	}
	select {
	case <-conn.writes:
		t.Fatal("expired limiter waiter sent")
	default:
	}
}
func TestLimiterDoesNotBlockHeartbeatOrACKReader(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	limiter := LimiterFunc(func(ctx context.Context, r LimitRequest) error {
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	c, conn := newTestClient(t, withTestLimiter(limiter), withHeartbeat(15*time.Millisecond, time.Second))
	done := make(chan error, 1)
	go func() {
		_, err := c.Send(context.Background(), "chat", aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "hi"}})
		done <- err
	}()
	<-entered
	ping := conn.next(t)
	if ping.Command != CmdPing {
		t.Fatal(ping)
	}
	conn.ack(ping.Headers.ReqID, 0, nil)
	close(release)
	for {
		frame := conn.next(t)
		conn.ack(frame.Headers.ReqID, 0, nil)
		if frame.Command == CmdSend {
			break
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestLimiterCancellationOnCloseAndMissingConversation(t *testing.T) {
	entered := make(chan struct{})
	limiter := LimiterFunc(func(ctx context.Context, r LimitRequest) error { close(entered); <-ctx.Done(); return ctx.Err() })
	c, _ := newTestClient(t, withTestLimiter(limiter))
	done := make(chan error, 1)
	go func() {
		_, err := c.Send(context.Background(), "chat", aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "hi"}})
		done <- err
	}()
	<-entered
	_ = c.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closed waiter succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("limiter not canceled on close")
	}
	c, conn := newTestClient(t, withTestLimiter(new(recordingLimiter)))
	conn.push(`{"cmd":"aibot_msg_callback","headers":{"req_id":"missing"},"body":{"msgtype":"text"}}`)
	var target *ReceivedCallback
	for n := range c.Notifications() {
		if n.Callback != nil {
			target = n.Callback
			break
		}
	}
	if _, err := c.Reply(context.Background(), target, markdownReply()); !errors.Is(err, ErrInvalidArgument) {
		t.Fatal("empty conversation counted")
	}
}

func TestConcurrentStreamCallsAdmittedInOrder(t *testing.T) {
	limiter := new(recordingLimiter)
	c, conn := newTestClient(t, withTestLimiter(limiter))
	target := callbackTarget(t, c, conn, "")
	first := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, streamReply("flow", false)); first <- err }()
	frame := conn.next(t)
	second := make(chan error, 1)
	go func() { _, err := c.Reply(context.Background(), target, streamReply("flow", true)); second <- err }()
	time.Sleep(20 * time.Millisecond)
	if len(limiter.all()) != 1 {
		t.Fatal("limiter ran ahead of ACK serialization")
	}
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	frame = conn.next(t)
	conn.ack(frame.Headers.ReqID, 0, nil)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	requests := limiter.all()
	if len(requests) != 2 || !requests[0].StreamStart || requests[1].StreamStart {
		t.Fatalf("wrong concurrent admission %+v", requests)
	}
	if _, err := c.Reply(context.Background(), target, streamReply("flow", false)); !errors.Is(err, aibot.ErrStreamFinished) {
		t.Fatal(err)
	}
	if len(limiter.all()) != 2 {
		t.Fatal("finished stream consumed quota")
	}
}
