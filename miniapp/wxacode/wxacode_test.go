package wxacode

import (
	"context"
	"encoding/json"
	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetUnlimitedImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/wxa/getwxacodeunlimit" || r.URL.Query().Get("access_token") != "token" {
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
	manager := auth.NewManager("miniapp", "wxa", cache.NewMemory(), auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	data, ct, err := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager).GetUnlimited(context.Background(), "abc", nil)
	if err != nil || ct != "image/png" || len(data) != 3 {
		t.Fatalf("data=%v ct=%q err=%v", data, ct, err)
	}
}
