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
type PlanCache struct {
	mu    sync.RWMutex
	items map[uuid.UUID]PlanStatus
	ttl   time.Duration
}

// NewPlanCache creates a plan cache with the given TTL.
func NewPlanCache(ttl time.Duration) *PlanCache {
	return &PlanCache{
		items: make(map[uuid.UUID]PlanStatus),
		ttl:   ttl,
	}
}

// Get returns the cached plan for a tenant. Returns ok=false if missing or expired.
func (c *PlanCache) Get(tenantID uuid.UUID) (string, json.RawMessage, bool) {
	c.mu.RLock()
	entry, exists := c.items[tenantID]
	c.mu.RUnlock()

	if !exists || time.Since(entry.CachedAt) > c.ttl {
		return "", nil, false
	}
	return entry.Plan, entry.Settings, true
}

// Set stores a plan entry in the cache.
func (c *PlanCache) Set(tenantID uuid.UUID, plan string, settings json.RawMessage) {
	c.mu.Lock()
	c.items[tenantID] = PlanStatus{Plan: plan, Settings: settings, CachedAt: time.Now()}
	c.mu.Unlock()
}

// Invalidate removes a tenant's cached plan entry.
func (c *PlanCache) Invalidate(tenantID uuid.UUID) {
	c.mu.Lock()
	delete(c.items, tenantID)
	c.mu.Unlock()
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

// IsPlanActive returns true if the plan allows normal API access.
func IsPlanActive(plan string) bool {
	switch plan {
	case "free", "standard", "plus", "pro":
		return true
	default:
		return false
	}
}
