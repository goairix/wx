package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type callerFunc func(context.Context, request.Request) error

func (f callerFunc) Do(ctx context.Context, r request.Request) error { return f(ctx, r) }

type timedValue struct {
	value string
	until time.Time
}
type clockCache struct {
	mu      sync.Mutex
	now     time.Time
	values  map[string]timedValue
	failKey string
	fail    error
	seen    []context.Context
}

func newClockCache() *clockCache {
	return &clockCache{now: time.Now(), values: make(map[string]timedValue)}
}
func (c *clockCache) Get(ctx context.Context, k string) (string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = append(c.seen, ctx)
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	v, ok := c.values[k]
	return v.value, ok && v.until.After(c.now), nil
}
func (c *clockCache) Put(ctx context.Context, k, v string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = append(c.seen, ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if k == c.failKey {
		return c.fail
	}
	c.values[k] = timedValue{v, c.now.Add(ttl)}
	return nil
}
func (c *clockCache) Delete(ctx context.Context, k string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = append(c.seen, ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(c.values, k)
	return nil
}
func (c *clockCache) advance(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.now = c.now.Add(d) }
func tokenFixture() AccessToken {
	return AccessToken{AccessToken: "fresh", RefreshToken: "refresh", ExpiresIn: 7200, OpenID: "user"}
}

// A four-hour refresh TTL or rewriting the same token on refresh breaks this lifecycle.
func TestRefreshLifetimeSurvivesHoursWithoutSliding(t *testing.T) {
	cache := newClockCache()
	var refreshes int
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		switch r.Path {
		case "sns/oauth2/access_token":
			*r.Result.(*AccessToken) = tokenFixture()
		case "sns/oauth2/refresh_token":
			refreshes++
			if got := r.Query.Get("refresh_token"); got != "refresh" {
				return fmt.Errorf("refresh=%q", got)
			}
			*r.Result.(*AccessToken) = tokenFixture()
		case "sns/userinfo":
			*r.Result.(*UserInfo) = UserInfo{BaseUserInfo: BaseUserInfo{OpenID: "user"}}
		default:
			return fmt.Errorf("path=%s", r.Path)
		}
		return nil
	}), Config{AppID: "app"}, cache)
	if _, err := client.TokenFromCode(context.Background(), "code"); err != nil {
		t.Fatal(err)
	}
	cache.advance(5 * time.Hour)
	if _, err := client.UserInfo(context.Background(), "user"); err != nil {
		t.Fatalf("refresh after five hours: %v", err)
	}
	cache.advance(180*24*time.Hour - 5*time.Hour + time.Second)
	if _, err := client.UserInfo(context.Background(), "user"); err == nil {
		t.Fatal("refresh lifetime slid beyond original issuance")
	}
	if refreshes != 1 {
		t.Fatalf("refreshes=%d, want 1", refreshes)
	}
}

// Replay only a recognized access rejection, and never retry rejected fresh credentials.
func TestUserInfoRecoversAccessRejectionOnce(t *testing.T) {
	for _, code := range []int{40001, 40014, 42001, 40003, 40013, 40125, 40029, 40030, 42002, 45011} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cache := corecache.NewMemory()
			var calls, refreshes int
			client := New(callerFunc(func(_ context.Context, r request.Request) error {
				if r.Path == "sns/oauth2/refresh_token" {
					refreshes++
					*r.Result.(*AccessToken) = tokenFixture()
					return nil
				}
				if r.Path != "sns/userinfo" {
					return fmt.Errorf("path=%s", r.Path)
				}
				calls++
				*r.Result.(*UserInfo) = UserInfo{ErrCode: code, ErrMsg: "rejected"}
				return nil
			}), Config{AppID: "app"}, cache)
			_ = cache.Put(context.Background(), client.cacheKey("access", "user"), "old", time.Hour)
			_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
			_, err := client.UserInfo(context.Background(), "user")
			var platformErr *wxerrors.Error
			if !errors.As(err, &platformErr) || platformErr.Code != fmt.Sprint(code) {
				t.Fatalf("err=%v", err)
			}
			want := 1
			if code == 40001 || code == 40014 || code == 42001 {
				want = 2
			}
			if calls != want || refreshes != want-1 {
				t.Fatalf("userinfo=%d refresh=%d, want %d/%d", calls, refreshes, want, want-1)
			}
			if want == 2 {
				if token, ok, _ := cache.Get(context.Background(), client.cacheKey("access", "user")); ok {
					t.Fatalf("terminally rejected token remains cached: %q", token)
				}
			}
		})
	}
}

