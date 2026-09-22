package health_card

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

func TestAppTokenCacheOptions(t *testing.T) {
	client := New("app-id", "secret", "hospital", "related-app", WithCache(cache.NewMemoryCache()), WithCacheKeyPrefix("custom."), WithLocker(&lock.Mutex{}))
	if got := client.AppTokenCacheKey(); got != "custom.health_card_app_token.app-id" {
		t.Fatalf("cache key = %q", got)
	}
}

func TestAppTokenCanBeSeededIntoCache(t *testing.T) {
	c := cache.NewMemoryCache()
	client := New("app-id", "secret", "hospital", "related-app", WithCache(c), WithAppToken("seed-token"))
	other := New("app-id", "secret", "hospital", "related-app", WithCache(c), WithBaseURL("http://invalid.example"))
	got, err := other.AppToken()
	if err != nil || got != "seed-token" {
		t.Fatalf("token=%q err=%v", got, err)
	}
	if !c.IsExist(client.AppTokenCacheKey()) {
		t.Fatal("seed token was not written to cache")
	}
}

func TestAppTokenConcurrentRefreshUsesOneRequest(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(20 * time.Millisecond)
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0},"rsp":{"appToken":"shared-token","expiresIn":7200}}`)
	}))
	defer server.Close()

	client := New("app-id", "secret", "hospital", "related-app", WithBaseURL(server.URL), WithRequestID(func() string { return "rid" }))
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := client.AppToken(); err != nil || got != "shared-token" {
				errs <- fmt.Errorf("token=%q err=%v", got, err)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("token requests = %d, want 1", got)
	}
}

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
		if body.CommonIn.AppToken != "" || body.CommonIn.RelateAppID != "" || body.CommonIn.RelateOpenID != "" || body.Req.AppID != "app-id" || body.CommonIn.Sign == "" {
			t.Fatalf("unexpected request: %+v", body)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"appToken":"fresh-token","expiresIn":7200}}`)
	}))
	defer server.Close()

	client := New("app-id", "secret", "hospital", "related-app", WithBaseURL(server.URL), WithClock(func() time.Time { return time.Unix(1000, 0) }), WithRequestID(func() string { return "rid" }))
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
	client := New("app-id", "secret", "hospital", "related-app", WithBaseURL(server.URL), WithClock(func() time.Time { return now }), WithRequestID(func() string { return "rid" }))
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
