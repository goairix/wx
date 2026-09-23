package transport

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
)

type recordedLog struct {
	level logging.Level
	event string
	attrs []logging.Attr
}

type recordingLogger struct {
	mu     sync.Mutex
	events []recordedLog
}

func (logger *recordingLogger) Log(_ context.Context, level logging.Level, event string, attrs ...logging.Attr) {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	logger.events = append(logger.events, recordedLog{
		level: level,
		event: event,
		attrs: append([]logging.Attr(nil), attrs...),
	})
}

func (logger *recordingLogger) snapshot() []recordedLog {
	logger.mu.Lock()
	defer logger.mu.Unlock()
	return append([]recordedLog(nil), logger.events...)
}

func TestClientLogsSuccessfulRequestLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "rid-success")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := &recordingLogger{}
	client := New(server.Client(), server.URL, RetryPolicy{}, WithLogger(logger))
	err := client.Do(context.Background(), request.Request{
		Operation: "miniapp.auth.code2session",
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "/session",
	})
	if err != nil {
		t.Fatal(err)
	}

	events := logger.snapshot()
	assertLogSequence(t, events,
		[]string{"wx.request.started", "wx.request.completed"},
		[]logging.Level{logging.LevelDebug, logging.LevelDebug},
	)
	assertAttrs(t, events[0], map[string]interface{}{
		"platform":     "miniapp",
		"operation":    "miniapp.auth.code2session",
		"method":       http.MethodGet,
		"attempt":      1,
		"max_attempts": 1,
	})
	assertAttrs(t, events[1], map[string]interface{}{
		"status":     http.StatusNoContent,
		"request_id": "rid-success",
	})
}

func TestClientLogsRetryLifecycle(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errcode":45009,"errmsg":"busy"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := &recordingLogger{}
	client := New(server.Client(), server.URL, RetryPolicy{
		MaxAttempts: 2,
		RetryStatus: map[int]bool{http.StatusTooManyRequests: true},
		Backoff:     func(int) time.Duration { return time.Millisecond },
	}, WithLogger(logger))
	if err := client.Do(context.Background(), request.Request{
		Operation: "official.token.get",
		Platform:  "official",
		Method:    http.MethodGet,
		Path:      "/token",
	}); err != nil {
		t.Fatal(err)
	}

	events := logger.snapshot()
	assertLogSequence(t, events,
		[]string{"wx.request.started", "wx.request.retrying", "wx.request.started", "wx.request.completed"},
		[]logging.Level{logging.LevelDebug, logging.LevelWarn, logging.LevelDebug, logging.LevelDebug},
	)
	assertAttrs(t, events[1], map[string]interface{}{
		"status":       http.StatusTooManyRequests,
		"code":         "45009",
		"attempt":      1,
		"max_attempts": 2,
		"next_attempt": 2,
		"retry_delay":  time.Millisecond,
	})
}

func TestClientLogsFinalNetworkFailure(t *testing.T) {
	networkError := stderrors.New("network unavailable")
	httpTransport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, networkError
	})
	logger := &recordingLogger{}
	client := New(
		&http.Client{Transport: httpTransport},
		"http://example.test",
		RetryPolicy{},
		WithLogger(logger),
	)

	err := client.Do(context.Background(), request.Request{
		Operation: "work.message.send",
		Platform:  "work",
		Method:    http.MethodPost,
		Path:      "/message",
	})
	if !stderrors.Is(err, networkError) {
		t.Fatalf("Do() error = %v", err)
	}

	events := logger.snapshot()
	assertLogSequence(t, events,
		[]string{"wx.request.started", "wx.request.failed"},
		[]logging.Level{logging.LevelDebug, logging.LevelError},
	)
	loggedErr, ok := attrsMap(events[1])["error"].(error)
	if !ok || !stderrors.Is(loggedErr, networkError) {
		t.Fatalf("logged error = %T %v, want wrapped network error", loggedErr, loggedErr)
	}
}

func TestClientLogsPreCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	logger := &recordingLogger{}
	client := New(nil, "http://example.test", RetryPolicy{}, WithLogger(logger))

	err := client.Do(ctx, request.Request{
		Operation: "miniapp.auth.code2session",
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "/session",
	})
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v", err)
	}

	events := logger.snapshot()
	assertLogSequence(t, events, []string{"wx.request.failed"}, []logging.Level{logging.LevelError})
	assertAttrs(t, events[0], map[string]interface{}{"error": context.Canceled})
}

func TestClientLoggerAndHookCoexist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	logger := &recordingLogger{}
	var hookEvents int32
	client := New(
		server.Client(),
		server.URL,
		RetryPolicy{},
		WithLogger(logger),
		WithHook(observability.HookFunc(func(observability.Event) {
			atomic.AddInt32(&hookEvents, 1)
		})),
	)
	if err := client.Do(context.Background(), request.Request{
		Operation: "test.coexist",
		Method:    http.MethodGet,
		Path:      "/",
	}); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&hookEvents); got != 2 {
		t.Fatalf("hook event count = %d, want 2", got)
	}
	if got := len(logger.snapshot()); got != 2 {
		t.Fatalf("log event count = %d, want 2", got)
	}
}

