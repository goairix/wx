package integration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/work"
)

func TestDomainMutationsNeverRetry(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		call func(*official.Client, *work.Client) error
	}{
		{"menu delete", func(o *official.Client, _ *work.Client) error { return o.Menu().Delete(ctx) }},
		{"user delete", func(_ *official.Client, w *work.Client) error { return w.Contact().Users().Delete(ctx, "user") }},
		{"department delete", func(_ *official.Client, w *work.Client) error { return w.Contact().Departments().Delete(ctx, 1) }},
		{"tag delete", func(_ *official.Client, w *work.Client) error { return w.Contact().Tags().Delete(ctx, 1) }},
		{"work miniapp code", func(_ *official.Client, w *work.Client) error { _, err := w.MiniApp().Session(ctx, "code"); return err }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			lostResponse := errors.New("lost response after mutation")
			h := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/cgi-bin/gettoken" || r.URL.Path == "/cgi-bin/token" {
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"token","expires_in":7200}`))}, nil
				}
				attempts++
				return nil, lostResponse
			})}
			policy := transport.RetryPolicy{MaxAttempts: 3, Backoff: func(int) time.Duration { return 0 }}
			o, err := official.NewClient(official.Config{AppID: "app", AppSecret: "secret"}, official.WithHTTPClient(h), official.WithRetryPolicy(policy))
			if err != nil {
				t.Fatal(err)
			}
			w, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithHTTPClient(h), work.WithRetryPolicy(policy))
			if err != nil {
				t.Fatal(err)
			}
			err = test.call(o, w)
			if !errors.Is(err, lostResponse) || attempts != 1 {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
		})
	}
}
