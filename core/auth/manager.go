package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/goairix/wx/v2/core/cache"
)

const refreshWindow = time.Minute

// Credential is an access credential and its expiry time.
type Credential struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Manager caches a credential and coordinates refreshes for one cache key.
type Manager struct {
	platform    string
	key         string
	cacheKey    string
	cache       cache.Cache
	provider    Provider
	coordinator *refreshCoordinator
}

// NewManager creates a credential manager for platform and key. A nil cache is
// replaced with an in-memory cache.
func NewManager(platform, key string, c Cache, provider Provider) *Manager {
	if c == nil {
		c = cache.NewMemory()
	}
	return &Manager{
		platform:    platform,
		key:         key,
		cacheKey:    makeCredentialCacheKey(platform, key),
		cache:       c,
		provider:    provider,
		coordinator: &refreshCoordinator{},
	}
}

// makeCredentialCacheKey encodes both components with their lengths so
// platform/key pairs remain distinct even when either component contains the
// separator used by a human-readable cache key.
func makeCredentialCacheKey(platform, key string) string {
	return strconv.Itoa(len(platform)) + ":" + platform + strconv.Itoa(len(key)) + ":" + key
}

// Token returns a cached credential or obtains and caches a fresh one. Calls
// sharing a cache and key observe the same in-flight refresh result.
func (m *Manager) Token(ctx context.Context) (Credential, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Credential{}, err
	}
	if m == nil {
		return Credential{}, fmt.Errorf("auth: nil manager")
	}

	if m.cache == nil {
		return Credential{}, fmt.Errorf("auth: nil cache")
	}
	credential, ok, err := m.cached(ctx)
	if err != nil || ok {
		return credential, err
	}

	for {
		callKey, call, leader := beginRefresh(m.cache, m.cacheKey, m.coordinator)
		if leader {
			credential, err = m.refresh(ctx)
			contextErr := ctx.Err()
			retryWaiters := contextErr != nil && err != nil
			finishRefresh(callKey, call, credential, err, retryWaiters)
			return credential, err
		}

		credential, err, retry := waitRefresh(ctx, call)
		if !retry || ctx.Err() != nil {
			return credential, err
		}
		// The previous refresh used another caller's context. If that caller
		// canceled, an active waiter must be allowed to elect a new leader.
	}
}

func (m *Manager) cached(ctx context.Context) (Credential, bool, error) {
	raw, ok, err := m.cache.Get(ctx, m.cacheKey)
	if err != nil {
		return Credential{}, false, err
	}
	if ok {
		var credential Credential
		if json.Unmarshal([]byte(raw), &credential) == nil && !credential.stale(time.Now()) {
			return credential, true, nil
		}
	}
	return Credential{}, false, nil
}

func (m *Manager) refresh(ctx context.Context) (Credential, error) {
	// A credential may have been stored between the initial cache check and this
	// call becoming the leader.
	credential, ok, err := m.cached(ctx)
	if err != nil || ok {
		return credential, err
	}

	if m.provider == nil {
		return Credential{}, fmt.Errorf("auth: nil provider")
	}
	credential, err = m.provider.Token(ctx)
	if err != nil {
		return Credential{}, err
	}
	if err := ctx.Err(); err != nil {
		return Credential{}, err
	}

	if !credential.ExpiresAt.IsZero() {
		ttl := time.Until(credential.ExpiresAt)
		if ttl > 0 {
			encoded, marshalErr := json.Marshal(credential)
			if marshalErr != nil {
				return Credential{}, marshalErr
			}
			if err := m.cache.Put(ctx, m.cacheKey, string(encoded), ttl); err != nil {
				return Credential{}, err
			}
		}
	}
	return credential, nil
}

func (c Credential) stale(now time.Time) bool {
	return c.ExpiresAt.IsZero() || !c.ExpiresAt.After(now.Add(refreshWindow))
}
