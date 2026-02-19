package service

import (
	"context"
	"sync"
	"time"
)

type memEntry struct {
	count  int
	expiry time.Time
}

// MemoryLoginLockoutStore implements LoginLockoutStore using an in-memory map.
// Suitable for development and testing; not shared across instances.
type MemoryLoginLockoutStore struct {
	mu      sync.Mutex
	entries map[string]*memEntry
}

// NewMemoryLoginLockoutStore creates a new in-memory lockout store.
func NewMemoryLoginLockoutStore() *MemoryLoginLockoutStore {
	return &MemoryLoginLockoutStore{entries: make(map[string]*memEntry)}
}

// IncrFailures increments the failure counter for the given key, resetting if the window has expired.
func (m *MemoryLoginLockoutStore) IncrFailures(_ context.Context, key string, window time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	if !ok || time.Now().After(e.expiry) {
		e = &memEntry{count: 0, expiry: time.Now().Add(window)}
		m.entries[key] = e
	}
	e.count++
	return e.count, nil
}

// ResetFailures deletes the failure counter for the given key.
func (m *MemoryLoginLockoutStore) ResetFailures(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
	return nil
}

// GetLockoutRemaining returns the remaining lockout duration, or 0 if the key is absent or expired.
func (m *MemoryLoginLockoutStore) GetLockoutRemaining(_ context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	if !ok {
		return 0, nil
	}
	remaining := time.Until(e.expiry)
	if remaining <= 0 {
		delete(m.entries, key)
		return 0, nil
	}
	return remaining, nil
}

// SetLockout creates a lockout entry that expires after the given duration.
func (m *MemoryLoginLockoutStore) SetLockout(_ context.Context, key string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[key] = &memEntry{count: 1, expiry: time.Now().Add(duration)}
	return nil
}