// Token persistence must never expose usable access after losing the long-lived credential.
func TestTokenDoesNotPublishAccessWhenRefreshPersistenceFails(t *testing.T) {
	cache := newClockCache()
	failure := errors.New("cache unavailable")
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		*r.Result.(*AccessToken) = tokenFixture()
		return nil
	}), Config{AppID: "app"}, cache)
	cache.failKey = client.cacheKey("refresh", "user")
	cache.fail = failure
	if _, err := client.TokenFromCode(context.Background(), "code"); !errors.Is(err, failure) {
		t.Fatalf("err=%v", err)
	}
	if value, ok, _ := cache.Get(context.Background(), client.cacheKey("access", "user")); ok {
		t.Fatalf("published access=%s after refresh failure", value)
	}
}

// Invalid endpoint success data must not create unusable or cross-user cached tokens.
func TestTokenRejectsMalformedSuccess(t *testing.T) {
	for _, field := range []string{"openid", "refresh", "expiry"} {
		t.Run(field, func(t *testing.T) {
			cache := corecache.NewMemory()
			out := tokenFixture()
			switch field {
			case "openid":
				out.OpenID = ""
			case "refresh":
				out.RefreshToken = ""
			case "expiry":
				out.ExpiresIn = 0
			}
			client := New(callerFunc(func(_ context.Context, r request.Request) error { *r.Result.(*AccessToken) = out; return nil }), Config{AppID: "app"}, cache)
			if _, err := client.TokenFromCode(context.Background(), "code"); err == nil {
				t.Fatal("malformed token response accepted")
			}
			if _, ok, _ := cache.Get(context.Background(), client.cacheKey("access", out.OpenID)); ok {
				t.Fatal("malformed credential cached")
			}
		})
	}
}

// initialReadBarrier makes each concurrent caller observe the same expired
// access entry before any of them can start refreshing.
type initialReadBarrier struct {
	corecache.Cache
	key       string
	remaining atomic.Int32
	reached   chan struct{}
	release   chan struct{}
}

func (c *initialReadBarrier) Get(ctx context.Context, key string) (string, bool, error) {
	raw, ok, err := c.Cache.Get(ctx, key)
	if key == c.key {
		left := c.remaining.Add(-1)
		if left >= 0 {
			if left == 0 {
				close(c.reached)
			}
			select {
			case <-c.release:
			case <-ctx.Done():
				return "", false, ctx.Err()
			}
		}
	}
	return raw, ok, err
}

// Concurrent callers across client objects sharing a cache must reuse one refresh.
func TestConcurrentExpiredAccessSharesRefresh(t *testing.T) {
	concurrentExpiredAccessSharesRefresh(t, false)
}

func TestNonComparableCacheStillSharesRefreshWithinClient(t *testing.T) {
	concurrentExpiredAccessSharesRefresh(t, true)
}

type nonComparableCache struct {
	corecache.Cache
	data []byte
}

func concurrentExpiredAccessSharesRefresh(t *testing.T, nonComparable bool) {
	cache := &initialReadBarrier{Cache: corecache.NewMemory(), reached: make(chan struct{}), release: make(chan struct{})}
	cache.remaining.Store(12)
	var refreshes atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	tr := callerFunc(func(_ context.Context, r request.Request) error {
		if r.Path == "sns/oauth2/refresh_token" {
			if refreshes.Add(1) == 1 {
				close(entered)
			}
			<-release
			*r.Result.(*AccessToken) = tokenFixture()
		} else {
			*r.Result.(*UserInfo) = UserInfo{Nickname: r.Query.Get("access_token")}
		}
		return nil
	})
	var backing corecache.Cache = cache
	if nonComparable {
		backing = nonComparableCache{Cache: cache, data: []byte{1}}
	}
	first := New(tr, Config{AppID: "app"}, backing)
	second := first
	if !nonComparable {
		second = New(tr, Config{AppID: "app"}, backing)
	}
	cache.key = first.cacheKey("access", "user")
	_ = cache.Put(context.Background(), first.cacheKey("refresh", "user"), "refresh", time.Hour)
	results := make(chan error, 12)
	var started sync.WaitGroup
	started.Add(12)
	for i := 0; i < 12; i++ {
		client := first
		if i%2 != 0 {
			client = second
		}
		go func() {
			started.Done()
			u, err := client.UserInfo(context.Background(), "user")
			if err == nil && u.Nickname != "fresh" {
				err = fmt.Errorf("nickname=%q", u.Nickname)
			}
			results <- err
		}()
	}
	started.Wait()
	<-cache.reached
	close(cache.release)
	<-entered
	close(release)
	for i := 0; i < 12; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if got := refreshes.Load(); got != 1 {
		t.Fatalf("refresh requests=%d, want 1", got)
	}
}

