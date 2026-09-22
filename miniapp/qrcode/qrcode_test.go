package qrcode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestCreateRequestAndError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/cgi-bin/wxaapp/createwxaqrcode" ||
			r.URL.Query().Get("access_token") != "token" {
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
	manager := auth.NewManager(
		"miniapp", "qrtest", cache.NewMemory(),
		auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
			return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
		}))
	client := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
	_, _, err := client.Create(context.Background(), "pages/home")
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40014" || apiErr.RequestID != "rid" {
		t.Fatalf("err=%#v", err)
	}
}

func TestCreateAcceptsOnlyNonEmptyImages(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantImage   bool
		wantCode    string
	}{
		{"image", "image/png", "PNG", true, ""},
		{"platform error", "application/json; charset=utf-8", `{"errcode":40014,"errmsg":"expired"}`, false, "40014"},
		{"text", "text/plain", "unexpected", false, ""},
		{"malformed JSON", "application/json", `{`, false, ""},
		{"empty image", "image/png", "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			manager := auth.NewManager("miniapp", "qr-image", cache.NewMemory(), auth.ProviderFunc(
				func(context.Context) (auth.Credential, error) {
					return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
				},
			))
			client := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
			image, _, err := client.Create(context.Background(), "pages/home")
			if tt.wantImage {
				if err != nil || string(image) != tt.body {
					t.Fatalf("image=%q err=%v", image, err)
				}
				return
			}
			if err == nil || len(image) != 0 {
				t.Fatalf("image=%q err=%v", image, err)
			}
			if tt.wantCode != "" {
				var apiErr *wxerrors.Error
				if !errors.As(err, &apiErr) || apiErr.Code != tt.wantCode {
					t.Fatalf("err=%#v", err)
				}
			}
		})
	}
}
