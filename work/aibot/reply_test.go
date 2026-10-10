package aibot

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
)

// Wire fixtures catch wrong command selection, omitted nested fields and accidental
// normalization of template card extensions.
func TestReplyWirePayloads(t *testing.T) {
	card := TemplateCard{"card_type": "button_interaction", "button_selection": map[string]any{"disable": false}, "extension": map[string]any{"custom": "retained"}}
	tests := []struct {
		name, command, body string
		active              bool
		call                func(*Client) (Frame, error)
	}{
		{"markdown", "aibot_respond_msg", `{"msgtype":"markdown","markdown":{"content":"**hello**"}}`, false, func(c *Client) (Frame, error) { return c.ReplyMarkdown(context.Background(), "request", "**hello**") }},
		{"generic", "aibot_respond_msg", `{"msgtype":"markdown","markdown":{"content":"generic"}}`, false, func(c *Client) (Frame, error) {
			return c.Reply(context.Background(), "request", MessageBody{MsgType: "markdown", Markdown: &TextContent{Content: "generic"}})
		}},
		{"empty stream content", "aibot_respond_msg", `{"msgtype":"stream","stream":{"id":"stream","finish":false,"content":""}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyStream(context.Background(), "request", Stream{ID: "stream"})
		}},
		{"card", "aibot_respond_msg", `{"msgtype":"template_card","template_card":{"card_type":"button_interaction","button_selection":{"disable":false},"extension":{"custom":"retained"}}}`, false, func(c *Client) (Frame, error) { return c.ReplyTemplateCard(context.Background(), "request", card) }},
		{"video", "aibot_respond_msg", `{"msgtype":"video","video":{"media_id":"media","title":"title","description":"description"}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyMedia(context.Background(), "request", "video", MediaContent{MediaID: "media", Title: "title", Description: "description"})
		}},
		{"file", "aibot_respond_msg", `{"msgtype":"file","file":{"media_id":"media"}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyMedia(context.Background(), "request", "file", MediaContent{MediaID: "media"})
		}},
		{"image", "aibot_respond_msg", `{"msgtype":"image","image":{"media_id":"media"}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyMedia(context.Background(), "request", "image", MediaContent{MediaID: "media"})
		}},
		{"voice", "aibot_respond_msg", `{"msgtype":"voice","voice":{"media_id":"media"}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyMedia(context.Background(), "request", "voice", MediaContent{MediaID: "media"})
		}},
		{"stream", "aibot_respond_msg", `{"msgtype":"stream","stream":{"id":"stream","finish":true,"content":"full content","msg_item":[{"msgtype":"image","image":{"base64":"eA==","md5":"9dd4e461268c8034f5c8564e155c67a6"}}],"feedback":{"id":"feedback"}}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyStream(context.Background(), "request", Stream{ID: "stream", Finish: true, Content: "full content", MsgItem: []StreamItem{{MsgType: "image", Image: StreamImage{Base64: "eA==", MD5: "9dd4e461268c8034f5c8564e155c67a6"}}}, Feedback: &Feedback{ID: "feedback"}})
		}},
		{"stream card", "aibot_respond_msg", `{"msgtype":"stream_with_template_card","stream":{"id":"stream","finish":false,"content":"partial"},"template_card":{"card_type":"button_interaction","button_selection":{"disable":false},"extension":{"custom":"retained"}}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyStreamWithTemplateCard(context.Background(), "request", Stream{ID: "stream", Content: "partial"}, card)
		}},
		{"update", "aibot_respond_update_msg", `{"response_type":"update_template_card","userids":["u1","u2"],"template_card":{"card_type":"button_interaction","button_selection":{"disable":false},"extension":{"custom":"retained"},"task_id":"original-task"}}`, false, func(c *Client) (Frame, error) {
			update := make(TemplateCard, len(card)+1)
			for key, value := range card {
				update[key] = value
			}
			update["task_id"] = "original-task"
			return c.UpdateTemplateCard(context.Background(), "request", update, []string{"u1", "u2"})
		}},
		{"welcome", "aibot_respond_welcome_msg", `{"msgtype":"text","text":{"content":"welcome"}}`, false, func(c *Client) (Frame, error) {
			return c.ReplyWelcome(context.Background(), "request", MessageBody{MsgType: "text", Text: &TextContent{Content: "welcome"}})
		}},
		{"send", "aibot_send_msg", `{"chatid":"chat","msgtype":"markdown","markdown":{"content":"active"}}`, true, func(c *Client) (Frame, error) {
			return c.Send(context.Background(), "chat", MessageBody{MsgType: "markdown", Markdown: &TextContent{Content: "active"}})
		}},
		{"send single with feedback", "aibot_send_msg", `{"chatid":"user","chat_type":1,"msgtype":"markdown","markdown":{"content":"active","feedback":{"id":"feedback"}}}`, true, func(c *Client) (Frame, error) {
			return c.Send(context.Background(), "user", MessageBody{ChatType: 1, MsgType: "markdown", Markdown: &TextContent{Content: "active", Feedback: &Feedback{ID: "feedback"}}})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, s := newTestClient(t)
			result := make(chan error, 1)
			go func() { _, err := tt.call(c); result <- err }()
			f := s.next(t)
			if f.Command != tt.command {
				t.Fatalf("command = %q", f.Command)
			}
			if f.Headers.ReqID == "" || (!tt.active && f.Headers.ReqID != "request") {
				t.Fatalf("req_id = %q", f.Headers.ReqID)
			}
			var got, want any
			if err := json.Unmarshal(f.Body, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tt.body), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("body = %s; want %s", f.Body, tt.body)
			}
			select {
			case err := <-result:
				t.Fatalf("completed before ACK: %v", err)
			default:
			}
			s.ack(f.Headers.ReqID, 0, nil)
			if err := <-result; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStreamRejectsInvalidPayloadBeforeWrite(t *testing.T) {
	items := make([]StreamItem, 11)
	for i := range items {
		items[i] = StreamItem{MsgType: "image", Image: StreamImage{Base64: "eA==", MD5: "9dd4e461268c8034f5c8564e155c67a6"}}
	}
	tests := []Stream{
		{Content: "missing ID"},
		{ID: "s", Content: strings.Repeat("中", 6827)},
		{ID: "s", Content: string([]byte{0xff})},
		{ID: "s", MsgItem: items[:1]},
		{ID: "s", Finish: true, MsgItem: items},
		{ID: "s", Feedback: &Feedback{ID: strings.Repeat("x", 257)}},
	}
	for i, stream := range tests {
		c, s := newTestClient(t)
		if _, err := c.ReplyStream(context.Background(), "request", stream); err == nil {
			t.Fatalf("case %d accepted", i)
		}
		select {
		case b := <-s.writes:
			t.Fatalf("invalid stream written: %s", b)
		default:
		}
	}
}

func TestReplyRejectsInvalidTargetAndMessage(t *testing.T) {
	c, s := newTestClient(t, WithRequestTimeout(20*time.Millisecond))
	tests := []func() (Frame, error){
		func() (Frame, error) { return c.ReplyMarkdown(context.Background(), "", "hello") },
		func() (Frame, error) { return c.Reply(context.Background(), "r", MessageBody{MsgType: "unknown"}) },
		func() (Frame, error) { return c.Reply(context.Background(), "r", MessageBody{MsgType: "markdown"}) },
		func() (Frame, error) {
			return c.ReplyMedia(context.Background(), "r", "audio", MediaContent{MediaID: "m"})
		},
		func() (Frame, error) { return c.ReplyMedia(context.Background(), "r", "file", MediaContent{}) },
		func() (Frame, error) {
			return c.ReplyWelcome(context.Background(), "r", MessageBody{MsgType: "markdown", Markdown: &TextContent{Content: "x"}})
		},
		func() (Frame, error) {
			return c.Send(context.Background(), "", MessageBody{MsgType: "markdown", Markdown: &TextContent{Content: "x"}})
		},
		func() (Frame, error) {
			return c.Send(context.Background(), "chat", MessageBody{MsgType: "stream", Stream: &Stream{ID: "s"}})
		},
		func() (Frame, error) {
			return c.UpdateTemplateCard(context.Background(), "r", TemplateCard{"card_type": "button_interaction"}, nil)
		},
	}
	for i, call := range tests {
		if _, err := call(); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("case %d validation error = %v", i, err)
		}
	}
	select {
	case b := <-s.writes:
		t.Fatalf("invalid reply written: %s", b)
	default:
	}
}

func TestStreamQueueUsesRequestTimeoutAndPreservesOperation(t *testing.T) {
	c, s := newTestClient(t, WithRequestTimeout(20*time.Millisecond))
	locked := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	go func() {
		_, _ = c.withKey(context.Background(), "stream:shared", func() (Frame, error) { close(locked); <-release; return Frame{}, nil })
	}()
	<-locked
	result := make(chan error, 1)
	go func() { _, err := c.ReplyStream(context.Background(), "queued", Stream{ID: "shared"}); result <- err }()
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("queued stream error = %v", err)
		}
		var sdkErr *coreerrors.Error
		if !errors.As(err, &sdkErr) || sdkErr.RequestID != "queued" || sdkErr.Operation != "aibot.aibot_respond_msg" {
			t.Fatalf("queue lost operation metadata: %+v", sdkErr)
		}
	case <-time.After(time.Second):
		t.Fatal("stream queue ignored request timeout")
	}
	select {
	case b := <-s.writes:
		t.Fatalf("timed out stream written: %s", b)
	default:
	}
}

func TestStreamOrderingAcrossRequestIDsAndCancellation(t *testing.T) {
	c, s := newTestClient(t)
	first := make(chan error, 1)
	go func() {
		_, err := c.ReplyStream(context.Background(), "first", Stream{ID: "shared", Content: "first"})
		first <- err
	}()
	f := s.next(t)
	ctx, cancel := context.WithCancel(context.Background())
	second := make(chan error, 1)
	go func() { _, err := c.ReplyStream(ctx, "second", Stream{ID: "shared", Content: "second"}); second <- err }()
	select {
	case b := <-s.writes:
		t.Fatalf("second sent before first ACK: %s", b)
	case <-time.After(20 * time.Millisecond):
	}
	cancel()
	if err := <-second; !errors.Is(err, context.Canceled) {
		t.Fatalf("queue cancellation: %v", err)
	}
	s.ack(f.Headers.ReqID, 0, nil)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	third := make(chan error, 1)
	go func() {
		_, err := c.ReplyStream(context.Background(), "third", Stream{ID: "shared", Finish: true, Content: strings.Repeat("x", 20480)})
		third <- err
	}()
	f = s.next(t)
	if f.Headers.ReqID != "third" {
		t.Fatalf("stale queued reply sent: %s", f.Headers.ReqID)
	}
	s.ack(f.Headers.ReqID, 0, nil)
	if err := <-third; err != nil {
		t.Fatal(err)
	}
}

func TestIncomingMessageAndEventModels(t *testing.T) {
	var m Message
	err := json.Unmarshal([]byte(`{"msgid":"m","aibotid":"bot","chatid":"chat","chattype":"group","from":{"userid":"u"},"create_time":123,"response_url":"response","msgtype":"mixed","mixed":{"msg_item":[{"msgtype":"text","text":{"content":"hello"}},{"msgtype":"image","image":{"url":"image","aeskey":"key"}}]},"quote":{"msgtype":"file","file":{"url":"file","aeskey":"key2"}},"voice":{"content":"speech"},"video":{"url":"video","aeskey":"key3"}}`), &m)
	if err != nil {
		t.Fatal(err)
	}
	if m.MsgID != "m" || m.AIBotID != "bot" || m.ChatID != "chat" || m.ChatType != "group" || m.From.UserID != "u" || m.CreateTime != 123 || m.ResponseURL != "response" || m.MsgType != "mixed" || m.Mixed.MsgItem[0].Text.Content != "hello" || m.Mixed.MsgItem[1].Image.AESKey != "key" || m.Quote.File.URL != "file" || m.Voice.Content != "speech" || m.Video.URL != "video" {
		t.Fatalf("incorrect incoming message: %+v", m)
	}
	var event EventMessage
	if err := json.Unmarshal([]byte(`{"msgid":"e","aibotid":"bot","create_time":124,"from":{"userid":"u","corpid":"corp"},"msgtype":"event","event":{"eventtype":"template_card_event","event_key":"clicked","task_id":"task"}}`), &event); err != nil {
		t.Fatal(err)
	}
	if event.Event.EventType != "template_card_event" || event.Event.EventKey != "clicked" || event.Event.TaskID != "task" || event.From.CorpID != "corp" || event.CreateTime != 124 {
		t.Fatalf("incorrect event: %+v", event)
	}
}

func TestIncomingOfficialNestedEvents(t *testing.T) {
	var card EventMessage
	if err := json.Unmarshal([]byte(`{"response_url":"response","msgtype":"event","event":{"eventtype":"template_card_event","template_card_event":{"card_type":"vote_interaction","event_key":"submit","task_id":"task","selected_items":{"selected_item":[{"question_key":"q","option_ids":{"option_id":["one","two"]}}]}}}}`), &card); err != nil {
		t.Fatal(err)
	}
	data := card.Event.TemplateCardEvent
	if data == nil || card.ResponseURL != "response" || data.CardType != "vote_interaction" || data.EventKey != "submit" || data.TaskID != "task" || data.SelectedItems.SelectedItem[0].QuestionKey != "q" || !reflect.DeepEqual(data.SelectedItems.SelectedItem[0].OptionIDs.OptionID, []string{"one", "two"}) {
		t.Fatalf("card event lost fields: %+v", card)
	}
	var feedback EventMessage
	if err := json.Unmarshal([]byte(`{"msgtype":"event","event":{"eventtype":"feedback_event","feedback_event":{"id":"feedback","type":2,"content":"incomplete","inaccurate_reason_list":[2,4]}}}`), &feedback); err != nil {
		t.Fatal(err)
	}
	f := feedback.Event.FeedbackEvent
	if f == nil || f.ID != "feedback" || f.Type != 2 || f.Content != "incomplete" || !reflect.DeepEqual(f.InaccurateReasonList, []int{2, 4}) {
		t.Fatalf("feedback event lost fields: %+v", feedback)
	}
}