// Recovery must carry the exact caller context through cache and network calls.
type oauthContextKey struct{}

func TestRecoveryPreservesContextIdentity(t *testing.T) {
	ctx := context.WithValue(context.Background(), oauthContextKey{}, "value")
	cache := newClockCache()
	var calls int
	client := New(callerFunc(func(got context.Context, r request.Request) error {
		if got != ctx {
			t.Errorf("network context replaced")
		}
		if r.Path == "sns/oauth2/refresh_token" {
			*r.Result.(*AccessToken) = tokenFixture()
			return nil
		}
		calls++
		if calls == 1 {
			*r.Result.(*UserInfo) = UserInfo{ErrCode: 42001}
		} else {
			*r.Result.(*UserInfo) = UserInfo{Nickname: "ok"}
		}
		return nil
	}), Config{AppID: "app"}, cache)
	_ = cache.Put(ctx, client.cacheKey("access", "user"), "old", time.Hour)
	_ = cache.Put(ctx, client.cacheKey("refresh", "user"), "refresh", time.Hour)
	user, err := client.UserInfo(ctx, "user")
	if err != nil || user.Nickname != "ok" {
		t.Fatalf("user=%v err=%v", user, err)
	}
	for _, got := range cache.seen {
		if got != ctx {
			t.Fatal("cache context replaced")
		}
	}
}

func TestRefreshRotationRetainsOriginalExpiry(t *testing.T) {
	cache := newClockCache()
	var exchange int
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		out := tokenFixture()
		exchange++
		if exchange > 1 {
			out.RefreshToken = "rotated"
		}
		*r.Result.(*AccessToken) = out
		return nil
	}), Config{AppID: "app"}, cache)
	if _, err := client.TokenFromCode(context.Background(), "code"); err != nil {
		t.Fatal(err)
	}
	// Represent an issuance that has only one hour remaining, without advancing wall time.
	deadline := time.Now().Add(time.Hour)
	if err := cache.Put(context.Background(), client.cacheKey("refresh_expiry", "user"), `{"refresh_token":"refresh","expires_at":"`+deadline.Format(time.RFC3339Nano)+`"}`, time.Hour); err != nil {
		t.Fatal(err)
	}
	_ = cache.Delete(context.Background(), client.cacheKey("access", "user"))
	token, err := client.accessToken(context.Background(), "user")
	if err != nil || token != "fresh" {
		t.Fatalf("token=%q err=%v", token, err)
	}
	cache.advance(2 * time.Hour)
	if _, ok, _ := cache.Get(context.Background(), client.cacheKey("refresh", "user")); ok {
		t.Fatal("rotation extended original refresh lifetime")
	}
}

func TestRefreshRejectsDifferentUser(t *testing.T) {
	cache := corecache.NewMemory()
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		out := tokenFixture()
		out.OpenID = "different"
		*r.Result.(*AccessToken) = out
		return nil
	}), Config{AppID: "app"}, cache)
	_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
	if token, err := client.accessToken(context.Background(), "user"); err == nil {
		t.Fatalf("cross-user response accepted: %q", token)
	}
	if _, ok, _ := cache.Get(context.Background(), client.cacheKey("access", "different")); ok {
		t.Fatal("cached another user's token")
	}
}

