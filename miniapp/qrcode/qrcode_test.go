package qrcode

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateRequestAndError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cgi-bin/wxaapp/createwxaqrcode" || r.URL.Query().Get("access_token") != "token" {
			t.Errorf("request=%s %s %v", r.Method, r.URL.Path, r.URL.Query())
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["path"] != "pages/home" {
			t.Errorf("body=%v err=%v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "rid")
		_, _ = w.Write([]byte(`{"errcode":40014,"errmsg":"bad token"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("miniapp", "qrtest", cache.NewMemory(), auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	_, _, err := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager).Create(context.Background(), "pages/home")
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40014" || apiErr.RequestID != "rid" {
		t.Fatalf("err=%#v", err)
	}
}
