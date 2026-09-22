package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestTokenFromCodeRequestAndSeparateKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/sns/oauth2/access_token" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("appid") != "id" || query.Get("secret") != "secret" ||
			query.Get("code") != "c" || query.Get("grant_type") != "authorization_code" {
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

func TestUserInfoPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.Method != http.MethodGet || r.URL.Path != "/sns/userinfo" ||
			query.Get("access_token") != "access" || query.Get("openid") != "openid" {
			t.Errorf("request=%s %s %v", r.Method, r.URL.Path, r.URL.Query())
		}
		w.Header().Set("X-Request-Id", "rid")
		_, _ = w.Write([]byte(`{"errcode":40003,"errmsg":"bad openid"}`))
	}))
	defer server.Close()
	cache := corecache.NewMemory()
	if err := cache.Put(context.Background(), "mobileapp:user:access:openid", "access", time.Hour); err != nil {
		t.Fatal(err)
	}
	client := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), Config{AppID: "id", AppSecret: "secret"}, cache)
	_, err := client.UserInfo(context.Background(), "openid")
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40003" || apiErr.RequestID != "rid" || apiErr.HTTPStatus != 200 {
		t.Fatalf("err=%#v", err)
	}
}