func TestLateUserInfoRejectionPreservesRefreshedToken(t *testing.T) {
	cache := corecache.NewMemory()
	entered := make(chan struct{})
	release := make(chan struct{})
	var oldCalls, refreshes atomic.Int32
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		if r.Path == "sns/oauth2/refresh_token" {
			refreshes.Add(1)
			*r.Result.(*AccessToken) = tokenFixture()
			return nil
		}
		if r.Query.Get("access_token") == "old" {
			if oldCalls.Add(1) == 1 {
				close(entered)
				<-release
			}
			*r.Result.(*UserInfo) = UserInfo{ErrCode: 42001}
		} else {
			*r.Result.(*UserInfo) = UserInfo{Nickname: r.Query.Get("access_token")}
		}
		return nil
	}), Config{AppID: "app"}, cache)
	_ = cache.Put(context.Background(), client.cacheKey("access", "user"), "old", time.Hour)
	_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
	result := make(chan error, 1)
	go func() {
		u, err := client.UserInfo(context.Background(), "user")
		if err == nil && u.Nickname != "fresh" {
			err = fmt.Errorf("user=%v", u)
		}
		result <- err
	}()
	<-entered
	user, err := client.UserInfo(context.Background(), "user")
	if err != nil || user.Nickname != "fresh" {
		t.Fatalf("user=%v err=%v", user, err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if got := refreshes.Load(); got != 1 {
		t.Fatalf("late rejection triggered %d refreshes", got)
	}
}

func TestUserInfoReturnsRefreshFailureWithoutReplay(t *testing.T) {
	for _, failure := range []error{context.Canceled, &wxerrors.Error{Code: "40001", Message: "invalid secret"}, &wxerrors.Error{Code: "40030", Message: "invalid refresh"}, errors.New("network unavailable")} {
		t.Run(failure.Error(), func(t *testing.T) {
			cache := corecache.NewMemory()
			var users int
			client := New(callerFunc(func(_ context.Context, r request.Request) error {
				if r.Path == "sns/oauth2/refresh_token" {
					return failure
				}
				users++
				*r.Result.(*UserInfo) = UserInfo{ErrCode: 42001}
				return nil
			}), Config{AppID: "app"}, cache)
			_ = cache.Put(context.Background(), client.cacheKey("access", "user"), "old", time.Hour)
			_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
			if _, err := client.UserInfo(context.Background(), "user"); !errors.Is(err, failure) {
				t.Fatalf("err=%v, want %v", err, failure)
			}
			if users != 1 {
				t.Fatalf("userinfo requests=%d, want 1", users)
			}
		})
	}
}

func TestCanceledRefreshFollowerDoesNotCancelLeader(t *testing.T) {
	cache := corecache.NewMemory()
	entered := make(chan struct{})
	release := make(chan struct{})
	var refreshes atomic.Int32
	client := New(callerFunc(func(ctx context.Context, r request.Request) error {
		if r.Path == "sns/oauth2/refresh_token" {
			if refreshes.Add(1) == 1 {
				close(entered)
			}
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			*r.Result.(*AccessToken) = tokenFixture()
		} else {
			*r.Result.(*UserInfo) = UserInfo{Nickname: "ok"}
		}
		return nil
	}), Config{AppID: "app"}, cache)
	_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
	leader := make(chan error, 1)
	go func() { _, err := client.UserInfo(context.Background(), "user"); leader <- err }()
	<-entered
	followerCtx, cancel := context.WithCancel(context.Background())
	follower := make(chan error, 1)
	go func() { _, err := client.UserInfo(followerCtx, "user"); follower <- err }()
	cancel()
	if err := <-follower; !errors.Is(err, context.Canceled) {
		t.Fatalf("follower err=%v", err)
	}
	close(release)
	if err := <-leader; err != nil {
		t.Fatal(err)
	}
	if got := refreshes.Load(); got != 1 {
		t.Fatalf("refreshes=%d", got)
	}
}

func TestMalformedRefreshDoesNotOverwriteExistingRefresh(t *testing.T) {
	cache := corecache.NewMemory()
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		out := tokenFixture()
		out.RefreshToken = ""
		*r.Result.(*AccessToken) = out
		return nil
	}), Config{AppID: "app"}, cache)
	_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "existing", time.Hour)
	if _, err := client.accessToken(context.Background(), "user"); err == nil {
		t.Fatal("accepted incomplete refresh response")
	}
	if got, ok, _ := cache.Get(context.Background(), client.cacheKey("refresh", "user")); !ok || got != "existing" {
		t.Fatalf("refresh=%q ok=%v", got, ok)
	}
}

// A failed issuance may leave new expiry metadata, but cannot authorize extending
// an older refresh credential using that different issuance's lifetime.
func TestFailedIssuanceDoesNotApplyNewExpiryToOldRefresh(t *testing.T) {
	cache := newClockCache()
	var exchanges int
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		out := tokenFixture()
		exchanges++
		switch exchanges {
		case 2:
			out.RefreshToken = "new-issuance"
		case 3:
			out.RefreshToken = "rotation"
		}
		*r.Result.(*AccessToken) = out
		return nil
	}), Config{AppID: "app"}, cache)
	if _, err := client.TokenFromCode(context.Background(), "first"); err != nil {
		t.Fatal(err)
	}
	cache.failKey = client.cacheKey("refresh", "user")
	cache.fail = errors.New("write failed")
	if _, err := client.TokenFromCode(context.Background(), "second"); err == nil {
		t.Fatal("failed cache write accepted")
	}
	cache.failKey = ""
	_ = cache.Delete(context.Background(), client.cacheKey("access", "user"))
	if _, err := client.accessToken(context.Background(), "user"); err == nil {
		t.Fatal("used another issuance's expiry for old refresh token")
	}
	if value, ok, _ := cache.Get(context.Background(), client.cacheKey("refresh", "user")); !ok || value != "refresh" {
		t.Fatalf("refresh=%q ok=%v", value, ok)
	}
}

