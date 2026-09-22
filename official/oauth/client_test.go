package oauth

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestClientTokenAndUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sns/oauth2/access_token" {
			if got := r.URL.Query().Get("appid"); got != "app" {
				t.Errorf("appid=%q", got)
			}
			if got := r.URL.Query().Get("secret"); got != "secret" {
				t.Errorf("secret=%q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"oauth-token","expires_in":7200,"openid":"user"}`))
			return
		}
		if r.URL.Path == "/sns/userinfo" {
			if got := r.URL.Query().Get("access_token"); got != "oauth-token" {
				t.Errorf("access_token=%q", got)
			}
			_, _ = w.Write([]byte(`{"openid":"user","nickname":"Alice"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), "app", "secret")
	token, err := client.TokenFromCode(context.Background(), "code")
	if err != nil || token.AccessToken != "oauth-token" || token.Openid != "user" {
		t.Fatalf("token=%+v err=%v", token, err)
	}
	profile, err := client.UserInfo(context.Background(), token.AccessToken, token.Openid)
	if err != nil || profile.Nickname != "Alice" {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
}

func TestClientReturnsStructuredPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer server.Close()
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), "app", "secret")
	_, err := client.TokenFromCode(context.Background(), "code")
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error %T does not satisfy core/errors.Error", err)
	}
	if platformErr.Code != "40013" {
		t.Fatalf("code=%q", platformErr.Code)
	}
}

func TestClientEscapesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"access_token":"t","openid":"o"}`))
	}))
	defer server.Close()
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), "app", "secret")
	_, _ = client.TokenFromCode(context.Background(), "a+b&c")
}
