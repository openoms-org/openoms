package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPlanCache_SetAndGet(t *testing.T) {
	cache := NewPlanCache(5 * time.Minute)

	tenantID := uuid.New()
	cache.Set(tenantID, "plus", nil)

	plan, _, ok := cache.Get(tenantID)
	assert.True(t, ok)
	assert.Equal(t, "plus", plan)
}

func TestPlanCache_Expired(t *testing.T) {
	cache := NewPlanCache(1 * time.Millisecond)

	tenantID := uuid.New()
	cache.Set(tenantID, "pro", nil)

	time.Sleep(5 * time.Millisecond)

	_, _, ok := cache.Get(tenantID)
	assert.False(t, ok) // expired
}

func TestPlanCache_Miss(t *testing.T) {
	cache := NewPlanCache(5 * time.Minute)

	_, _, ok := cache.Get(uuid.New())
	assert.False(t, ok)
}

func TestPlanCache_Invalidate(t *testing.T) {
	cache := NewPlanCache(5 * time.Minute)

	tenantID := uuid.New()
	cache.Set(tenantID, "plus", nil)
	cache.Invalidate(tenantID)

	_, _, ok := cache.Get(tenantID)
	assert.False(t, ok)
}
