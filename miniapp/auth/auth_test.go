package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goairix/wx/v2/core/transport"
)

func TestCode2SessionRequestAndError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sns/jscode2session" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("appid") != "app" || r.URL.Query().Get("js_code") != "code" {
			t.Fatalf("query=%v", r.URL.Query())
		}
		w.Header().Set("X-Request-Id", "rid")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"bad appid"}`))
	}))
	defer server.Close()
	a := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), Config{AppID: "app", AppSecret: "secret"})
	_, err := a.Code2Session(context.Background(), "code")
	if err == nil || !errors.Is(err, context.Canceled) && err.Error() == "" {
		t.Fatalf("err=%v", err)
	}
}
func TestCode2SessionCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	a := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), Config{AppID: "app", AppSecret: "secret"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := a.Code2Session(ctx, "code")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
