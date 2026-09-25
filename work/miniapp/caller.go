package miniapp

import (
	"context"
	"net/url"
)

// Caller executes the authenticated requests needed by this domain.
type Caller interface {
	GetOnce(
		ctx context.Context,
		operation string,
		path string,
		query url.Values,
		result interface{},
	) error
}
