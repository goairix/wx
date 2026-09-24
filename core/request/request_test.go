package request_test

import (
	"context"
	"testing"

	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
)

var _ request.Caller = (*transport.Client)(nil)

func TestTransportImplementsCaller(t *testing.T) {
	caller := transport.New(nil, "https://example.test", transport.RetryPolicy{})
	if caller == nil {
		t.Fatal("transport caller is nil")
	}
}

type callerFunc func(context.Context, request.Request) error

func (f callerFunc) Do(ctx context.Context, req request.Request) error {
	return f(ctx, req)
}

func TestCallerCanBeImplementedByANarrowAdapter(t *testing.T) {
	var called bool
	var caller request.Caller = callerFunc(func(context.Context, request.Request) error {
		called = true
		return nil
	})
	if err := caller.Do(context.Background(), request.Request{}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("caller adapter was not invoked")
	}
}
