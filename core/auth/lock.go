package auth

import (
	"reflect"
	"sync"

	"github.com/goairix/wx/v2/core/cache"
)

// cacheLockKey identifies a refresh lock without requiring cache
// implementations to be comparable. Pointer-backed caches are isolated by
// instance; value-backed implementations share a lock per concrete type,
// which is safe but may serialize unrelated values.
type cacheLockKey struct {
	typeOf reflect.Type
	ptr    uintptr
	key    string
}

type refreshLockEntry struct {
	mu   sync.Mutex
	refs int
}

var refreshLocks = struct {
	sync.Mutex
	entries map[cacheLockKey]*refreshLockEntry
}{entries: make(map[cacheLockKey]*refreshLockEntry)}

// acquireRefreshLock returns a release function for the cache/key pair. The
// registry reference count removes idle entries, preventing a long-lived
// process from leaking one mutex for every credential key it has seen.
func acquireRefreshLock(c cache.Cache, key string) func() {
	lockKey := makeCacheLockKey(c, key)
	refreshLocks.Lock()
	entry := refreshLocks.entries[lockKey]
	if entry == nil {
		entry = &refreshLockEntry{}
		refreshLocks.entries[lockKey] = entry
	}
	entry.refs++
	refreshLocks.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		refreshLocks.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(refreshLocks.entries, lockKey)
		}
		refreshLocks.Unlock()
	}
}

func makeCacheLockKey(c cache.Cache, key string) cacheLockKey {
	typeOf := reflect.TypeOf(c)
	result := cacheLockKey{typeOf: typeOf, key: key}
	if typeOf == nil {
		return result
	}
	value := reflect.ValueOf(c)
	if value.Kind() == reflect.Ptr {
		result.ptr = value.Pointer()
	}
	return result
}
