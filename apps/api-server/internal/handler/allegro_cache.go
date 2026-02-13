package handler

import (
	"sync"
	"time"
)

// allegroCache is a simple thread-safe in-memory TTL cache for Allegro API responses.
// Used for data that changes infrequently (categories, parameters, delivery methods).
type allegroCache struct {
	mu      sync.RWMutex
	entries map[string]allegroCacheEntry
	ttl     time.Duration
}

type allegroCacheEntry struct {
	data      any
	expiresAt time.Time
}

func newAllegroCache(ttl time.Duration) *allegroCache {
	return &allegroCache{
		entries: make(map[string]allegroCacheEntry),
		ttl:     ttl,
	}
}

// Get returns the cached value and true if it exists and hasn't expired.
func (c *allegroCache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

// Set stores a value in the cache.
func (c *allegroCache) Set(key string, data any) {
	c.mu.Lock()
	c.entries[key] = allegroCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}
