package jssdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/official/internal/api"
)

// Accepting an empty ticket or nonpositive lifetime must never produce a signed
// configuration or poison the cache before a subsequent valid ticket response.
func TestInvalidTicketDoesNotSignOrCache(t *testing.T) {
	for _, bad := range []string{
		`{"errcode":0,"ticket":"","expires_in":7200}`,
		`{"errcode":0,"ticket":"invalid","expires_in":0}`,
		`{"errcode":0,"ticket":"invalid","expires_in":-1}`,
	} {
		t.Run(bad, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/cgi-bin/ticket/getticket" || r.URL.Query().Get("type") != "jsapi" || r.URL.Query().Get("access_token") != "token" {
					t.Errorf("request = %s", r.URL)
				}
				calls++
				if calls == 1 {
					_, _ = io.WriteString(w, bad)
					return
				}
				_, _ = fmt.Fprint(w, `{"errcode":0,"ticket":"valid-ticket","expires_in":7200}`)
			}))
			defer server.Close()
			manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
				return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
			}))
			client := NewClient(api.New(transport.New(nil, server.URL, transport.RetryPolicy{}), manager), "app", cache.NewMemory())
			client.now = func() time.Time { return time.Unix(1, 0) }
			client.nonce = func(int) (string, error) { return "nonce", nil }
			if out, err := client.BuildConfig(context.Background(), "https://example.test", []string{"scanQRCode"}, false, false); err == nil || out != nil {
				t.Fatalf("invalid ticket signed: config=%v, err=%v", out, err)
			}
			out, err := client.BuildConfig(context.Background(), "https://example.test", nil, false, false)
			if err != nil {
				t.Fatal(err)
			}
			if out["signature"] != "f80fb957067ab3f2f33a0b6ec529a053ff39c8a2" {
				t.Fatalf("signature = %v", out["signature"])
			}
			if calls != 2 {
				t.Fatalf("ticket calls = %d", calls)
			}
			if _, err := client.BuildConfig(context.Background(), "https://example.test", nil, false, false); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("valid ticket was not cached: calls=%d", calls)
			}
		})
	}
}
