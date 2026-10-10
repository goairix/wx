package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
)

type callerFunc func(context.Context, request.Request) error

func (f callerFunc) Do(ctx context.Context, r request.Request) error { return f(ctx, r) }

func TestPlatformDecoderAndCustomCaller(t *testing.T) {
	for _, decoder := range []bool{true, false} {
		t.Run(map[bool]string{true: "decoder", false: "unmarshal fallback"}[decoder], func(t *testing.T) {
			manager := auth.NewManager("test", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
				return auth.Credential{AccessToken: "token"}, nil
			}))
			caller := callerFunc(func(_ context.Context, r request.Request) error {
				if r.Query.Get("access_token") != "token" {
					t.Fatal("missing token")
				}
				r.Meta.StatusCode = 202
				r.Meta.RequestID = "request-id"
				body := []byte(`{"errcode":40001,"errmsg":"  exact message  "}`)
				if decoder {
					d, ok := r.Result.(request.ResponseDecoder)
					if !ok {
						t.Fatal("platform result does not implement ResponseDecoder")
					}
					return d.DecodeResponse(body, *r.Meta)
				}
				return json.Unmarshal(body, r.Result)
			})
			err := New(caller, manager).Get(context.Background(), "test.operation", "test", nil, nil)
			var platformErr *wxerrors.Error
			if !errors.As(err, &platformErr) || platformErr.Code != "40001" || platformErr.Message != "  exact message  " || platformErr.HTTPStatus != 202 || platformErr.RequestID != "request-id" || platformErr.Platform != platform || platformErr.Operation != "test.operation" {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	}
}

type exactContextCache struct {
	cache.Cache
	t       *testing.T
	ctx     context.Context
	deletes int
}

func (c *exactContextCache) Get(ctx context.Context, key string) (string, bool, error) {
	if ctx != c.ctx {
		c.t.Error("cache Get replaced caller context")
	}
	return c.Cache.Get(ctx, key)
}

func (c *exactContextCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if ctx != c.ctx {
		c.t.Error("cache Put replaced caller context")
	}
	return c.Cache.Put(ctx, key, value, ttl)
}

func (c *exactContextCache) Delete(ctx context.Context, key string) error {
	if ctx != c.ctx {
		c.t.Error("cache Delete replaced caller context")
	}
	c.deletes++
	return c.Cache.Delete(ctx, key)
}

// Query mutation, stale envelopes, changed contexts and accidental network
// retry enablement each break the observable contract of the recovery attempt.
func TestRecoveryPreservesContextQueryAndRetryMode(t *testing.T) {
	for _, once := range []bool{false, true} {
		t.Run(map[bool]string{false: "get", true: "get once"}[once], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := &exactContextCache{Cache: cache.NewMemory(), t: t, ctx: ctx}
			refreshes := 0
			manager := auth.NewManager("miniapp", "app", store, auth.ProviderFunc(func(received context.Context) (auth.Credential, error) {
				if received != ctx {
					t.Error("provider replaced caller context")
				}
				refreshes++
				token := "old"
				if refreshes > 1 {
					token = "new"
				}
				return auth.Credential{AccessToken: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			}))
			calls := 0
			query := url.Values{"access_token": {"caller value"}, "ids": {"a", "b"}}
			caller := callerFunc(func(received context.Context, r request.Request) error {
				if received != ctx {
					t.Error("network replaced caller context")
				}
				calls++
				wantMode := request.RetryDefault
				if once {
					wantMode = request.RetryNever
				}
				if r.RetryMode != wantMode {
					t.Errorf("RetryMode=%v want %v", r.RetryMode, wantMode)
				}
				if len(r.Query["ids"]) != 2 || r.Query["ids"][0] != "a" {
					t.Errorf("query=%v", r.Query)
				}
				r.Query["ids"][0] = "mutated"
				if calls == 1 {
					if r.Query.Get("access_token") != "old" {
						t.Errorf("first token=%q", r.Query.Get("access_token"))
					}
					return json.Unmarshal([]byte(`{"errcode":42001,"errmsg":"expired"}`), r.Result)
				}
				if r.Query.Get("access_token") != "new" {
					t.Errorf("retry token=%q", r.Query.Get("access_token"))
				}
				return json.Unmarshal([]byte(`{"value":"fresh"}`), r.Result)
			})
			client := New(caller, manager)
			get := client.Get
			if once {
				get = client.GetOnce
			}
			var result struct {
				Value string `json:"value"`
			}
			if err := get(ctx, "miniapp.test.recovery", "path", query, &result); err != nil {
				t.Fatal(err)
			}
			if result.Value != "fresh" || calls != 2 || refreshes != 2 || store.deletes != 1 {
				t.Fatalf("result=%+v calls=%d refreshes=%d invalidations=%d", result, calls, refreshes, store.deletes)
			}
			if query.Get("access_token") != "caller value" || query["ids"][0] != "a" {
				t.Fatalf("caller query mutated: %v", query)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPlatformErrorReachesObserver(t *testing.T) {
	manager := auth.NewManager("test", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token"}, nil
	}))
	var observed observability.Event
	finished := 0
	h := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 202,
			Header:     http.Header{"X-Request-Id": []string{"response-id"}},
			Body:       io.NopCloser(strings.NewReader(`{"errcode":"40013","errmsg":"  exact message  "}`)),
		}, nil
	})}
	tr := transport.New(h, "https://example.test", transport.RetryPolicy{}, transport.WithObserver(
		observability.ObserverFunc(func(ctx context.Context, _ observability.Event) (context.Context, func(observability.Event)) {
			return ctx, func(e observability.Event) { finished++; observed = e }
		}),
	))
	err := New(tr, manager).Get(context.Background(), "test.observed", "test", nil, nil)
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || observed.Err != err || finished != 1 || observed.StatusCode != 202 || observed.RequestID != "response-id" {
		t.Fatalf("error=%v observed=%+v finished=%d", err, observed, finished)
	}
	if platformErr.Platform != platform || platformErr.Operation != "test.observed" || platformErr.Message != "  exact message  " {
		t.Fatalf("platform error=%+v", platformErr)
	}
}

func TestGetOnceDisablesRetryButGetRetainsRetry(t *testing.T) {
	for _, once := range []bool{false, true} {
		manager := auth.NewManager("test", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
			return auth.Credential{AccessToken: "token"}, nil
		}))
		calls := 0
		networkErr := errors.New("response lost")
		h := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, networkErr })}
		tr := transport.New(h, "https://example.test", transport.RetryPolicy{MaxAttempts: 3, Backoff: func(int) time.Duration { return 0 }})
		client := New(tr, manager)
		get := client.Get
		want := 3
		if once {
			get = client.GetOnce
			want = 1
		}
		err := get(context.Background(), "arbitrary.operation", "test", nil, nil)
		if !errors.Is(err, networkErr) || calls != want {
			t.Fatalf("once=%v calls=%d err=%v", once, calls, err)
		}
	}
}
