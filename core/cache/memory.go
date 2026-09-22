package cache

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value     string
	expiresAt time.Time
}

// Memory is an in-process cache safe for concurrent use.
type Memory struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
}

// NewMemory creates an empty in-process cache.
func NewMemory() *Memory {
	return &Memory{entries: make(map[string]memoryEntry)}
}

// Get returns a value when the key exists and has not expired. Expired values
// are removed as part of the read.
func (m *Memory) Get(ctx context.Context, key string) (string, bool, error) {
	if err := contextError(ctx); err != nil {
		return "", false, err
	}
	if m == nil {
		return "", false, nil
	}

	m.mu.RLock()
	entry, ok := m.entries[key]
	m.mu.RUnlock()
	if !ok {
		return "", false, nil
	}
	if entry.expiresAt.IsZero() || time.Now().Before(entry.expiresAt) {
		return entry.value, true, nil
	}

	// Upgrade to a write lock and only remove the entry observed above. A
	// concurrent Put may have replaced it while the read lock was released.
	m.mu.Lock()
	if current, exists := m.entries[key]; exists && current.expiresAt.Equal(entry.expiresAt) && current.value == entry.value {
		delete(m.entries, key)
	}
	m.mu.Unlock()
	return "", false, nil
}

// Put stores value for ttl. A non-positive ttl removes the key immediately.
func (m *Memory) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if ttl <= 0 {
		delete(m.entries, key)
		return nil
	}
	if m.entries == nil {
		m.entries = make(map[string]memoryEntry)
	}
	m.entries[key] = memoryEntry{value: value, expiresAt: time.Now().Add(ttl)}
	return nil
}

// Delete removes key from the cache.
func (m *Memory) Delete(ctx context.Context, key string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if m == nil {
		return nil
	}
	m.mu.Lock()
	delete(m.entries, key)
	m.mu.Unlock()
	return nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
