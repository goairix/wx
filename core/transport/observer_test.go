package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
)

type observerContextKey struct{}
type contextLogger struct {
	t     *testing.T
	count int
}

func (l *contextLogger) Log(ctx context.Context, _ logging.Level, _ string, _ ...logging.Attr) {
	if ctx.Value(observerContextKey{}) != "span" {
		l.t.Error("logger did not receive observer context")
	}
	l.count++
}

func TestObserverPropagatesContextAndCoexistsWithHook(t *testing.T) {
	var start, finish observability.Event
	var hooks []observability.Event
	logger := &contextLogger{t: t}
	client := New(
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Context().Value(observerContextKey{}) != "span" {
				t.Error("HTTP request did not receive observer context")
			}
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{"X-Request-Id": {"rid"}},
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Request:    r,
			}, nil
		})},
		"http://example.test",
		RetryPolicy{},
		WithLogger(logger),
		WithHook(observability.HookFunc(func(e observability.Event) { hooks = append(hooks, e) })),
		WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
			start = e
			return context.WithValue(ctx, observerContextKey{}, "span"), func(e observability.Event) { finish = e }
		})),
	)
	if err := client.Do(context.Background(), request.Request{
		Operation: "same.operation",
		Platform:  "test",
		Method:    http.MethodGet,
		Result:    &struct{}{},
	}); err != nil {
		t.Fatal(err)
	}
	if start.Operation != "same.operation" || start.Attempt != 1 || start.MaxAttempts != 1 {
		t.Fatalf("start=%+v", start)
	}
	if finish.StatusCode != 200 || finish.RequestID != "rid" || finish.Err != nil || finish.Duration <= 0 {
		t.Fatalf("start=%+v finish=%+v", start, finish)
	}
	if len(hooks) != 2 || logger.count != 2 {
		t.Fatalf("hooks=%+v log count=%d", hooks, logger.count)
	}
	for _, e := range append(hooks, finish) {
		if e.Context.Value(observerContextKey{}) != "span" {
			t.Error("event context missing span")
		}
	}
}

func TestObserverFinishesEveryOutcome(t *testing.T) {
	sentinel := errors.New("failure")
	for _, tc := range []struct {
		name, method, body string
		status             int
		network            error
		writer             io.Writer
		result             interface{}
		cancel             bool
	}{
		{
			name:   "success",
			status: 204,
		}, {
			name:    "network",
			network: sentinel,
		}, {
			name:   "HTTP",
			status: 503,
		},
		{
			name:   "platform",
			status: 200,
			body:   `{"errcode":40013,"errmsg":"bad"}`,
		},
		{
			name:   "decode",
			status: 200,
			body:   `{`,
			result: &struct{}{},
		},
		{
			name:   "custom decode",
			status: 200,
			body:   `{}`,
			result: &responseDecoder{err: sentinel},
		},
		{
			name:   "stream",
			status: 200,
			body:   "binary",
			writer: observerFailWriter{sentinel},
		},
		{
			name:   "construction",
			method: "bad method",
		}, {
			name:   "canceled",
			cancel: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			starts, finishes := 0, 0
			var final observability.Event
			client := New(
				&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if tc.network != nil {
						return nil, tc.network
					}
					return &http.Response{
						StatusCode: tc.status,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(tc.body)),
						Request:    r,
					}, nil
				})},
				"http://example.test",
				RetryPolicy{},
				WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
					starts++
					if tc.cancel {
						var cancel context.CancelFunc
						ctx, cancel = context.WithCancel(ctx)
						cancel()
					}
					return ctx, func(e observability.Event) { finishes++; final = e }
				})),
			)
			method := tc.method
			if method == "" {
				method = http.MethodGet
			}
			err := client.Do(context.Background(), request.Request{
				Method:         method,
				Result:         tc.result,
				ResponseWriter: tc.writer,
			})
			if starts != 1 || finishes != 1 {
				t.Fatalf("starts=%d finishes=%d", starts, finishes)
			}
			if tc.name == "success" {
				if err != nil || final.Err != nil {
					t.Fatalf("err=%v event=%+v", err, final)
				}
			} else if err == nil || final.Err == nil {
				t.Fatalf("err=%v event=%+v", err, final)
			}
			if tc.cancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

type observerFailWriter struct{ err error }

func (w observerFailWriter) Write([]byte) (int, error) { return 0, w.err }

func TestObserverRetryAttempts(t *testing.T) {
	var count int
	var events []observability.Event
	client := New(
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			count++
			if count == 1 {
				return nil, errors.New("temporary")
			}
			status := 200
			if count == 2 {
				status = 503
			}
			return &http.Response{
				StatusCode: status,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Request:    r,
			}, nil
		})},
		"http://example.test",
		RetryPolicy{
			MaxAttempts: 3,
			Backoff:     func(int) time.Duration { return 0 },
		},
		WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
			events = append(events, e)
			return ctx, func(e observability.Event) { events = append(events, e) }
		})),
	)
	if err := client.Do(context.Background(), request.Request{Method: http.MethodGet}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 6 {
		t.Fatalf("events=%+v", events)
	}
	for i, e := range events {
		if e.Attempt != i/2+1 || e.MaxAttempts != 3 {
			t.Fatalf("event %d=%+v", i, e)
		}
	}
	if events[1].Err == nil || events[3].Err == nil || events[5].Err != nil {
		t.Fatalf("events=%+v", events)
	}
}

