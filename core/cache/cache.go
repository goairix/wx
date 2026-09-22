// Package cache defines the context-aware cache contract shared by platform clients.
package cache

import (
	"context"
	"time"
)

// Cache stores string values with an optional time-to-live.
//
// Implementations should return the caller's context error unchanged when the
// context is canceled or its deadline expires.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Put(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
