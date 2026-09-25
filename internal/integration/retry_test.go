package integration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/transport"
	miniappauth "github.com/goairix/wx/v2/miniapp/auth"
	"github.com/goairix/wx/v2/miniapp/multiterminal"
	mobileoauth "github.com/goairix/wx/v2/mobileapp/oauth"
	officialoauth "github.com/goairix/wx/v2/official/oauth"
	"github.com/goairix/wx/v2/work"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOneTimeExchangesDoNotRetry(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		call func(*http.Client, transport.RetryPolicy) error
	}{
		{"miniapp code", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			_, err := miniappauth.New(transport.New(httpClient, "https://example.test", policy), miniappauth.Config{AppID: "app", AppSecret: "secret"}).Code2Session(ctx, "code")
			return err
		}},
		{"multiterminal code", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			_, err := multiterminal.NewClient("app", "secret", transport.New(httpClient, "https://example.test", policy)).CodeToVerifyInfo(ctx, "code")
			return err
		}},
		{"official oauth code", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			_, err := officialoauth.NewClient(transport.New(httpClient, "https://example.test", policy), "app", "secret").TokenFromCode(ctx, "code")
			return err
		}},
		{"component oauth code", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			provider := auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
				return auth.Credential{AccessToken: "component-token"}, nil
			})
			_, err := officialoauth.NewClient(transport.New(httpClient, "https://example.test", policy), "app", "", officialoauth.WithComponent("component", provider)).TokenFromCode(ctx, "code")
			return err
		}},
		{"mobile oauth code", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			_, err := mobileoauth.New(transport.New(httpClient, "https://example.test", policy), mobileoauth.Config{AppID: "app", AppSecret: "secret"}, nil).TokenFromCode(ctx, "code")
			return err
		}},
		{"work oauth code", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithHTTPClient(httpClient), work.WithRetryPolicy(policy))
			if err != nil {
				return err
			}
			_, err = client.Auth().UserFromCode(ctx, "code")
			return err
		}},
		{"work login confirmation", func(httpClient *http.Client, policy transport.RetryPolicy) error {
			client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithHTTPClient(httpClient), work.WithRetryPolicy(policy))
			if err != nil {
				return err
			}
			return client.Auth().ConfirmLoginTFA(ctx, "user")
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			lostResponse := errors.New("response lost after consuming code")
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/cgi-bin/gettoken" {
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"token","expires_in":7200}`))}, nil
				}
				attempts++
				return nil, lostResponse
			})}
			err := test.call(client, transport.RetryPolicy{MaxAttempts: 3, Backoff: func(int) time.Duration { return 0 }})
			if !errors.Is(err, lostResponse) || attempts != 1 {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
		})
	}
}

func TestCodeExchangeCannotBeReplayedByStandardTransport(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/warm" {
			_, _ = io.WriteString(w, "ready")
			return
		}
		if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
			t.Errorf("empty GET framing changed: length=%d transfer=%v", r.ContentLength, r.TransferEncoding)
		}
		if attempts.Add(1) == 1 {
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = connection.Close()
			return
		}
		_, _ = io.WriteString(w, `{"openid":"user","session_key":"session"}`)
	}))
	defer server.Close()
	httpClient := server.Client()
	defer httpClient.CloseIdleConnections()
	warmup, err := httpClient.Get(server.URL + "/warm")
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(io.Discard, warmup.Body)
	_ = warmup.Body.Close()
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	client := miniappauth.New(
		transport.New(httpClient, server.URL, transport.RetryPolicy{MaxAttempts: 1}),
		miniappauth.Config{AppID: "app", AppSecret: "secret"},
	)
	_, err = client.Code2Session(context.Background(), "one-time-code")
	if err == nil || attempts.Load() != 1 {
		t.Fatalf("attempts=%d err=%v; consumed code must not be replayed", attempts.Load(), err)
	}
}