func TestObserverConcurrentSameOperation(t *testing.T) {
	var sequence int64
	var mu sync.Mutex
	finished := map[int64]bool{}
	const requests = 32
	client := responseClient(
		204,
		"",
		WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
			id := atomic.AddInt64(&sequence, 1)
			return context.WithValue(ctx, observerContextKey{}, id), func(e observability.Event) {
				mu.Lock()
				defer mu.Unlock()
				if e.Context.Value(observerContextKey{}) != id || finished[id] || e.Operation != "same.operation" {
					t.Errorf("mismatched completion %d: %+v", id, e)
				}
				finished[id] = true
			}
		})),
	)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.Do(context.Background(), request.Request{
				Method:    http.MethodGet,
				Operation: "same.operation",
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(finished) != requests {
		t.Fatalf("finished=%d", len(finished))
	}
}

func TestObserverDerivedCancellationDuringRequestStopsRetries(t *testing.T) {
	var cancel context.CancelFunc
	var attempts, finishes int
	client := New(
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			cancel()
			return nil, errors.New("connection ended")
		})},
		"http://example.test",
		RetryPolicy{MaxAttempts: 3},
		WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
			derived, stop := context.WithCancel(ctx)
			cancel = stop
			return derived, func(e observability.Event) {
				finishes++
				if !errors.Is(e.Err, context.Canceled) {
					t.Errorf("finish error = %v", e.Err)
				}
			}
		})),
	)
	if err := client.Do(context.Background(), request.Request{Method: http.MethodGet}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if attempts != 1 || finishes != 1 {
		t.Fatalf("attempts=%d finishes=%d", attempts, finishes)
	}
}

func TestObserverNetworkErrorDoesNotExposeURL(t *testing.T) {
	sentinel := errors.New("network failure")
	var finished observability.Event
	client := New(
		&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, sentinel })},
		"http://example.test",
		RetryPolicy{},
		WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
			return ctx, func(e observability.Event) { finished = e }
		})),
	)
	err := client.Do(context.Background(), request.Request{
		Method: http.MethodGet,
		Path:   "/?access_token=secret-value",
	})
	if !errors.Is(err, sentinel) || !errors.Is(finished.Err, sentinel) {
		t.Fatalf("error=%v event=%+v", err, finished)
	}
	if strings.Contains(finished.Err.Error(), "secret-value") {
		t.Fatalf("observer leaked URL: %v", finished.Err)
	}
}

func TestObserverFinishesReadErrorAndAllowsNilReturns(t *testing.T) {
	sentinel := errors.New("read failed")
	finishes := 0
	client := New(
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       io.NopCloser(observerFailReader{sentinel}),
				Request:    r,
			}, nil
		})},
		"http://example.test",
		RetryPolicy{},
		WithObserver(observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
			return nil, func(e observability.Event) {
				finishes++
				if !errors.Is(e.Err, sentinel) {
					t.Errorf("finish error=%v", e.Err)
				}
			}
		})),
	)
	if err := client.Do(context.Background(), request.Request{Method: http.MethodGet}); !errors.Is(err, sentinel) {
		t.Fatalf("error=%v", err)
	}
	if finishes != 1 {
		t.Fatalf("finishes=%d", finishes)
	}
	client = responseClient(
		204,
		"",
		WithObserver(observability.ObserverFunc(func(context.Context, observability.Event) (context.Context, func(observability.Event)) {
			return nil, nil
		})),
	)
	if err := client.Do(context.Background(), request.Request{Method: http.MethodGet}); err != nil {
		t.Fatal(err)
	}
}

type observerFailReader struct{ err error }

func (r observerFailReader) Read([]byte) (int, error) { return 0, r.err }

func TestObserverFinishesURLResolutionErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		path string
	}{
		{
			name: "base",
			base: "http://[invalid",
			path: "/",
		},
		{
			name: "path",
			base: "http://example.test",
			path: "/%zz",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			starts, finishes := 0, 0
			var final observability.Event
			observer := observability.ObserverFunc(func(ctx context.Context, event observability.Event) (context.Context, func(observability.Event)) {
				starts++
				return context.WithValue(ctx, observerContextKey{}, "span"), func(event observability.Event) {
					finishes++
					final = event
				}
			})
			logger := &contextLogger{t: t}
			client := New(
				nil,
				tc.base,
				RetryPolicy{MaxAttempts: 3},
				WithObserver(observer),
				WithLogger(logger),
			)
			err := client.Do(context.Background(), request.Request{
				Operation: "test.invalid-url",
				Method:    http.MethodGet,
				Path:      tc.path,
			})
			if err == nil || final.Err == nil {
				t.Fatalf("error=%v event=%+v", err, final)
			}
			if starts != 1 || finishes != 1 || final.Attempt != 1 {
				t.Fatalf("starts=%d finishes=%d event=%+v", starts, finishes, final)
			}
			if final.Context.Value(observerContextKey{}) != "span" {
				t.Fatal("finish did not preserve derived context")
			}
		})
	}
}
