// Package auth coordinates credential caching and refresh lifecycles.
package auth

import (
	"context"

	"github.com/goairix/wx/v2/core/cache"
)

// Cache is the cache contract used by Manager.
type Cache = cache.Cache

// Provider obtains a credential from a platform API.
type Provider interface {
	Token(context.Context) (Credential, error)
}

// ProviderFunc adapts a function to Provider.
type ProviderFunc func(context.Context) (Credential, error)

// Token invokes f.
func (f ProviderFunc) Token(ctx context.Context) (Credential, error) {
	return f(ctx)
}
