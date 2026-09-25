package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/healthcard"
	"github.com/goairix/wx/v2/miniapp"
	"github.com/goairix/wx/v2/mobileapp"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/openplatform"
	"github.com/goairix/wx/v2/work"
)

type observerKey struct{}

func TestPlatformsForwardObserver(t *testing.T) {
	cases := []struct {
		name string
		call func(context.Context, *http.Client, observability.Observer) error
	}{
		{"official", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := official.NewClient(official.Config{AppID: "app", AppSecret: "secret"}, official.WithHTTPClient(h), official.WithObserver(o))
			if err != nil {
				return err
			}
			_, err = c.Users().Info(ctx, "user")
			return err
		}},
		{"miniapp", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := miniapp.NewClient(miniapp.Config{AppID: "app", AppSecret: "secret"}, miniapp.WithHTTPClient(h), miniapp.WithObserver(o))
			if err != nil {
				return err
			}
			_, err = c.Auth().Code2Session(ctx, "code")
			return err
		}},
		{"mobileapp", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := mobileapp.NewClient(mobileapp.Config{AppID: "app", AppSecret: "secret"}, mobileapp.WithHTTPClient(h), mobileapp.WithObserver(o))
			if err != nil {
				return err
			}
			_, err = c.OAuth().TokenFromCode(ctx, "code")
			return err
		}},
		{"work", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithHTTPClient(h), work.WithObserver(o))
			if err != nil {
				return err
			}
			_, err = c.Contact().Users().Get(ctx, "user")
			return err
		}},
		{"openplatform work", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := openplatform.NewClient(openplatform.Config{AppID: "app", AppSecret: "secret"}, openplatform.WithHTTPClient(h), openplatform.WithObserver(o))
			if err != nil {
				return err
			}
			_, err = c.WorkAuthorizer().PermanentCode(ctx, "suite", "code")
			return err
		}},
		{"openplatform authorized", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := openplatform.NewClient(openplatform.Config{AppID: "app", AppSecret: "secret"}, openplatform.WithHTTPClient(h), openplatform.WithObserver(o))
			if err != nil {
				return err
			}
			child, err := c.AuthorizedOfficial("authorizer", "refresh")
			if err != nil {
				return err
			}
			_, err = child.OAuth().UserInfo(ctx, "oauth-token", "user")
			return err
		}},
		{"healthcard", func(ctx context.Context, h *http.Client, o observability.Observer) error {
			c, err := healthcard.NewClient(healthcard.Config{AppID: "app", AppSecret: "secret", HospitalID: "hospital"}, healthcard.WithHTTPClient(h), healthcard.WithObserver(o))
			if err != nil {
				return err
			}
			return c.Call(ctx, "/test", nil, nil)
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			started, finished := 0, 0
			networkErr := errors.New("network failure")
			observer := observability.ObserverFunc(func(ctx context.Context, e observability.Event) (context.Context, func(observability.Event)) {
				started++
				return context.WithValue(ctx, observerKey{}, true), func(done observability.Event) {
					finished++
					if !errors.Is(done.Err, networkErr) {
						t.Errorf("observer error=%v", done.Err)
					}
				}
			})
			httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Context().Value(observerKey{}) != true {
					t.Error("observer context missing from HTTP request")
				}
				return nil, networkErr
			})}
			err := test.call(context.Background(), httpClient, observer)
			if !errors.Is(err, networkErr) || started != 1 || finished != 1 {
				t.Fatalf("started=%d finished=%d err=%v", started, finished, err)
			}
		})
	}
}
