package auth

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/cache"
)

func TestInvalidatePreservesReplacementCredential(t *testing.T) {
	ctx := context.Background()
	var calls atomic.Int32
	manager := NewManager("official", "app", nil, ProviderFunc(func(context.Context) (Credential, error) {
		token := "old"
		if calls.Add(1) > 1 {
			token = "new"
		}
		return Credential{AccessToken: token, ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	if _, err := manager.Token(ctx); err != nil {
		t.Fatal(err)
	}
	if err := manager.Invalidate(ctx, "old"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Token(ctx); err != nil {
		t.Fatal(err)
	}
	if err := manager.Invalidate(ctx, "old"); err != nil {
		t.Fatal(err)
	}
	credential, err := manager.Token(ctx)
	if err != nil || credential.AccessToken != "new" || calls.Load() != 2 {
		t.Fatalf("credential=%+v calls=%d err=%v", credential, calls.Load(), err)
	}
}

func TestInvalidateWaitsForInFlightRefresh(t *testing.T) {
	ctx := context.Background()
	store := cache.NewMemory()
	started := make(chan struct{})
	release := make(chan struct{})
	manager := NewManager("official", "app", store, ProviderFunc(func(context.Context) (Credential, error) {
		close(started)
		<-release
		return Credential{AccessToken: "old", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	refreshDone := make(chan error, 1)
	go func() {
		_, err := manager.Token(ctx)
		refreshDone <- err
	}()
	<-started
	invalidateDone := make(chan error, 1)
	go func() { invalidateDone <- manager.Invalidate(ctx, "") }()
	waitForParticipants(t, store, manager.cacheKey, manager.coordinator, 2)
	close(release)
	if err := <-refreshDone; err != nil {
		t.Fatal(err)
	}
	if err := <-invalidateDone; err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Get(ctx, manager.cacheKey); err != nil || ok {
		t.Fatalf("cache exists=%v err=%v", ok, err)
	}
}

func TestInvalidateCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manager := NewManager("official", "app", nil, nil)
	if err := manager.Invalidate(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Invalidate error=%v", err)
	}
}

type blockedDeleteCache struct {
	cache.Cache
	started chan struct{}
	release chan struct{}
}

func (c *blockedDeleteCache) Delete(ctx context.Context, key string) error {
	close(c.started)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.release:
		return c.Cache.Delete(ctx, key)
	}
}

func TestTokenWaiterRechecksAfterInvalidation(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "refresh", true: "cancel"}[canceled], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := &blockedDeleteCache{Cache: cache.NewMemory(), started: make(chan struct{}), release: make(chan struct{})}
			manager := NewManager("official", "app", store, ProviderFunc(func(context.Context) (Credential, error) {
				return Credential{AccessToken: "fresh", ExpiresAt: time.Now().Add(time.Hour)}, nil
			}))
			invalidated := make(chan error, 1)
			go func() { invalidated <- manager.Invalidate(context.Background(), "") }()
			<-store.started
			completed := make(chan error, 1)
			go func() {
				credential, err := manager.Token(ctx)
				if err == nil && credential.AccessToken != "fresh" {
					err = errors.New("Token returned an empty or stale credential")
				}
				completed <- err
			}()
			waitForParticipants(t, store, manager.cacheKey, manager.coordinator, 2)
			if canceled {
				cancel()
			}
			close(store.release)
			if err := <-invalidated; err != nil {
				t.Fatal(err)
			}
			err := <-completed
			if canceled && !errors.Is(err, context.Canceled) || !canceled && err != nil {
				t.Fatalf("canceled=%v err=%v", canceled, err)
			}
		})
	}
}
