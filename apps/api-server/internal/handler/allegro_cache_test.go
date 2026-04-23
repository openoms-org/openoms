package handler

import (
	"testing"
	"time"
)

func TestAllegroCache_GetSet(t *testing.T) {
	c := newAllegroCache(time.Hour)
	defer c.Stop()

	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss on unset key")
	}

	c.Set("k", "v")
	got, ok := c.Get("k")
	if !ok || got != "v" {
		t.Fatalf("expected hit with value v, got %v ok=%v", got, ok)
	}
}

func TestAllegroCache_GetEvictsExpiredLazily(t *testing.T) {
	c := newAllegroCache(time.Millisecond)
	defer c.Stop()

	c.Set("k", "v")
	if c.len() != 1 {
		t.Fatalf("expected 1 entry after Set, got %d", c.len())
	}

	time.Sleep(5 * time.Millisecond)

	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss on expired entry")
	}
	// Lazy eviction must remove the expired entry from the map.
	if c.len() != 0 {
		t.Fatalf("expected 0 entries after expired Get, got %d", c.len())
	}
}

func TestAllegroCache_SweepRemovesExpired(t *testing.T) {
	c := newAllegroCache(time.Millisecond)
	defer c.Stop()

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	if c.len() != 3 {
		t.Fatalf("expected 3 entries, got %d", c.len())
	}

	time.Sleep(5 * time.Millisecond)
	c.sweep()

	if c.len() != 0 {
		t.Fatalf("sweep should have removed all expired entries, got %d", c.len())
	}
}

func TestAllegroCache_SweepKeepsFresh(t *testing.T) {
	c := newAllegroCache(time.Hour)
	defer c.Stop()

	c.Set("fresh", "value")
	c.sweep()

	if _, ok := c.Get("fresh"); !ok {
		t.Fatal("sweep removed a non-expired entry")
	}
}

func TestAllegroCache_GetReturnsRefreshedEntryAfterRace(t *testing.T) {
	// Simulate the race: the first RLock-read sees an expired entry, but before
	// the write-lock recheck another goroutine refreshes it via Set. Get must
	// return the fresh value instead of forcing a redundant upstream fetch.
	c := newAllegroCache(time.Hour)
	defer c.Stop()

	// Seed an already-expired entry directly so the first read path hits the
	// expired branch.
	c.mu.Lock()
	c.entries["k"] = allegroCacheEntry{data: "stale", expiresAt: time.Now().Add(-time.Second)}
	c.mu.Unlock()

	// Refresh the entry — this models the concurrent Set that completes before
	// the Get path re-acquires the write lock.
	c.Set("k", "fresh")

	got, ok := c.Get("k")
	if !ok || got != "fresh" {
		t.Fatalf("expected refreshed value 'fresh', got %v ok=%v", got, ok)
	}
}

func TestAllegroCache_StopIsIdempotent(_ *testing.T) {
	c := newAllegroCache(time.Hour)
	c.Stop()
	c.Stop() // must not panic on second call
}
