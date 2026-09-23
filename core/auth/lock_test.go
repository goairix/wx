package auth

import (
	"context"
	"testing"
	"time"
	"unsafe"
)

func TestCacheLockKeyIdentityKinds(t *testing.T) {
	firstOwner := &refreshCoordinator{}
	secondOwner := &refreshCoordinator{}
	assertShared := func(t *testing.T, value interface{}) {
		t.Helper()
		first := makeCacheLockKey(value, "key", firstOwner)
		second := makeCacheLockKey(value, "key", secondOwner)
		if first != second {
			t.Fatalf("same reliable cache instance produced different keys")
		}
	}
	assertPrivate := func(t *testing.T, value interface{}) {
		t.Helper()
		first := makeCacheLockKey(value, "key", firstOwner)
		second := makeCacheLockKey(value, "key", secondOwner)
		if first == second {
			t.Fatalf("cache without reliable identity produced a shared key")
		}
	}

	t.Run("pointer", func(t *testing.T) {
		assertShared(t, &identityPointerCache{})
		var nilCache *identityPointerCache
		assertPrivate(t, nilCache)
	})

	t.Run("map", func(t *testing.T) {
		assertShared(t, mapCache{})
		var nilCache mapCache
		assertPrivate(t, nilCache)
	})

	t.Run("channel", func(t *testing.T) {
		assertShared(t, make(channelCache))
		var nilCache channelCache
		assertPrivate(t, nilCache)
	})

	t.Run("unsafe pointer", func(t *testing.T) {
		value := byte(1)
		assertShared(t, unsafe.Pointer(&value))
		var nilCache unsafe.Pointer
		assertPrivate(t, nilCache)
	})

	t.Run("comparable value", func(t *testing.T) {
		storage := &identityStorage{}
		first := comparableValueCache{storage: storage}
		second := comparableValueCache{storage: storage}
		firstKey := makeCacheLockKey(first, "key", firstOwner)
		secondKey := makeCacheLockKey(second, "key", secondOwner)
		if firstKey != secondKey {
			t.Fatal("equal comparable cache values did not share identity")
		}
	})

	t.Run("slice", func(t *testing.T) {
		assertPrivate(t, sliceCache{1})
		assertPrivate(t, sliceCache{})
		var nilCache sliceCache
		assertPrivate(t, nilCache)
	})

	t.Run("function", func(t *testing.T) {
		assertPrivate(t, functionCache(func() {}))
		var nilCache functionCache
		assertPrivate(t, nilCache)
	})

	t.Run("non-comparable value", func(t *testing.T) {
		assertPrivate(t, nonComparableValueCache{values: []string{"one"}})
	})

	t.Run("interface containing non-comparable value", func(t *testing.T) {
		assertPrivate(t, interfaceValueCache{value: []string{"one"}})
	})
}

type identityPointerCache struct{}

type identityStorage struct {
	value byte
}

type comparableValueCache struct {
	storage *identityStorage
}

type nonComparableValueCache struct {
	values []string
}

type interfaceValueCache struct {
	value interface{}
}

type channelCache chan struct{}

type sliceCache []byte

type functionCache func()

func cacheMiss(ctx context.Context) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	return "", false, nil
}

func cachePut(ctx context.Context) error {
	return ctx.Err()
}

func (c *identityPointerCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c *identityPointerCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return cachePut(ctx)
}

func (c *identityPointerCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}

func (c comparableValueCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c comparableValueCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return cachePut(ctx)
}

func (c comparableValueCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}

func (c nonComparableValueCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c nonComparableValueCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return cachePut(ctx)
}

func (c nonComparableValueCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}

func (c interfaceValueCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c interfaceValueCache) Put(
	ctx context.Context,
	key string,
	value string,
	ttl time.Duration,
) error {
	return cachePut(ctx)
}

func (c interfaceValueCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}

func (c channelCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c channelCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return cachePut(ctx)
}

func (c channelCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}

func (c sliceCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c sliceCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return cachePut(ctx)
}

func (c sliceCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}

func (c functionCache) Get(ctx context.Context, key string) (string, bool, error) {
	return cacheMiss(ctx)
}

func (c functionCache) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return cachePut(ctx)
}

func (c functionCache) Delete(ctx context.Context, key string) error {
	return cachePut(ctx)
}
