package auth

import (
	"context"
	stderrors "errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/cache"
)

func TestManagerRefreshesOnceForConcurrentCallers(t *testing.T) {
	var calls int32
	provider := ProviderFunc(func(ctx context.Context) (Credential, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(10 * time.Millisecond)
		return Credential{AccessToken: "token", TokenType: "Bearer", ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	manager := NewManager("work", "corp-1", cache.NewMemory(), provider)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := manager.Token(context.Background()); err != nil {
				t.Errorf("Token() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("provider called %d times, want 1", got)
	}
}

func TestManagersSharingCacheAndKeyRefreshOnce(t *testing.T) {
	var calls int32
	provider := ProviderFunc(func(context.Context) (Credential, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(10 * time.Millisecond)
		return Credential{AccessToken: "shared", ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	sharedCache := cache.NewMemory()
	first := NewManager("work", "corp-1", sharedCache, provider)
	second := NewManager("work", "corp-1", sharedCache, provider)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		manager := first
		if i%2 == 1 {
			manager = second
		}
		go func() {
			defer wg.Done()
			if _, err := manager.Token(context.Background()); err != nil {
				t.Errorf("Token() error = %v", err)
			}
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("provider called %d times, want 1", got)
	}
}

func TestMemoryCacheExpiresEntries(t *testing.T) {
	c := cache.NewMemory()
	if err := c.Put(context.Background(), "key", "value", 10*time.Millisecond); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if value, ok, err := c.Get(context.Background(), "key"); err != nil || !ok || value != "value" {
		t.Fatalf("Get() = (%q, %v, %v), want value, true, nil", value, ok, err)
	}
	time.Sleep(20 * time.Millisecond)
	if value, ok, err := c.Get(context.Background(), "key"); err != nil || ok || value != "" {
		t.Fatalf("expired Get() = (%q, %v, %v), want empty, false, nil", value, ok, err)
	}
	if _, ok, err := c.Get(context.Background(), "key"); err != nil || ok {
		t.Fatalf("expired entry remained in cache: ok=%v err=%v", ok, err)
	}
}

func TestManagerRefreshesStaleCredential(t *testing.T) {
	var calls int32
	provider := ProviderFunc(func(context.Context) (Credential, error) {
		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			return Credential{AccessToken: "stale", ExpiresAt: time.Now().Add(30 * time.Second)}, nil
		}
		return Credential{AccessToken: "fresh", ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	manager := NewManager("official", "app-1", cache.NewMemory(), provider)

	first, err := manager.Token(context.Background())
	if err != nil || first.AccessToken != "stale" {
		t.Fatalf("first Token() = (%+v, %v)", first, err)
	}
	second, err := manager.Token(context.Background())
	if err != nil || second.AccessToken != "fresh" {
		t.Fatalf("second Token() = (%+v, %v)", second, err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("provider called %d times, want 2", got)
	}
}

func TestManagerReturnsContextCancellationUnchanged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	manager := NewManager("work", "corp-1", cache.NewMemory(), ProviderFunc(func(context.Context) (Credential, error) {
		t.Fatal("provider called with canceled context")
		return Credential{}, nil
	}))

	_, err := manager.Token(ctx)
	if !stderrors.Is(err, context.Canceled) || err != context.Canceled {
		t.Fatalf("Token() error = %v, want context.Canceled unchanged", err)
	}
}

func TestManagerPropagatesProviderContextCancellation(t *testing.T) {
	manager := NewManager("work", "corp-1", cache.NewMemory(), ProviderFunc(func(context.Context) (Credential, error) {
		return Credential{}, context.DeadlineExceeded
	}))

	_, err := manager.Token(context.Background())
	if err != context.DeadlineExceeded {
		t.Fatalf("Token() error = %v, want context.DeadlineExceeded unchanged", err)
	}
}

func TestMemoryCacheHandlesNonPositiveTTL(t *testing.T) {
	c := cache.NewMemory()
	if err := c.Put(context.Background(), "key", "value", 0); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if value, ok, err := c.Get(context.Background(), "key"); err != nil || ok || value != "" {
		t.Fatalf("Get() after zero TTL = (%q, %v, %v), want empty, false, nil", value, ok, err)
	}
}
