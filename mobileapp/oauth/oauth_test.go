package oauth

import (
	"context"
	"encoding/json"
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
	tr := transport.New(server.Client(), server.URL, transport.RetryPolicy{})
	o := New(tr, Config{AppID: "id", AppSecret: "secret"}, c)
	got, err := o.TokenFromCode(context.Background(), "c")
	if err != nil || got.OpenID != "o" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if v, ok, _ := c.Get(context.Background(), o.cacheKey("access", "o")); !ok || v != "a" {
		t.Fatalf("access cache=%q %v", v, ok)
	}
	if v, ok, _ := c.Get(context.Background(), o.cacheKey("refresh", "o")); !ok || v != "r" {
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
	tr := transport.New(server.Client(), server.URL, transport.RetryPolicy{})
	client := New(tr, Config{AppID: "id", AppSecret: "secret"}, cache)
	if err := cache.Put(context.Background(), client.cacheKey("access", "openid"), "access", time.Hour); err != nil {
		t.Fatal(err)
	}
	_, err := client.UserInfo(context.Background(), "openid")
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40003" || apiErr.RequestID != "rid" || apiErr.HTTPStatus != 200 {
		t.Fatalf("err=%#v", err)
	}
}

func TestSharedCacheIsolatedByAppID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sns/oauth2/access_token":
			appID := r.URL.Query().Get("appid")
			response := map[string]interface{}{
				"access_token":  appID + "-access",
				"refresh_token": appID + "-refresh",
				"expires_in":    3600,
				"openid":        "shared-openid",
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("encode token response: %v", err)
			}
		case "/sns/userinfo":
			_, _ = w.Write([]byte(`{"openid":"shared-openid","nickname":"` + r.URL.Query().Get("access_token") + `"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	sharedCache := corecache.NewMemory()
	tr := transport.New(server.Client(), server.URL, transport.RetryPolicy{})
	first := New(tr, Config{AppID: "app-a", AppSecret: "secret"}, sharedCache)
	second := New(tr, Config{AppID: "app-b", AppSecret: "secret"}, sharedCache)
	for _, client := range []*Client{first, second} {
		if _, err := client.TokenFromCode(context.Background(), "code"); err != nil {
			t.Fatal(err)
		}
	}
	if first.cacheKey("access", "shared-openid") == second.cacheKey("access", "shared-openid") ||
		first.cacheKey("refresh", "shared-openid") == second.cacheKey("refresh", "shared-openid") {
		t.Fatal("different AppIDs share OAuth cache keys")
	}
	for _, test := range []struct {
		client   *Client
		expected string
	}{{first, "app-a-access"}, {second, "app-b-access"}} {
		user, err := test.client.UserInfo(context.Background(), "shared-openid")
		if err != nil || user.Nickname != test.expected {
			t.Fatalf("user=%+v err=%v, want nickname %q", user, err, test.expected)
		}
	}
}
