package transport

import (
	"bytes"
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
	"github.com/goairix/wx/v2/core/observability"
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

func TestClientDoesNotRetryPostByDefault(t *testing.T) {
	var attempts int32
	httpTransport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		atomic.AddInt32(&attempts, 1)
		return nil, fmt.Errorf("connection reset")
	})
	client := New(
		&http.Client{Transport: httpTransport},
		"http://example.test",
		RetryPolicy{
			MaxAttempts: 3,
			Backoff:     func(int) time.Duration { return 0 },
		},
	)

	err := client.Do(context.Background(), request.Request{
		Operation: "test.post-no-retry",
		Method:    http.MethodPost,
		Path:      "/messages",
		Body:      map[string]string{"message": "hello"},
	})
	if err == nil {
		t.Fatal("Do() error = nil")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestClientRetriesPostWhenExplicitlyAllowed(t *testing.T) {
	var attempts int32
	httpTransport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			return nil, fmt.Errorf("connection reset")
		}
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})
	client := New(
		&http.Client{Transport: httpTransport},
		"http://example.test",
		RetryPolicy{
			MaxAttempts: 2,
			Backoff:     func(int) time.Duration { return 0 },
		},
	)

	err := client.Do(context.Background(), request.Request{
		Operation: "test.post-explicit-retry",
		Method:    http.MethodPost,
		Path:      "/idempotent-operation",
		RetryMode: request.RetryAlways,
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
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

func TestClientReportsNetworkFailureToHook(t *testing.T) {
	networkError := stderrors.New("network unavailable")
	httpTransport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, networkError
	})
	var responseEvent observability.Event
	client := New(
		&http.Client{Transport: httpTransport},
		"http://example.test",
		RetryPolicy{},
		WithHook(observability.HookFunc(func(event observability.Event) {
			if event.Err != nil {
				responseEvent = event
			}
		})),
	)

	err := client.Do(context.Background(), request.Request{
		Operation: "test.network-observation",
		Platform:  "test",
		Method:    http.MethodGet,
		Path:      "/",
	})
	if !stderrors.Is(err, networkError) {
		t.Fatalf("Do() error = %v", err)
	}
	if !stderrors.Is(responseEvent.Err, networkError) {
		t.Fatalf("response hook error = %v", responseEvent.Err)
	}
	if responseEvent.Operation != "test.network-observation" {
		t.Fatalf("response hook operation = %q", responseEvent.Operation)
	}
}

func TestClientReportsSuccessfulHTTPPlatformFailureToHook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer server.Close()

	var responseEvent observability.Event
	client := New(
		server.Client(),
		server.URL,
		RetryPolicy{},
		WithHook(observability.HookFunc(func(event observability.Event) {
			if event.StatusCode != 0 {
				responseEvent = event
			}
		})),
	)
	var result map[string]interface{}
	err := client.Do(context.Background(), request.Request{
		Operation: "test.platform-error",
		Platform:  "official",
		Method:    http.MethodGet,
		Path:      "/",
		Result:    &result,
	})
	var platformError *wxerrors.Error
	if !stderrors.As(err, &platformError) {
		t.Fatalf("Do() error = %T %v, want *errors.Error", err, err)
	}
	if !stderrors.As(responseEvent.Err, &platformError) {
		t.Fatalf("response hook error = %T %v", responseEvent.Err, responseEvent.Err)
	}
	if responseEvent.StatusCode != http.StatusOK {
		t.Fatalf("response hook status = %d", responseEvent.StatusCode)
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

func TestClientRejectsResponseLargerThanConfiguredLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("12345"))
	}))
	defer server.Close()

	client := New(
		server.Client(),
		server.URL,
		RetryPolicy{},
		WithMaxResponseBytes(4),
	)
	var result []byte
	err := client.Do(context.Background(), request.Request{
		Operation: "test.response-limit",
		Method:    http.MethodGet,
		Path:      "/",
		Result:    &result,
	})
	if !stderrors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("Do() error = %v, want ErrResponseTooLarge", err)
	}
}

func TestClientStreamsBinaryResponseToWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("streamed-media"))
	}))
	defer server.Close()

	var destination bytes.Buffer
	err := New(server.Client(), server.URL, RetryPolicy{}).Do(
		context.Background(),
		request.Request{
			Operation:      "test.stream",
			Method:         http.MethodGet,
			Path:           "/",
			ResponseWriter: &destination,
		},
	)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if got := destination.String(); got != "streamed-media" {
		t.Fatalf("streamed body = %q", got)
	}
}

func TestStreamingResponseDoesNotUseBufferedResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("12345"))
	}))
	defer server.Close()

	var destination bytes.Buffer
	err := New(
		server.Client(),
		server.URL,
		RetryPolicy{},
		WithMaxResponseBytes(4),
	).Do(context.Background(), request.Request{
		Operation:      "test.unbounded-stream",
		Method:         http.MethodGet,
		Path:           "/",
		ResponseWriter: &destination,
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if got := destination.String(); got != "12345" {
		t.Fatalf("streamed body = %q", got)
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

func TestClientPropagatesContextToHook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "trace-sentinel")
	var events []observability.Event
	client := New(
		server.Client(),
		server.URL,
		RetryPolicy{},
		WithHook(observability.HookFunc(func(event observability.Event) {
			events = append(events, event)
		})),
	)

	if err := client.Do(ctx, request.Request{
		Operation: "test.context-hook",
		Method:    http.MethodGet,
		Path:      "/",
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("hook event count = %d, want 2", len(events))
	}
	for index, event := range events {
		if event.Context != ctx {
			t.Errorf("event %d context was not propagated", index)
		}
		if got := event.Context.Value(contextKey{}); got != "trace-sentinel" {
			t.Errorf("event %d context value = %v", index, got)
		}
	}
}

func TestClientNormalizesNilContextForHook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var events []observability.Event
	client := New(
		server.Client(),
		server.URL,
		RetryPolicy{},
		WithHook(observability.HookFunc(func(event observability.Event) {
			events = append(events, event)
		})),
	)

	if err := client.Do(nil, request.Request{
		Operation: "test.nil-context-hook",
		Method:    http.MethodGet,
		Path:      "/",
	}); err != nil {
		t.Fatal(err)
	}
	for index, event := range events {
		if event.Context == nil {
			t.Errorf("event %d has nil context", index)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
