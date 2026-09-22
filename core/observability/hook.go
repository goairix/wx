// Package observability defines request instrumentation hooks.
package observability

import "time"

// Event describes one request lifecycle event.
type Event struct {
	Operation  string
	Platform   string
	StatusCode int
	RequestID  string
	Duration   time.Duration
	Err        error
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
