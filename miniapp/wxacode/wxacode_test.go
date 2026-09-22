package wxacode

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

func TestGetUnlimitedImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/wxa/getwxacodeunlimit" ||
			r.URL.Query().Get("access_token") != "token" {
			t.Errorf("request=%s %s %v", r.Method, r.URL.Path, r.URL.Query())
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["scene"] != "abc" {
			t.Errorf("body=%v err=%v", body, err)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{1, 2, 3})
	}))
	defer server.Close()
	manager := auth.NewManager(
		"miniapp", "wxa", cache.NewMemory(),
		auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
			return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
		}))
	client := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
	data, ct, err := client.GetUnlimited(context.Background(), "abc", nil)
	if err != nil || ct != "image/png" || len(data) != 3 {
		t.Fatalf("data=%v ct=%q err=%v", data, ct, err)
	}
}

func TestGetUnlimitedAcceptsOnlyNonEmptyImages(t *testing.T) {
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
			manager := auth.NewManager("miniapp", "wxa-image", cache.NewMemory(), auth.ProviderFunc(
				func(context.Context) (auth.Credential, error) {
					return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
				},
			))
			client := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
			image, _, err := client.GetUnlimited(context.Background(), "scene", nil)
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
