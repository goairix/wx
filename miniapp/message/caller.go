package message

import (
	"context"
	"net/url"
)

// Caller executes authenticated requests needed by this domain.
type Caller interface {
	Get(
		ctx context.Context,
		operation string,
		path string,
		query url.Values,
		result interface{},
	) error
	Post(
		ctx context.Context,
		operation string,
		path string,
		body interface{},
		result interface{},
	) error
}
