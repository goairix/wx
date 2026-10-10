package wechat

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
)

func rejectedToken(platform, operation, code string) error {
	return &wxerrors.Error{Platform: platform, Operation: operation, Code: code, HTTPStatus: 200, Message: "rejected"}
}

func TestTokenRecoveryPreservesContextAndFailure(t *testing.T) {
	for _, stage := range []string{"initial token", "refresh", "invalidate", "canceled", "non-replayable", "second rejection", "foreign operation"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			failure := errors.New("credential storage failed")
			calls, sends := 0, 0
			store := &recoveryCache{Cache: cache.NewMemory(), ctx: ctx}
			manager := auth.NewManager("official", "app", store, auth.ProviderFunc(func(got context.Context) (auth.Credential, error) {
				if got != ctx {
					t.Error("provider context was replaced")
				}
				calls++
				if stage == "initial token" || stage == "refresh" && calls == 2 {
					return auth.Credential{}, failure
				}
				token := "old"
				if calls > 1 {
					token = "new"
				}
				return auth.Credential{AccessToken: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
			}))
			if stage == "invalidate" {
				store.deleteErr = failure
			}
			err := DoWithToken(ctx, manager, "official", "business", func(credential auth.Credential) (bool, error) {
				sends++
				if stage == "canceled" {
					cancel()
				}
				operation := "business"
				if stage == "foreign operation" {
					operation = "different operation"
				}
				return stage != "non-replayable", rejectedToken("official", operation, "42001")
			})
			switch stage {
			case "initial token", "refresh", "invalidate":
				if !errors.Is(err, failure) {
					t.Fatalf("error = %v, want credential failure", err)
				}
			case "canceled":
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want canceled", err)
				}
			default:
				var pe *wxerrors.Error
				if !errors.As(err, &pe) || pe.Code != "42001" {
					t.Fatalf("error = %v", err)
				}
			}
			wantCalls, wantSends := 1, 1
			if stage == "initial token" {
				wantSends = 0
			}
			if stage == "refresh" {
				wantCalls = 2
			}
			if stage == "second rejection" {
				wantCalls, wantSends = 2, 2
			}
			if calls != wantCalls || sends != wantSends {
				t.Errorf("calls=%d sends=%d want %d/%d", calls, sends, wantCalls, wantSends)
			}
			if store.contextLost {
				t.Error("cache context was replaced")
			}
			if stage == "second rejection" {
				if _, err := manager.Token(ctx); err != nil {
					t.Fatal(err)
				}
				if calls != 3 {
					t.Error("second rejected token remained cached")
				}
			}
		})
	}
}

type recoveryCache struct {
	cache.Cache
	ctx         context.Context
	contextLost bool
	deleteErr   error
}

func (c *recoveryCache) Delete(ctx context.Context, key string) error {
	c.contextLost = c.contextLost || ctx != c.ctx
	if c.deleteErr != nil {
		return c.deleteErr
	}
	return c.Cache.Delete(ctx, key)
}

func TestTokenRecoveryPreservesTokenRefreshedByAnotherRequest(t *testing.T) {
	var fetches atomic.Int32
	store := cache.NewMemory()
	provider := auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		token := "old"
		if fetches.Add(1) > 1 {
			token = "new"
		}
		return auth.Credential{AccessToken: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	first := auth.NewManager("official", "app", store, provider)
	late := auth.NewManager("official", "app", store, provider)
	oldUsed := make(chan struct{})
	allowLateError := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- DoWithToken(context.Background(), late, "official", "business", func(c auth.Credential) (bool, error) {
			if c.AccessToken == "old" {
				close(oldUsed)
				<-allowLateError
				return true, rejectedToken("official", "business", "42001")
			}
			if c.AccessToken != "new" {
				return true, errors.New("unexpected token")
			}
			return true, nil
		})
	}()
	<-oldUsed
	err := DoWithToken(context.Background(), first, "official", "business", func(c auth.Credential) (bool, error) {
		if c.AccessToken == "old" {
			return true, rejectedToken("official", "business", "42001")
		}
		return true, nil
	})
	close(allowLateError)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if fetches.Load() != 2 {
		t.Errorf("fetches = %d, want 2", fetches.Load())
	}
}
