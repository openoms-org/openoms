package service

import (
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// dashboardStatsCacheTTL bounds how stale a cached dashboard snapshot may be.
// The dashboard home aggregates six unbounded full-tenant scans per load, and the
// React Query client refetches on window focus, so a single user generates a burst
// of identical aggregate queries. A short per-pod cache collapses that burst into
// one query per tenant per window; dashboard KPIs tolerate sub-minute staleness.
const dashboardStatsCacheTTL = 30 * time.Second

// statsCache is a thread-safe in-memory TTL cache for per-tenant dashboard stats.
//
// Keys are tenant IDs, so a cached snapshot is only ever served back to the same
// tenant — one tenant can never read another's data. It is a per-pod cache (Redis
// is absent in production): each API replica caches independently, so a tenant may
// observe up to one TTL of staleness and minor cross-replica skew within the
// window, which is acceptable for dashboard KPIs.
//
// Concurrent misses for the same tenant are collapsed with a singleflight group:
// when several requests miss simultaneously (e.g. a React Query window-focus
// refetch storm), exactly one runs the expensive aggregate load and the rest share
// its result, instead of each re-scanning the tenant's orders.
//
// A background sweeper evicts expired entries so the map stays bounded by the
// active-tenant count even under sparse access (a tenant that loads once and never
// returns); get also reports expired entries as misses.
type statsCache struct {
	mu       sync.RWMutex
	entries  map[string]statsCacheEntry
	ttl      time.Duration
	group    singleflight.Group
	stop     chan struct{}
	stopOnce sync.Once
}

type statsCacheEntry struct {
	stats     *model.DashboardStats
	expiresAt time.Time
}

func newStatsCache(ttl time.Duration) *statsCache {
	c := &statsCache{
		entries: make(map[string]statsCacheEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	asyncutil.SafeGo(c.sweepLoop)
	return c
}

// getOrLoad returns the cached snapshot for key, or runs load exactly once across
// concurrent callers for the same key, caches the result, and returns it. The
// returned snapshot must be treated as read-only: it is shared across callers and
// with the cache itself, so mutating it would corrupt other reads.
func (c *statsCache) getOrLoad(key string, load func() (*model.DashboardStats, error)) (*model.DashboardStats, error) {
	if cached, ok := c.get(key); ok {
		return cached, nil
	}

	v, err, _ := c.group.Do(key, func() (any, error) {
		// Re-check under the flight: another concurrent miss may have already
		// populated the cache between our get above and acquiring the flight.
		if cached, ok := c.get(key); ok {
			return cached, nil
		}
		stats, loadErr := load()
		if loadErr != nil {
			return nil, loadErr
		}
		c.set(key, stats)
		return stats, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*model.DashboardStats), nil
}

// get returns the cached snapshot for key if present and unexpired.
func (c *statsCache) get(key string) (*model.DashboardStats, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.stats, true
}

// set stores a snapshot for key with a fresh TTL.
func (c *statsCache) set(key string, stats *model.DashboardStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = statsCacheEntry{
		stats:     stats,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Stop halts the background sweeper. Safe to call multiple times.
func (c *statsCache) Stop() {
	c.stopOnce.Do(func() { close(c.stop) })
}

// sweepLoop periodically evicts expired entries. Runs until Stop is called.
func (c *statsCache) sweepLoop() {
	interval := max(c.ttl, time.Minute)
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
func (c *statsCache) sweep() {
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
func (c *statsCache) len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
