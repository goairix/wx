package auth

import (
	"context"
	"reflect"
	"sync"

	"github.com/goairix/wx/v2/core/cache"
)

// cacheLockKey identifies an in-flight refresh without requiring cache
// implementations to be comparable. Pointer-backed caches are isolated by
// instance; value-backed implementations share calls per concrete type.
type cacheLockKey struct {
	typeOf reflect.Type
	ptr    uintptr
	owner  *refreshCoordinator
	key    string
}

type refreshCoordinator struct {
	identity byte
}

type refreshCall struct {
	done         chan struct{}
	credential   Credential
	err          error
	participants int
}

var refreshCalls = struct {
	sync.Mutex
	entries map[cacheLockKey]*refreshCall
}{
	entries: make(map[cacheLockKey]*refreshCall),
}

// beginRefresh joins an existing refresh or creates a call for its leader.
func beginRefresh(
	c cache.Cache,
	key string,
	owner *refreshCoordinator,
) (cacheLockKey, *refreshCall, bool) {
	callKey := makeCacheLockKey(c, key, owner)
	refreshCalls.Lock()
	defer refreshCalls.Unlock()

	if call := refreshCalls.entries[callKey]; call != nil {
		call.participants++
		return callKey, call, false
	}
	call := &refreshCall{
		done:         make(chan struct{}),
		participants: 1,
	}
	refreshCalls.entries[callKey] = call
	return callKey, call, true
}

// finishRefresh publishes one result to every waiter before removing the call.
func finishRefresh(
	callKey cacheLockKey,
	call *refreshCall,
	credential Credential,
	err error,
) {
	refreshCalls.Lock()
	call.credential = credential
	call.err = err
	close(call.done)
	if refreshCalls.entries[callKey] == call {
		delete(refreshCalls.entries, callKey)
	}
	refreshCalls.Unlock()
}

func waitRefresh(ctx context.Context, call *refreshCall) (Credential, error) {
	select {
	case <-call.done:
		return call.credential, call.err
	case <-ctx.Done():
		return Credential{}, ctx.Err()
	}
}

func makeCacheLockKey(
	c cache.Cache,
	key string,
	owner *refreshCoordinator,
) cacheLockKey {
	typeOf := reflect.TypeOf(c)
	result := cacheLockKey{
		typeOf: typeOf,
		key:    key,
	}
	if typeOf == nil {
		return result
	}
	value := reflect.ValueOf(c)
	switch value.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Slice, reflect.UnsafePointer:
		result.ptr = value.Pointer()
		if result.ptr == 0 {
			result.owner = owner
		}
	default:
		// Value caches have no stable runtime identity. Keep coordination private
		// to their Manager rather than merging independent values by type.
		result.owner = owner
	}
	return result
}
