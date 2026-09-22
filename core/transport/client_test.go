package transport

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

func TestClientStopsWhenContextIsCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := New(server.Client(), server.URL, RetryPolicy{MaxAttempts: 3, Backoff: func(int) time.Duration { return time.Hour }})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := client.Do(ctx, request.Request{Operation: "test.cancel", Method: http.MethodGet, Path: "/"})
	if !stderrors.Is(got, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", got)
	}
}

func TestClientRetriesOnlyConfiguredStatus(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errcode":45009,"errmsg":"busy"}`))
			return
		}
		w.Header().Set("X-Request-Id", "req-2")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var result struct {
		OK bool `json:"ok"`
	}
	var meta request.ResponseMeta
	err := New(server.Client(), server.URL, RetryPolicy{
		MaxAttempts: 2,
		RetryStatus: map[int]bool{http.StatusTooManyRequests: true},
		Backoff:     func(int) time.Duration { return 0 },
	}).Do(context.Background(), request.Request{Operation: "test.retry", Method: http.MethodGet, Path: "/", Result: &result, Meta: &meta})
	if err != nil || !result.OK || atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("err=%v result=%+v attempts=%d", err, result, attempts)
	}
	if meta.StatusCode != http.StatusOK || meta.RequestID != "req-2" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
}

func TestClientRetriesTransientNetworkErrors(t *testing.T) {
	var attempts int32
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			return nil, fmt.Errorf("temporary network error %d", attempt)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Request:    r,
		}, nil
	})
	client := New(&http.Client{Transport: transport}, "http://example.test", RetryPolicy{MaxAttempts: 3, Backoff: func(int) time.Duration { return 0 }})
	var result struct {
		OK bool `json:"ok"`
	}
	if err := client.Do(context.Background(), request.Request{Operation: "test.network-retry", Method: http.MethodGet, Path: "/", Result: &result}); err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if !result.OK || atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("result=%+v attempts=%d", result, attempts)
	}
}

func TestClientStopsNetworkRetryWhenContextCanceled(t *testing.T) {
	var attempts int32
	ctx, cancel := context.WithCancel(context.Background())
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(&attempts, 1)
		return nil, fmt.Errorf("temporary network error")
	})
	client := New(&http.Client{Transport: transport}, "http://example.test", RetryPolicy{
		MaxAttempts: 3,
		Backoff: func(int) time.Duration {
			cancel()
			return 0
		},
	})
	err := client.Do(ctx, request.Request{Operation: "test.network-cancel", Method: http.MethodGet, Path: "/"})
	if !stderrors.Is(err, context.Canceled) || atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("err=%v attempts=%d", err, attempts)
	}
}

func TestClientPreservesBasePathPrefixAndEscapedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/a/b" || r.URL.RawPath != "/v1/a%2Fb" {
			t.Errorf("path=%q rawPath=%q", r.URL.Path, r.URL.RawPath)
		}
		if r.URL.Query().Get("from") != "path" || r.URL.Query().Get("extra") != "query" {
			t.Errorf("query=%v", r.URL.Query())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := New(server.Client(), server.URL+"/v1", RetryPolicy{RetryStatus: map[int]bool{}}).Do(context.Background(), request.Request{
		Operation: "test.base-path",
		Method:    http.MethodGet,
		Path:      "/a%2Fb?from=path",
		Query:     url.Values{"extra": {"query"}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientDoesNotRetryWhenStatusIsNotConfigured(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errcode":45009,"errmsg":"busy"}`))
	}))
	defer server.Close()

	err := New(server.Client(), server.URL, RetryPolicy{MaxAttempts: 3, RetryStatus: map[int]bool{}, Backoff: func(int) time.Duration { return 0 }}).Do(context.Background(), request.Request{Operation: "test.no-retry", Method: http.MethodGet, Path: "/"})
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) || atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("err=%v attempts=%d", err, attempts)
	}
}

func TestClientReturnsPlatformErrorAndRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "rid-1")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"bad_request","message":"invalid"}`))
	}))
	defer server.Close()

	var meta request.ResponseMeta
	err := New(server.Client(), server.URL, RetryPolicy{RetryStatus: map[int]bool{}}).Do(context.Background(), request.Request{Operation: "test.error", Method: http.MethodGet, Path: "/", Meta: &meta})
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error type = %T, want *errors.Error: %v", err, err)
	}
	if platformErr.HTTPStatus != http.StatusBadRequest || platformErr.Code != "bad_request" || platformErr.RequestID != "rid-1" {
		t.Fatalf("unexpected platform error: %+v", platformErr)
	}
	if meta.StatusCode != http.StatusBadRequest || meta.RequestID != "rid-1" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
}

func TestClientEncodesJSONBodyWithoutHTMLEscaping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(body) != "{\"value\":\"<tag>\"}\n" {
			t.Errorf("body = %q", body)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := New(server.Client(), server.URL, RetryPolicy{RetryStatus: map[int]bool{}}).Do(context.Background(), request.Request{Operation: "test.body", Method: http.MethodPost, Path: "/", Body: struct {
		Value string `json:"value"`
	}{Value: "<tag>"}})
	if err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
