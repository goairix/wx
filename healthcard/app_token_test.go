package healthcard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/goairix/wx/v2/core/cache"
)

func TestAppTokenCanBeSharedThroughCoreCache(t *testing.T) {
	shared := cache.NewMemory()
	first, err := NewClient(
		testConfig(),
		WithCache(shared),
		WithAppToken("seed-token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.AppToken(context.Background()); err != nil {
		t.Fatal(err)
	}

	second, err := NewClient(
		testConfig(),
		WithCache(shared),
		WithBaseURL("http://invalid.example"),
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err := second.AppToken(context.Background())
	if err != nil || got != "seed-token" {
		t.Fatalf("token=%q err=%v", got, err)
	}
}

func TestAppTokenConcurrentRefreshUsesOneRequest(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = io.WriteString(
			w,
			`{"commonOut":{"requestId":"rid","resultCode":0},"rsp":{"appToken":"shared-token","expiresIn":7200}}`,
		)
	}))
	defer server.Close()

	client, err := NewClient(
		testConfig(),
		WithBaseURL(server.URL),
		WithRequestID(func() string { return "rid" }),
	)
	if err != nil {
		t.Fatal(err)
	}

	var group sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			got, tokenErr := client.AppToken(context.Background())
			if tokenErr != nil || got != "shared-token" {
				errs <- fmt.Errorf("token=%q err=%v", got, tokenErr)
			}
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("token requests = %d, want 1", got)
	}
}

func TestAppTokenRequestPreservesHealthcardEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if body.CommonIn.AppToken != "" ||
			body.CommonIn.RelateAppID != "" ||
			body.CommonIn.RelateOpenID != "" ||
			body.Req.AppID != "app" ||
			body.CommonIn.Sign == "" {
			t.Fatalf("unexpected request: %+v", body)
		}
		_, _ = io.WriteString(
			w,
			`{"commonOut":{"requestId":"rid","resultCode":0},"rsp":{"appToken":"fresh-token","expiresIn":7200}}`,
		)
	}))
	defer server.Close()

	client, err := NewClient(
		testConfig(),
		WithBaseURL(server.URL),
		WithRequestID(func() string { return "rid" }),
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.AppToken(context.Background())
	if err != nil || got != "fresh-token" {
		t.Fatalf("token=%q err=%v", got, err)
	}
}

func TestAppTokenRefreshesWhenCredentialIsInsideRefreshWindow(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if r.URL.Path != getAppTokenPath {
			t.Errorf("path = %q, want %q", r.URL.Path, getAppTokenPath)
		}

		var body struct {
			CommonIn CommonIn `json:"commonIn"`
			Req      struct {
				AppID string `json:"appId"`
			} `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.CommonIn.AppToken != "" ||
			body.CommonIn.RequestID == "" ||
			body.CommonIn.Sign == "" ||
			body.Req.AppID != "app" {
			t.Errorf("unexpected request envelope: %+v", body)
		}

		_, _ = fmt.Fprintf(
			w,
			`{"commonOut":{"requestId":"rid-%d","resultCode":0},"rsp":{"appToken":"token-%d","expiresIn":60}}`,
			call,
			call,
		)
	}))
	defer server.Close()

	client, err := NewClient(
		testConfig(),
		WithBaseURL(server.URL),
		WithRequestID(func() string { return "request-id" }),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	first, err := client.AppToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.AppToken(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first != "token-1" || second != "token-2" {
		t.Fatalf("tokens = %q, %q", first, second)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("token requests = %d, want 2", got)
	}
}
