package middleware_test

import (
	"sync"
	"testing"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

// --- Dedicated composite blacklist tests ---

func TestCompositeTokenBlacklist_ImplementsTokenBlacklistStore(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	// The composite should satisfy the TokenBlacklistStore interface
	var _ middleware.TokenBlacklistStore = composite
}

func TestCompositeTokenBlacklist_Revoke_PropagatesExpiryToBothStores(t *testing.T) {
	primary := &expiryTrackingStore{}
	fallback := &expiryTrackingStore{}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	expiry := time.Now().Add(2 * time.Hour)
	composite.Revoke("token-1", expiry)

	assert.Equal(t, expiry, primary.lastExpiry, "primary should receive the expiry time")
	assert.Equal(t, expiry, fallback.lastExpiry, "fallback should receive the expiry time")
	assert.Equal(t, "token-1", primary.lastToken)
	assert.Equal(t, "token-1", fallback.lastToken)
}

func TestCompositeTokenBlacklist_IsRevoked_FallbackTrueShortCircuits(t *testing.T) {
	primary := &trackingBlacklistStore{tokens: make(map[string]bool)}
	fallback := &trackingBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	fallback.tokens["cached-token"] = true

	result := composite.IsRevoked("cached-token")
	assert.True(t, result)
	assert.True(t, fallback.queriedIsRevoked, "fallback should have been queried")
	assert.False(t, primary.queriedIsRevoked, "primary should NOT be queried when fallback returns true")
}

func TestCompositeTokenBlacklist_IsRevoked_PrimaryCheckedOnFallbackMiss(t *testing.T) {
	primary := &trackingBlacklistStore{tokens: make(map[string]bool)}
	fallback := &trackingBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	primary.tokens["redis-only"] = true

	result := composite.IsRevoked("redis-only")
	assert.True(t, result)
	assert.True(t, fallback.queriedIsRevoked, "fallback should be checked first")
	assert.True(t, primary.queriedIsRevoked, "primary should be checked when fallback misses")
}

func TestCompositeTokenBlacklist_IsRevoked_NeitherStoreHasToken(t *testing.T) {
	primary := &trackingBlacklistStore{tokens: make(map[string]bool)}
	fallback := &trackingBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	result := composite.IsRevoked("nonexistent")
	assert.False(t, result)
	assert.True(t, fallback.queriedIsRevoked, "fallback should be checked")
	assert.True(t, primary.queriedIsRevoked, "primary should be checked on fallback miss")
}

func TestCompositeTokenBlacklist_Revoke_MultipleTokens(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	tokens := []string{"tok-a", "tok-b", "tok-c"}
	for _, tok := range tokens {
		composite.Revoke(tok, time.Now().Add(1*time.Hour))
	}

	for _, tok := range tokens {
		assert.True(t, primary.tokens[tok], "primary should have %s", tok)
		assert.True(t, fallback.tokens[tok], "fallback should have %s", tok)
		assert.True(t, composite.IsRevoked(tok), "composite should report %s as revoked", tok)
	}
}

func TestCompositeTokenBlacklist_Concurrent_RevokeAndIsRevoked(t *testing.T) {
	primary := middleware.NewMemoryTokenBlacklist()
	fallback := middleware.NewMemoryTokenBlacklist()
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			composite.Revoke("concurrent-token", time.Now().Add(1*time.Hour))
			_ = idx
		}(i)
	}

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = composite.IsRevoked("concurrent-token")
			_ = composite.IsRevoked("other-token")
		}()
	}

	wg.Wait()
	assert.True(t, composite.IsRevoked("concurrent-token"))
	assert.False(t, composite.IsRevoked("other-token"))
}

func TestCompositeTokenBlacklist_UsableWithMemoryStores(t *testing.T) {
	// Integration test: two real MemoryTokenBlacklist instances
	primary := middleware.NewMemoryTokenBlacklist()
	fallback := middleware.NewMemoryTokenBlacklist()
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	composite.Revoke("real-token", time.Now().Add(30*time.Minute))

	assert.True(t, composite.IsRevoked("real-token"))
	assert.True(t, primary.IsRevoked("real-token"))
	assert.True(t, fallback.IsRevoked("real-token"))
	assert.False(t, composite.IsRevoked("unknown"))
}

func TestCompositeTokenBlacklist_ExpiredInFallback_ChecksPrimary(t *testing.T) {
	// If fallback reports token as NOT revoked (e.g. expired), primary should be checked
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	// Token only in primary (simulates fallback having expired entry)
	primary.tokens["maybe-expired"] = true

	assert.True(t, composite.IsRevoked("maybe-expired"))
}

// --- additional test helpers ---

type expiryTrackingStore struct {
	lastToken  string
	lastExpiry time.Time
	tokens     map[string]bool
}

func (e *expiryTrackingStore) Revoke(tokenHash string, expiresAt time.Time) {
	e.lastToken = tokenHash
	e.lastExpiry = expiresAt
}

func (e *expiryTrackingStore) IsRevoked(_ string) bool {
	return false
}

type trackingBlacklistStore struct {
	tokens           map[string]bool
	queriedIsRevoked bool
}

func (t *trackingBlacklistStore) Revoke(tokenHash string, _ time.Time) {
	t.tokens[tokenHash] = true
}

func (t *trackingBlacklistStore) IsRevoked(tokenHash string) bool {
	t.queriedIsRevoked = true
	return t.tokens[tokenHash]
}
