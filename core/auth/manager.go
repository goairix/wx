package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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
	platform string
	key      string
	cacheKey string
	cache    cache.Cache
	provider Provider
	mu       sync.Mutex
}

// NewManager creates a credential manager for platform and key. A nil cache is
// replaced with an in-memory cache.
func NewManager(platform, key string, c Cache, provider Provider) *Manager {
	if c == nil {
		c = cache.NewMemory()
	}
	return &Manager{
		platform: platform,
		key:      key,
		cacheKey: platform + ":" + key,
		cache:    c,
		provider: provider,
	}
}

// Token returns a cached credential or obtains and caches a fresh one.
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

	// Each Manager owns one cache key, so this mutex is the per-cache-key
	// single-flight coordinator. Recheck the cache after acquiring it so all
	// concurrent callers observe the first successful refresh.
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Credential{}, err
	}

	if m.cache == nil {
		return Credential{}, fmt.Errorf("auth: nil cache")
	}
	raw, ok, err := m.cache.Get(ctx, m.cacheKey)
	if err != nil {
		return Credential{}, err
	}
	if ok {
		var credential Credential
		if json.Unmarshal([]byte(raw), &credential) == nil && !credential.stale(time.Now()) {
			return credential, nil
		}
	}

	if m.provider == nil {
		return Credential{}, fmt.Errorf("auth: nil provider")
	}
	credential, err := m.provider.Token(ctx)
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
