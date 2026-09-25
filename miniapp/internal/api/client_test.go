package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
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
