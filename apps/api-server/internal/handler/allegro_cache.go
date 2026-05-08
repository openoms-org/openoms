package handler

import (
	"sync"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
)

// allegroCache is a simple thread-safe in-memory TTL cache for Allegro API responses.
// Used for data that changes infrequently (categories, parameters, delivery methods).
//
// Keys are scoped per-tenant and per-parent (e.g. "cat:{tenantID}:{parentID}"),
// so entry count can grow large with many tenants. A background sweeper evicts
// expired entries periodically; Get also evicts lazily on expired hits to keep
// the map bounded even if the sweeper is paused or the process is idle.
type allegroCache struct {
	mu       sync.RWMutex
	entries  map[string]allegroCacheEntry
	ttl      time.Duration
	stop     chan struct{}
	stopOnce sync.Once
}

type allegroCacheEntry struct {
	data      any
	expiresAt time.Time
}

func newAllegroCache(ttl time.Duration) *allegroCache {
	c := &allegroCache{
		entries: make(map[string]allegroCacheEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	asyncutil.SafeGo(func() { c.sweepLoop() })
	return c
}

// Get returns the cached value and true if it exists and hasn't expired.
// Expired entries are evicted lazily on read.
func (c *allegroCache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		// Re-read under write lock: another goroutine may have refreshed the entry
		// between RLock release and Lock acquire. In that case return the fresh value
		// instead of forcing a redundant upstream fetch.
		fresh, ok := c.entries[key]
		if ok && time.Now().After(fresh.expiresAt) {
			delete(c.entries, key)
			ok = false
		}
		c.mu.Unlock()
		if !ok {
			return nil, false
		}
		return fresh.data, true
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

// Stop halts the background sweeper. Safe to call multiple times.
func (c *allegroCache) Stop() {
	c.stopOnce.Do(func() { close(c.stop) })
}

// sweepLoop periodically evicts expired entries. Runs until Stop is called.
func (c *allegroCache) sweepLoop() {
	interval := max(c.ttl/4, time.Minute)
	interval = min(interval, time.Hour)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-t.C:
			c.sweep()
		}
	}
}

// sweep removes all expired entries. Exposed for tests.
func (c *allegroCache) sweep() {
	now := time.Now()
	c.mu.Lock()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

// len returns the current number of entries. Exposed for tests.
func (c *allegroCache) len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