func TestUserInfoDoesNotRecoverForeignError(t *testing.T) {
	for _, failure := range []*wxerrors.Error{{Platform: "official", Operation: "mobileapp.oauth.userinfo", Code: "42001"}, {Platform: "mobileapp", Operation: "mobileapp.oauth.token", Code: "40001"}} {
		t.Run(failure.Platform+"/"+failure.Operation, func(t *testing.T) {
			cache := corecache.NewMemory()
			var calls int
			client := New(callerFunc(func(_ context.Context, r request.Request) error { calls++; return failure }), Config{AppID: "app"}, cache)
			_ = cache.Put(context.Background(), client.cacheKey("access", "user"), "old", time.Hour)
			_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
			if _, err := client.UserInfo(context.Background(), "user"); !errors.Is(err, failure) {
				t.Fatalf("err=%v", err)
			}
			if calls != 1 {
				t.Fatalf("requests=%d, want 1", calls)
			}
			if token, ok, _ := cache.Get(context.Background(), client.cacheKey("access", "user")); !ok || token != "old" {
				t.Fatalf("foreign error removed token=%q ok=%v", token, ok)
			}
		})
	}
}

type contextReadSignal struct {
	corecache.Cache
	observed context.Context
	seen     chan struct{}
	once     sync.Once
}

func (c *contextReadSignal) Get(ctx context.Context, key string) (string, bool, error) {
	raw, ok, err := c.Cache.Get(ctx, key)
	if ctx == c.observed {
		c.once.Do(func() { close(c.seen) })
	}
	return raw, ok, err
}

func TestCanceledRefreshLeaderLetsActiveWaiterRefresh(t *testing.T) {
	waiterCtx := context.WithValue(context.Background(), oauthContextKey{}, "waiter")
	cache := &contextReadSignal{Cache: corecache.NewMemory(), observed: waiterCtx, seen: make(chan struct{})}
	entered := make(chan struct{})
	var refreshes atomic.Int32
	client := New(callerFunc(func(ctx context.Context, r request.Request) error {
		if r.Path == "sns/oauth2/refresh_token" {
			if refreshes.Add(1) == 1 {
				close(entered)
				<-ctx.Done()
				return ctx.Err()
			}
			*r.Result.(*AccessToken) = tokenFixture()
		} else {
			*r.Result.(*UserInfo) = UserInfo{Nickname: "ok"}
		}
		return nil
	}), Config{AppID: "app"}, cache)
	_ = cache.Put(context.Background(), client.cacheKey("refresh", "user"), "refresh", time.Hour)
	leaderCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leader := make(chan error, 1)
	go func() { _, err := client.UserInfo(leaderCtx, "user"); leader <- err }()
	<-entered
	waiter := make(chan error, 1)
	go func() {
		u, err := client.UserInfo(waiterCtx, "user")
		if err == nil && u.Nickname != "ok" {
			err = fmt.Errorf("user=%v", u)
		}
		waiter <- err
	}()
	<-cache.seen
	cancel()
	if err := <-leader; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader err=%v", err)
	}
	if err := <-waiter; err != nil {
		t.Fatal(err)
	}
	if got := refreshes.Load(); got != 2 {
		t.Fatalf("refreshes=%d, want canceled leader and active waiter", got)
	}
}

func TestFailedRotationPreservesOriginalRefreshLifetime(t *testing.T) {
	cache := newClockCache()
	var calls int
	client := New(callerFunc(func(_ context.Context, r request.Request) error {
		out := tokenFixture()
		calls++
		if calls > 1 {
			out.RefreshToken = fmt.Sprintf("rotation-%d", calls)
		}
		*r.Result.(*AccessToken) = out
		return nil
	}), Config{AppID: "app"}, cache)
	if _, err := client.TokenFromCode(context.Background(), "code"); err != nil {
		t.Fatal(err)
	}
	_ = cache.Delete(context.Background(), client.cacheKey("access", "user"))
	cache.failKey = client.cacheKey("refresh", "user")
	cache.fail = errors.New("rotation cache write failed")
	if _, err := client.accessToken(context.Background(), "user"); !errors.Is(err, cache.fail) {
		t.Fatalf("err=%v", err)
	}
	if _, ok, _ := cache.Get(context.Background(), client.cacheKey("access", "user")); ok {
		t.Fatal("access published after rotation persistence failed")
	}
	cache.failKey = ""
	if token, err := client.accessToken(context.Background(), "user"); err != nil || token != "fresh" {
		t.Fatalf("old refresh expiry lost after failed rotation: token=%q err=%v", token, err)
	}
}
