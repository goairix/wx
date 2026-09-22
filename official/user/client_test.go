package user

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestClientInfoUsesAccessTokenAndAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("access_token"); got != "app-token" {
			t.Errorf("access_token=%q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer app-token" {
			t.Errorf("authorization=%q", got)
		}
		_, _ = w.Write([]byte(`{"openid":"user","nickname":"Alice"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "app-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
	info, err := client.Info(context.Background(), "user")
	if err != nil || info.Nickname != "Alice" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestClientInfoReturnsStructuredPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40003,"errmsg":"invalid openid"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "app-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
	_, err := client.Info(context.Background(), "bad")
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error %T does not satisfy core/errors.Error", err)
	}
	if platformErr.Code != "40003" {
		t.Fatalf("code=%q", platformErr.Code)
	}
}
