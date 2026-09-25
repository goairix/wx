package accountid

import (
	"context"
)

// Caller executes the authenticated requests needed by this domain.
type Caller interface {
	Post(
		ctx context.Context,
		operation string,
		path string,
		body interface{},
		result interface{},
	) error
}
