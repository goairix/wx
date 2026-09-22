package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	corecache "github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
)

func TestTokenFromCodeRequestAndSeparateKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sns/oauth2/access_token" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("code") != "c" {
			t.Fatalf("query=%v", r.URL.Query())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a","refresh_token":"r","expires_in":3600,"openid":"o"}`))
	}))
	defer server.Close()
	c := corecache.NewMemory()
	o := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), Config{AppID: "id", AppSecret: "secret"}, c)
	got, err := o.TokenFromCode(context.Background(), "c")
	if err != nil || got.OpenID != "o" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if v, ok, _ := c.Get(context.Background(), "mobileapp:user:access:o"); !ok || v != "a" {
		t.Fatalf("access cache=%q %v", v, ok)
	}
	if v, ok, _ := c.Get(context.Background(), "mobileapp:user:refresh:o"); !ok || v != "r" {
		t.Fatalf("refresh cache=%q %v", v, ok)
	}
}
