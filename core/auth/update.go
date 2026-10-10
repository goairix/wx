package auth

import (
	"context"
	"fmt"
)

// Update explicitly replaces a credential using provider, even when the cache
// is fresh. It coordinates with refreshes, invalidation and other updates for
// this cache and key in the current process. The provider can persist related
// refresh credentials before the access credential is published. Its context is
// unchanged; waiting callers recheck the cache after either success or failure.
func (m *Manager) Update(ctx context.Context, provider Provider) (Credential, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil || m.cache == nil {
		return Credential{}, fmt.Errorf("auth: credential manager and cache are required")
	}
	for {
		if err := ctx.Err(); err != nil {
			return Credential{}, err
		}
		key, call, leader := beginRefresh(m.cache, m.cacheKey, m.coordinator)
		if !leader {
			_, _, _ = waitRefresh(ctx, call)
			continue
		}
		credential, err := m.fetch(ctx, provider)
		finishRefresh(key, call, Credential{}, nil, true)
		return credential, err
	}
}
