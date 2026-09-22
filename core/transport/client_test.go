package transport

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
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
