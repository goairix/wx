package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	corehook "github.com/goairix/wx/v2/core/webhook"
	"github.com/goairix/wx/v2/work/aibot"
)

const testKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

func TestHandlerRejectsTypedNil(t *testing.T) {
	c, err := New(Config{Token: "token", EncodingAESKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := corehook.NewMessageCipher(corehook.CipherConfig{EncodingAESKey: testKey, AllowEmptyReceiver: true})
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt([]byte(`{"msgtype":"text","text":{"content":"hello"}}`))
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{"timestamp": {"123"}, "nonce": {"nonce"}, "msg_signature": {corehook.MessageSignature("token", "123", "nonce", encrypted)}}
	body, err := json.Marshal(map[string]string{"encrypt": encrypted})
	if err != nil {
		t.Fatal(err)
	}
	var next HandlerFunc
	r := httptest.NewRequest(http.MethodPost, "/?"+query.Encode(), strings.NewReader(string(body)))
	w := httptest.NewRecorder()
	c.Handler(next).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWebhookVerificationAndIncomingCallbacks(t *testing.T) {
	c, err := New(Config{Token: "token", EncodingAESKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := corehook.NewMessageCipher(corehook.CipherConfig{EncodingAESKey: testKey, AllowEmptyReceiver: true})
	if err != nil {
		t.Fatal(err)
	}
	type contextKey struct{}
	calls := 0
	handler := c.Handler(HandlerFunc(func(ctx context.Context, cb *aibot.Callback) (*aibot.Reply, error) {
		calls++
		if ctx.Value(contextKey{}) != "original" || cb.Message.Text.Content != "hello" {
			t.Fatal("context or body lost")
		}
		return nil, nil
	}))
	challenge, _ := cipher.Encrypt([]byte("challenge"))
	query := url.Values{"timestamp": {"123"}, "nonce": {"request-nonce"}, "echostr": {challenge}, "msg_signature": {corehook.MessageSignature("token", "123", "request-nonce", challenge)}}
	r := httptest.NewRequest(http.MethodGet, "/?"+query.Encode(), nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.String() != "challenge" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	encrypted, _ := cipher.Encrypt([]byte(`{"msgtype":"text","text":{"content":"hello"},"future":1}`))
	query.Del("echostr")
	query.Set("msg_signature", corehook.MessageSignature("token", "123", "request-nonce", encrypted))
	body, _ := json.Marshal(map[string]string{"encrypt": encrypted})
	r = httptest.NewRequest(http.MethodPost, "/?"+query.Encode(), strings.NewReader(string(body))).WithContext(context.WithValue(context.Background(), contextKey{}, "original"))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 200 || w.Body.Len() != 0 || calls != 1 {
		t.Fatalf("%d %s %d", w.Code, w.Body.String(), calls)
	}
	query.Set("msg_signature", "bad")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/?"+query.Encode(), strings.NewReader(string(body))))
	if w.Code == 200 || calls != 1 {
		t.Fatal("bad signature dispatched")
	}
}
func TestWebhookReplyMatrixAndEnvelope(t *testing.T) {
	c, err := New(Config{Token: "token", EncodingAESKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cb, msg string
		want    bool
	}{
		{`{"msgtype":"event","event":{"eventtype":"enter_chat"}}`, "text", true},
		{`{"msgtype":"text"}`, "stream", true},
		{`{"msgtype":"text"}`, "stream_with_template_card", true},
		{`{"msgtype":"stream","stream":{"id":"s"}}`, "stream", true},
		{`{"msgtype":"text"}`, "markdown", false},
		{`{"msgtype":"event","event":{"eventtype":"feedback_event"}}`, "stream", false},
	} {
		t.Run(tc.msg+tc.cb, func(t *testing.T) {
			cb, _ := aibot.ParseCallback([]byte(tc.cb))
			reply := &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: tc.msg}}
			switch tc.msg {
			case "text":
				reply.Text = &aibot.TextContent{Content: "welcome"}
			case "markdown":
				reply.Markdown = &aibot.TextContent{Content: "hi"}
			default:
				reply.Stream = &aibot.Stream{ID: "s", Content: "hi"}
				if tc.msg == "stream_with_template_card" {
					reply.TemplateCard = aibot.RawTemplateCard(`{"card_type":"text_notice","future":true}`)
				}
			}
			wire, err := c.encodeReply(cb, reply, "original-nonce")
			if !tc.want {
				if err == nil {
					t.Fatal("accepted unsupported")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var out struct {
				Encrypt, MsgSignature, Nonce string
				Timestamp                    int64
			}
			if err = json.Unmarshal(wire, &out); err != nil {
				t.Fatal(err)
			}
			if out.Nonce != "original-nonce" || out.Timestamp <= 0 || out.MsgSignature != corehook.MessageSignature("token", fmt.Sprint(out.Timestamp), out.Nonce, out.Encrypt) {
				t.Fatalf("%s", wire)
			}
			cipher, _ := corehook.NewMessageCipher(corehook.CipherConfig{EncodingAESKey: testKey, AllowEmptyReceiver: true})
			plain, err := cipher.Decrypt(out.Encrypt)
			if err != nil || !strings.Contains(string(plain), tc.msg) {
				t.Fatalf("%s %v", plain, err)
			}
		})
	}
}
func TestReplyBudgetsBeforeEncoding(t *testing.T) {
	if _, err := New(Config{Token: "t", EncodingAESKey: testKey, MaxPlainReplyBytes: 20 << 20, MaxWireReplyBytes: 20 << 20}); err == nil {
		t.Fatal("accepted insufficient wire budget")
	}
	c, err := New(Config{Token: "t", EncodingAESKey: testKey, MaxPlainReplyBytes: 100, MaxWireReplyBytes: 8192})
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := aibot.ParseCallback([]byte(`{"msgtype":"text"}`))
	_, err = c.encodeReply(cb, &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "stream", Stream: &aibot.Stream{ID: "s", Content: strings.Repeat("a", 100)}}}, "n")
	if err == nil {
		t.Fatal("plaintext budget not enforced")
	}
}

func TestHandlerCannotChangeOriginalStreamTarget(t *testing.T) {
	c, _ := New(Config{Token: "t", EncodingAESKey: testKey})
	cipher, _ := corehook.NewMessageCipher(corehook.CipherConfig{EncodingAESKey: testKey, AllowEmptyReceiver: true})
	encrypted, _ := cipher.Encrypt([]byte(`{"msgtype":"stream","stream":{"id":"original"}}`))
	q := url.Values{"timestamp": {"1"}, "nonce": {"n"}, "msg_signature": {corehook.MessageSignature("t", "1", "n", encrypted)}}
	body, _ := json.Marshal(map[string]string{"encrypt": encrypted})
	handler := c.Handler(HandlerFunc(func(_ context.Context, cb *aibot.Callback) (*aibot.Reply, error) {
		cb.Message.MsgType = "text"
		return &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "stream", Stream: &aibot.Stream{ID: "different", Content: "reply"}}}, nil
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/?"+q.Encode(), strings.NewReader(string(body))))
	if w.Code == 200 {
		t.Fatal("handler mutation bypassed stream target")
	}
}
