// Package contracts defines the narrow request interface used by healthcard domains.
package contracts

import "context"

// Caller executes requests through the root healthcard client.
type Caller interface {
	Call(
		ctx context.Context,
		path string,
		req interface{},
		result interface{},
	) error
	CallWithRelated(
		ctx context.Context,
		path string,
		req interface{},
		result interface{},
		relateOpenID string,
	) error
}
