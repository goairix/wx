package auth

import (
	"context"
	"reflect"
	"sync"

	"github.com/goairix/wx/v2/core/cache"
)

// cacheLockKey identifies an in-flight refresh. Reference-backed caches use a
// stable runtime pointer, comparable value caches use value equality, and all
// other cache implementations use a Manager-private owner.
type cacheLockKey struct {
	typeOf reflect.Type
	ptr    uintptr
	value  interface{}
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
	retryWaiters bool
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
	retryWaiters bool,
) {
	refreshCalls.Lock()
	call.credential = credential
	call.err = err
	call.retryWaiters = retryWaiters
	close(call.done)
	if refreshCalls.entries[callKey] == call {
		delete(refreshCalls.entries, callKey)
	}
	refreshCalls.Unlock()
}

func waitRefresh(ctx context.Context, call *refreshCall) (Credential, error, bool) {
	select {
	case <-call.done:
		return call.credential, call.err, call.retryWaiters
	case <-ctx.Done():
		return Credential{}, ctx.Err(), false
	}
}

func makeCacheLockKey(
	c interface{},
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
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.UnsafePointer:
		result.ptr = value.Pointer()
		if result.ptr == 0 {
			result.owner = owner
		}
	default:
		if comparableType(typeOf) && comparableValue(value) {
			result.value = c
		} else {
			// Slices, functions, and non-comparable values have no stable identity.
			result.owner = owner
		}
	}
	return result
}

func comparableValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Interface:
		return value.IsNil() || comparableValue(value.Elem())
	case reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if !comparableValue(value.Index(index)) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if !comparableValue(value.Field(index)) {
				return false
			}
		}
		return true
	default:
		return comparableType(value.Type())
	}
}

// comparableType reports whether values of valueType are comparable.
func comparableType(valueType reflect.Type) bool {
	switch valueType.Kind() {
	case reflect.Array:
		return comparableType(valueType.Elem())
	case reflect.Struct:
		for index := 0; index < valueType.NumField(); index++ {
			if !comparableType(valueType.Field(index).Type) {
				return false
			}
		}
		return true
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Chan,
		reflect.Interface,
		reflect.Ptr,
		reflect.String,
		reflect.UnsafePointer:
		return true
	default:
		return false
	}
}
