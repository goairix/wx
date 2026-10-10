package agent

import (
	"context"
	"net/url"
)

// Caller executes the authenticated requests needed by application management.
type Caller interface {
	Get(context.Context, string, string, url.Values, interface{}) error
	GetOnce(context.Context, string, string, url.Values, interface{}) error
	Post(context.Context, string, string, interface{}, interface{}) error
	Do(context.Context, string, string, string, url.Values, interface{}, interface{}) error
}
