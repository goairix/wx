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

func TestManagerSharesProviderFailureWithConcurrentCallers(t *testing.T) {
	const callers = 20
	providerError := stderrors.New("provider unavailable")
	release := make(chan struct{})
	var calls int32
	sharedCache := cache.NewMemory()
	manager := NewManager("work", "corp-failure", sharedCache, ProviderFunc(func(context.Context) (Credential, error) {
		atomic.AddInt32(&calls, 1)
		<-release
		return Credential{}, providerError
	}))

	start := make(chan struct{})
	errorsByCaller := make(chan error, callers)
	var group sync.WaitGroup
	for i := 0; i < callers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := manager.Token(context.Background())
			errorsByCaller <- err
		}()
	}
	close(start)
	waitForParticipants(t, sharedCache, manager.cacheKey, manager.coordinator, callers)
	close(release)
	group.Wait()
	close(errorsByCaller)

	for err := range errorsByCaller {
		if err != providerError {
			t.Fatalf("Token() error = %v, want shared provider error", err)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("provider called %d times, want 1", got)
	}
}

func TestManagerWaiterCanCancelDuringRefresh(t *testing.T) {
	providerStarted := make(chan struct{})
	releaseProvider := make(chan struct{})
	sharedCache := cache.NewMemory()
	manager := NewManager("work", "corp-cancel", sharedCache, ProviderFunc(func(context.Context) (Credential, error) {
		close(providerStarted)
		<-releaseProvider
		return Credential{
			AccessToken: "token",
			ExpiresAt:   time.Now().Add(time.Hour),
		}, nil
	}))

	leaderDone := make(chan error, 1)
	go func() {
		_, err := manager.Token(context.Background())
		leaderDone <- err
	}()
	<-providerStarted

	waiterContext, cancel := context.WithCancel(context.Background())
	waiterDone := make(chan error, 1)
	go func() {
		_, err := manager.Token(waiterContext)
		waiterDone <- err
	}()
	waitForParticipants(t, sharedCache, manager.cacheKey, manager.coordinator, 2)
	cancel()
	select {
	case err := <-waiterDone:
		if err != context.Canceled {
			t.Fatalf("waiter error = %v, want context.Canceled", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("canceled waiter remained blocked on provider")
	}

	close(releaseProvider)
	if err := <-leaderDone; err != nil {
		t.Fatalf("leader Token() error = %v", err)
	}
}

func waitForParticipants(
	t *testing.T,
	c cache.Cache,
	key string,
	owner *refreshCoordinator,
	want int,
) {
	t.Helper()
	callKey := makeCacheLockKey(c, key, owner)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		refreshCalls.Lock()
		call := refreshCalls.entries[callKey]
		got := 0
		if call != nil {
			got = call.participants
		}
		refreshCalls.Unlock()
		if got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("in-flight refresh did not reach %d participants", want)
}

func TestManagersWithAmbiguousComponentsDoNotShareCacheEntry(t *testing.T) {
	var calls int32
	provider := ProviderFunc(func(context.Context) (Credential, error) {
		call := atomic.AddInt32(&calls, 1)
		return Credential{AccessToken: "token-" + string(rune('0'+call)), ExpiresAt: time.Now().Add(time.Hour)}, nil
	})
	sharedCache := cache.NewMemory()
	first := NewManager("a", "b:c", sharedCache, provider)
	second := NewManager("a:b", "c", sharedCache, provider)

	firstCredential, err := first.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token() error = %v", err)
	}
	secondCredential, err := second.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token() error = %v", err)
	}
	if firstCredential.AccessToken == secondCredential.AccessToken {
		t.Fatalf("ambiguous platform/key pairs shared credential %q", firstCredential.AccessToken)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("provider called %d times, want 2", got)
	}
}

func TestManagersWithDifferentMapCachesDoNotShareRefresh(t *testing.T) {
	firstCache := mapCache{}
	secondCache := mapCache{}
	assertIndependentRefreshes(t, firstCache, secondCache)
}

func TestManagersWithValueCachesDoNotShareRefresh(t *testing.T) {
	firstCache := valueCache{id: "first"}
	secondCache := valueCache{id: "second"}
	assertIndependentRefreshes(t, firstCache, secondCache)
}

func TestManagersSharingMapCacheAndKeyShareRefresh(t *testing.T) {
	sharedCache := mapCache{}
	release := make(chan struct{})
	var calls int32
	provider := ProviderFunc(func(context.Context) (Credential, error) {
		atomic.AddInt32(&calls, 1)
		<-release
		return Credential{
			AccessToken: "shared-map-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		}, nil
	})
	first := NewManager("work", "map-corp", sharedCache, provider)
	second := NewManager("work", "map-corp", sharedCache, provider)
	results := make(chan Credential, 2)
	for _, manager := range []*Manager{first, second} {
		manager := manager
		go func() {
			credential, _ := manager.Token(context.Background())
			results <- credential
		}()
	}
	waitForParticipants(t, sharedCache, first.cacheKey, first.coordinator, 2)
	close(release)
	for i := 0; i < 2; i++ {
		credential := <-results
		if credential.AccessToken != "shared-map-token" {
			t.Fatalf("credential = %#v", credential)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("provider called %d times, want 1", got)
	}
}

func assertIndependentRefreshes(t *testing.T, firstCache, secondCache cache.Cache) {
	t.Helper()
	started := make(chan string, 2)
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	provider := func(token string) Provider {
		return ProviderFunc(func(context.Context) (Credential, error) {
			started <- token
			<-release
			return Credential{
				AccessToken: token,
				ExpiresAt:   time.Now().Add(time.Hour),
			}, nil
		})
	}
	first := NewManager("work", "same-key", firstCache, provider("first-token"))
	second := NewManager("work", "same-key", secondCache, provider("second-token"))
	results := make(chan Credential, 2)
	for _, manager := range []*Manager{first, second} {
		manager := manager
		go func() {
			credential, _ := manager.Token(context.Background())
			results <- credential
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("independent cache refresh was merged with another cache")
		}
	}
	close(release)
	released = true
	tokens := make(map[string]bool)
	for i := 0; i < 2; i++ {
		tokens[(<-results).AccessToken] = true
	}
	if !tokens["first-token"] || !tokens["second-token"] {
		t.Fatalf("independent credentials = %#v", tokens)
	}
}

var mapCacheMutex sync.Mutex

type mapCache map[string]string

func (c mapCache) Get(ctx context.Context, key string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	mapCacheMutex.Lock()
	defer mapCacheMutex.Unlock()
	value, ok := c[key]
	return value, ok, nil
}

func (c mapCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mapCacheMutex.Lock()
	c[key] = value
	mapCacheMutex.Unlock()
	return nil
}

func (c mapCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mapCacheMutex.Lock()
	delete(c, key)
	mapCacheMutex.Unlock()
	return nil
}

type valueCache struct {
	id string
}

func (c valueCache) Get(ctx context.Context, key string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	return "", false, nil
}

func (c valueCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return ctx.Err()
}

func (c valueCache) Delete(ctx context.Context, key string) error {
	return ctx.Err()
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
