package health_card

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAppTokenFetchesAndCaches(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != getAppTokenPath {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			CommonIn CommonIn `json:"commonIn"`
			Req      struct {
				AppID string `json:"appId"`
			} `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.CommonIn.AppToken != "" || body.Req.AppID != "app-id" || body.CommonIn.Sign == "" {
			t.Fatalf("unexpected request: %+v", body)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"appToken":"fresh-token","expiresIn":7200}}`)
	}))
	defer server.Close()

	client := New("app-id", "secret", "hospital", WithBaseURL(server.URL), WithClock(func() time.Time { return time.Unix(1000, 0) }), WithRequestID(func() string { return "rid" }))
	first, err := client.AppToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.AppToken()
	if err != nil {
		t.Fatal(err)
	}
	if first != "fresh-token" || second != first || calls != 1 {
		t.Fatalf("token=%q/%q calls=%d", first, second, calls)
	}
}

func TestAppTokenRefreshesAfterExpiry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"appToken":"token-`+string(rune('0'+calls))+`","expiresIn":100}}`)
	}))
	defer server.Close()

	now := time.Unix(1000, 0)
	client := New("app-id", "secret", "hospital", WithBaseURL(server.URL), WithClock(func() time.Time { return now }), WithRequestID(func() string { return "rid" }))
	first, err := client.AppToken()
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(41 * time.Second)
	second, err := client.AppToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || calls != 2 {
		t.Fatalf("token=%q/%q calls=%d", first, second, calls)
	}
}
