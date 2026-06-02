package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlanStatus represents cached plan state for a tenant.
type PlanStatus struct {
	Plan     string
	Settings json.RawMessage
	CachedAt time.Time
}

// PlanCache provides an in-memory cache for tenant plan lookups.
// TTL-based: entries expire after the configured duration.
// Expired entries are lazily evicted on reads and periodically on writes.
type PlanCache struct {
	mu         sync.RWMutex
	items      map[uuid.UUID]PlanStatus
	ttl        time.Duration
	writeCount int
}

const evictionInterval = 100 // clean up every N writes

// NewPlanCache creates a plan cache with the given TTL.
func NewPlanCache(ttl time.Duration) *PlanCache {
	return &PlanCache{
		items: make(map[uuid.UUID]PlanStatus),
		ttl:   ttl,
	}
}

// Get returns the cached plan for a tenant. Returns ok=false if missing or expired.
// Expired entries are lazily deleted.
func (c *PlanCache) Get(tenantID uuid.UUID) (string, json.RawMessage, bool) {
	c.mu.RLock()
	entry, exists := c.items[tenantID]
	c.mu.RUnlock()

	if !exists {
		return "", nil, false
	}
	if time.Since(entry.CachedAt) > c.ttl {
		// Lazy eviction of expired entry
		c.mu.Lock()
		if e, ok := c.items[tenantID]; ok && time.Since(e.CachedAt) > c.ttl {
			delete(c.items, tenantID)
		}
		c.mu.Unlock()
		return "", nil, false
	}
	return entry.Plan, entry.Settings, true
}

// Set stores a plan entry in the cache and periodically evicts expired entries.
func (c *PlanCache) Set(tenantID uuid.UUID, plan string, settings json.RawMessage) {
	c.mu.Lock()
	c.items[tenantID] = PlanStatus{Plan: plan, Settings: settings, CachedAt: time.Now()}
	c.writeCount++
	shouldEvict := c.writeCount >= evictionInterval
	if shouldEvict {
		c.writeCount = 0
	}
	c.mu.Unlock()

	if shouldEvict {
		c.evictExpired()
	}
}

// Invalidate removes a tenant's cached plan entry.
func (c *PlanCache) Invalidate(tenantID uuid.UUID) {
	c.mu.Lock()
	delete(c.items, tenantID)
	c.mu.Unlock()
}

// evictExpired removes all expired entries from the cache.
func (c *PlanCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for id, entry := range c.items {
		if now.Sub(entry.CachedAt) > c.ttl {
			delete(c.items, id)
		}
	}
}

// LoadFromDB fetches tenant plan via SECURITY DEFINER function (no RLS context needed).
func (c *PlanCache) LoadFromDB(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) (string, json.RawMessage, error) {
	var plan string
	var settings json.RawMessage
	err := pool.QueryRow(ctx, `SELECT plan, settings FROM get_tenant_plan($1)`, tenantID).Scan(&plan, &settings)
	if err != nil {
		return "", nil, err
	}
	c.Set(tenantID, plan, settings)
	return plan, settings, nil
}

// GetOrLoad returns cached plan, or fetches from DB if cache miss.
func (c *PlanCache) GetOrLoad(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) (string, json.RawMessage, error) {
	plan, settings, ok := c.Get(tenantID)
	if ok {
		return plan, settings, nil
	}
	return c.LoadFromDB(ctx, pool, tenantID)
}
