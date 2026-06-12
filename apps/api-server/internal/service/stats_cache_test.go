package service

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func TestStatsCacheHitAndMiss(t *testing.T) {
	c := newStatsCache(time.Minute)
	defer c.Stop()

	_, ok := c.get("tenant-a")
	assert.False(t, ok, "absent key must be a miss")

	want := &model.DashboardStats{Revenue: model.Revenue{Total: 123, Currency: "PLN"}}
	c.set("tenant-a", want)

	got, ok := c.get("tenant-a")
	assert.True(t, ok, "stored key must be a hit")
	assert.Same(t, want, got, "cache returns the stored snapshot")
}

func TestStatsCacheTenantIsolation(t *testing.T) {
	c := newStatsCache(time.Minute)
	defer c.Stop()
	c.set("tenant-a", &model.DashboardStats{Revenue: model.Revenue{Total: 1}})

	// A different tenant key must never read tenant-a's snapshot.
	_, ok := c.get("tenant-b")
	assert.False(t, ok, "one tenant must not read another tenant's cached stats")
}

func TestStatsCacheExpiry(t *testing.T) {
	c := newStatsCache(10 * time.Millisecond)
	defer c.Stop()
	c.set("tenant-a", &model.DashboardStats{})

	_, ok := c.get("tenant-a")
	assert.True(t, ok, "fresh entry is a hit")

	time.Sleep(20 * time.Millisecond)

	_, ok = c.get("tenant-a")
	assert.False(t, ok, "expired entry must be a miss")
}

// TestStatsCacheGetOrLoadCollapsesConcurrentMisses asserts that a burst of
// concurrent misses for the same tenant runs the loader exactly once (the
// singleflight guard), instead of each request re-running the aggregate load.
func TestStatsCacheGetOrLoadCollapsesConcurrentMisses(t *testing.T) {
	c := newStatsCache(time.Minute)
	defer c.Stop()

	var calls int32
	start := make(chan struct{})
	release := make(chan struct{})

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			<-start
			got, err := c.getOrLoad("tenant-a", func() (*model.DashboardStats, error) {
				atomic.AddInt32(&calls, 1)
				<-release // hold the flight open so concurrent callers join it
				return &model.DashboardStats{Revenue: model.Revenue{Total: 42}}, nil
			})
			assert.NoError(t, err)
			assert.NotNil(t, got)
		}()
	}

	close(start)                      // launch all goroutines together
	time.Sleep(30 * time.Millisecond) // let them reach the flight and join it
	close(release)                    // let the single loader complete
	wg.Wait()

	assert.EqualValues(t, 1, atomic.LoadInt32(&calls), "concurrent misses must collapse to one load")

	got, ok := c.get("tenant-a")
	assert.True(t, ok, "the load result must be cached")
	assert.EqualValues(t, 42, got.Revenue.Total)
}

func TestStatsCacheGetOrLoadPropagatesError(t *testing.T) {
	c := newStatsCache(time.Minute)
	defer c.Stop()

	sentinel := assert.AnError
	_, err := c.getOrLoad("tenant-a", func() (*model.DashboardStats, error) {
		return nil, sentinel
	})
	assert.ErrorIs(t, err, sentinel)

	// A failed load must not poison the cache with a nil/empty entry.
	_, ok := c.get("tenant-a")
	assert.False(t, ok, "failed load must not be cached")
}

func TestStatsCacheSweepEvictsExpired(t *testing.T) {
	c := newStatsCache(5 * time.Millisecond)
	defer c.Stop()

	c.set("tenant-a", &model.DashboardStats{})
	assert.Equal(t, 1, c.len())

	time.Sleep(15 * time.Millisecond)
	c.sweep()
	assert.Equal(t, 0, c.len(), "sweep must remove expired entries so the map stays bounded")
}