func TestClientLogsDoNotExposeRequestSecrets(t *testing.T) {
	const secret = "secret-sentinel-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errcode":-1,"errmsg":"system busy"}`))
	}))
	defer server.Close()

	logger := &recordingLogger{}
	client := New(server.Client(), server.URL, RetryPolicy{RetryStatus: map[int]bool{}}, WithLogger(logger))
	_ = client.Do(context.Background(), request.Request{
		Operation: "official.message.send",
		Platform:  "official",
		Method:    http.MethodPost,
		Path:      "/message?access_token=" + secret,
		Query:     url.Values{"code": {secret}},
		Header:    http.Header{"Authorization": {"Bearer " + secret}},
		Body:      map[string]string{"openid": secret},
	})

	for _, event := range logger.snapshot() {
		if strings.Contains(event.event, secret) {
			t.Fatalf("event name exposes secret: %q", event.event)
		}
		for _, attr := range event.attrs {
			if strings.Contains(attr.Key, secret) || strings.Contains(toString(attr.Value), secret) {
				t.Fatalf("attribute exposes secret: %#v", attr)
			}
		}
	}
}

func TestClientNetworkFailureLogDoesNotExposeRequestURL(t *testing.T) {
	const secret = "secret-query-value"
	httpTransport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, stderrors.New("connection refused")
	})
	logger := &recordingLogger{}
	client := New(
		&http.Client{Transport: httpTransport},
		"http://example.test",
		RetryPolicy{},
		WithLogger(logger),
	)

	err := client.Do(context.Background(), request.Request{
		Operation: "miniapp.auth.code2session",
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "/session?code=" + secret,
	})
	if err == nil || !strings.Contains(err.Error(), secret) {
		t.Fatalf("returned error should remain unchanged: %v", err)
	}

	for _, event := range logger.snapshot() {
		for _, attr := range event.attrs {
			if strings.Contains(toString(attr.Value), secret) {
				t.Fatalf("logged attribute exposes request URL: %#v", attr)
			}
		}
	}
}

func TestClientPreservesPlatformErrorTextInLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code, rid: original-rid"}`))
	}))
	defer server.Close()

	logger := &recordingLogger{}
	client := New(server.Client(), server.URL, RetryPolicy{}, WithLogger(logger))
	err := client.Do(context.Background(), request.Request{
		Operation: "miniapp.auth.code2session",
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "/session",
	})

	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("Do() error = %T %v", err, err)
	}
	const want = "miniapp miniapp.auth.code2session: invalid code, rid: original-rid"
	if got := err.Error(); got != want {
		t.Fatalf("error text = %q, want %q", got, want)
	}

	events := logger.snapshot()
	assertLogSequence(t, events,
		[]string{"wx.request.started", "wx.request.failed"},
		[]logging.Level{logging.LevelDebug, logging.LevelError},
	)
	assertAttrs(t, events[1], map[string]interface{}{
		"code":  "40029",
		"error": err,
	})
	if _, exists := attrsMap(events[1])["request_id"]; exists {
		t.Fatal("request_id was invented from the error message")
	}
	if got := toString(attrsMap(events[1])["error"]); got != want {
		t.Fatalf("logged error text = %q, want %q", got, want)
	}
}

func assertLogSequence(t *testing.T, events []recordedLog, names []string, levels []logging.Level) {
	t.Helper()
	if len(events) != len(names) {
		t.Fatalf("event count = %d, want %d: %#v", len(events), len(names), events)
	}
	for index := range names {
		if events[index].event != names[index] || events[index].level != levels[index] {
			t.Errorf("event %d = (%s, %s), want (%s, %s)", index, events[index].level, events[index].event, levels[index], names[index])
		}
	}
}

func assertAttrs(t *testing.T, event recordedLog, want map[string]interface{}) {
	t.Helper()
	got := attrsMap(event)
	for key, value := range want {
		if !reflect.DeepEqual(got[key], value) {
			t.Errorf("%s attribute %q = %#v, want %#v", event.event, key, got[key], value)
		}
	}
}

func attrsMap(event recordedLog) map[string]interface{} {
	result := make(map[string]interface{}, len(event.attrs))
	for _, attr := range event.attrs {
		result[attr.Key] = attr.Value
	}
	return result
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if err, ok := value.(error); ok {
		return err.Error()
	}
	if stringer, ok := value.(interface{ String() string }); ok {
		return stringer.String()
	}
	if reader, ok := value.(io.Reader); ok {
		data, _ := io.ReadAll(reader)
		return string(data)
	}
	return ""
}
