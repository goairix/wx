// Package observability defines request instrumentation hooks.
package observability

import (
	"context"
	"time"
)

// Event describes one request lifecycle event.
type Event struct {
	Context    context.Context
	Operation  string
	Platform   string
	StatusCode int
	RequestID  string
	Duration   time.Duration
	Err        error
	// Attempt is one-based; MaxAttempts is the configured attempt limit.
	Attempt     int
	MaxAttempts int
}

// Hook receives request and response events from the transport layer.
type Hook interface {
	OnRequest(Event)
	OnResponse(Event)
}

// HookFunc adapts a function to Hook.
type HookFunc func(Event)

// OnRequest invokes f for a request event.
func (f HookFunc) OnRequest(event Event) {
	if f != nil {
		f(event)
	}
}

// OnResponse invokes f for a response event.
func (f HookFunc) OnResponse(event Event) {
	if f != nil {
		f(event)
	}
}

// Observer starts an observation for each HTTP attempt before request
// construction. The returned context is used for HTTP, logs and Hooks. The
// finish callback, when non-nil, is called exactly once with the final outcome.
// Implementations must support concurrent calls. A nil context preserves the
// parent context; a nil finish callback is allowed.
type Observer interface {
	Start(context.Context, Event) (context.Context, func(Event))
}

// ObserverFunc adapts a function to Observer.
type ObserverFunc func(context.Context, Event) (context.Context, func(Event))

// Start invokes f, or preserves ctx when f is nil.
func (f ObserverFunc) Start(ctx context.Context, event Event) (context.Context, func(Event)) {
	if f == nil {
		return ctx, nil
	}
	return f(ctx, event)
}
