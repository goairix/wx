package security

import (
	"context"
)

// Caller executes authenticated requests needed by this domain.
type Caller interface {
	Post(
		ctx context.Context,
		operation string,
		path string,
		body interface{},
		result interface{},
	) error
}
