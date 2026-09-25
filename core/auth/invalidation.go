package auth

import (
	"context"
	"encoding/json"
	"fmt"
)

// Invalidate removes a rejected credential. When accessToken is nonempty, a
// different cached token is preserved. An empty accessToken explicitly clears
// the credential regardless of its value (for example, after reauthorization).
// It coordinates with refreshes sharing the same cache instance and key in
// this process; distributed atomicity requires an external credential service.
func (m *Manager) Invalidate(ctx context.Context, accessToken string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil || m.cache == nil {
		return fmt.Errorf("auth: credential manager and cache are required")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		key, call, leader := beginRefresh(m.cache, m.cacheKey, m.coordinator)
		if !leader {
			_, _, _ = waitRefresh(ctx, call)
			continue
		}
		err := m.invalidateCached(ctx, accessToken)
		// Token waiters must recheck the cache and elect a refresh leader rather
		// than treating an invalidation's empty result as a fresh credential.
		finishRefresh(key, call, Credential{}, nil, true)
		return err
	}
}

func (m *Manager) invalidateCached(ctx context.Context, accessToken string) error {
	if accessToken != "" {
		raw, ok, err := m.cache.Get(ctx, m.cacheKey)
		if err != nil || !ok {
			return err
		}
		var credential Credential
		if json.Unmarshal([]byte(raw), &credential) == nil && credential.AccessToken != accessToken {
			return nil
		}
	}
	return m.cache.Delete(ctx, m.cacheKey)
}
