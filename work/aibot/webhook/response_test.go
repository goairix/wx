package webhook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

func TestReplyByURLPartialResultAndSingleAttempt(t *testing.T) {
	for _, tc := range []struct {
		body      string
		status    int
		wantCode  int
		wantError bool
	}{
		{`{"errcode":0,"errmsg":"ok"}`, 200, 0, false},
		{`{"errcode":40001,"errmsg":"invalid"}`, 200, 40001, true},
		{`{"errcode":500,"errmsg":"failed"}`, 502, 500, true},
		{`bad`, 200, 0, true},
	} {
		t.Run(tc.body, func(t *testing.T) {
			var count atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count.Add(1)
				if r.Method != "POST" {
					t.Error(r.Method)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			c, err := New(Config{Token: "t", EncodingAESKey: testKey})
			if err != nil {
				t.Fatal(err)
			}
			result, err := c.ReplyByURL(context.Background(), server.URL+"?secret=private", aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "hi", Feedback: &aibot.Feedback{ID: "feedback"}}})
			if (err != nil) != tc.wantError || result.HTTPStatus != tc.status || result.ErrCode != tc.wantCode || count.Load() != 1 {
				t.Fatalf("%+v %v %d", result, err, count.Load())
			}
			if err != nil {
				var e *coreerrors.Error
				if !errors.As(err, &e) || e.HTTPStatus != tc.status {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestReplyByURLDoesNotFollowRedirect(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Location", "/again")
			w.WriteHeader(307)
			return
		}
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()
	c, _ := New(Config{Token: "t", EncodingAESKey: testKey})
	result, err := c.ReplyByURL(context.Background(), server.URL, aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "hi"}})
	if err == nil || result.HTTPStatus != 307 || requests.Load() != 1 {
		t.Fatalf("redirect replay: %+v %v %d", result, err, requests.Load())
	}
}
