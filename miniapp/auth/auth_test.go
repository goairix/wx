package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestCode2SessionRequestAndError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sns/jscode2session" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("appid") != "app" || r.URL.Query().Get("secret") != "secret" || r.URL.Query().Get("js_code") != "code" || r.URL.Query().Get("grant_type") != "authorization_code" {
			t.Fatalf("query=%v", r.URL.Query())
		}
		w.Header().Set("X-Request-Id", "rid")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"bad appid"}`))
	}))
	defer server.Close()
	a := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), Config{AppID: "app", AppSecret: "secret"})
	_, err := a.Code2Session(context.Background(), "code")
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || platformErr.Code != "40013" || platformErr.HTTPStatus != http.StatusOK || platformErr.RequestID != "rid" {
		t.Fatalf("err=%#v", err)
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
