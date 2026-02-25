package middleware_test

import (
	"sync"
	"testing"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

// --- MemoryTokenBlacklist tests ---

func TestMemoryTokenBlacklist_Revoke_IsRevoked(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	bl.Revoke("token-abc", time.Now().Add(1*time.Hour))

	assert.True(t, bl.IsRevoked("token-abc"), "revoked token should be reported as revoked")
}

func TestMemoryTokenBlacklist_NonRevoked_ReturnsFalse(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()

	assert.False(t, bl.IsRevoked("never-revoked"), "non-revoked token should return false")
}

func TestMemoryTokenBlacklist_Expired_ReturnsFalse(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	bl.Revoke("expired-token", time.Now().Add(-10*time.Second))

	assert.False(t, bl.IsRevoked("expired-token"), "expired token should return false")
}

func TestMemoryTokenBlacklist_MultipleTokens_Independent(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	bl.Revoke("token-1", time.Now().Add(1*time.Hour))
	bl.Revoke("token-2", time.Now().Add(1*time.Hour))

	assert.True(t, bl.IsRevoked("token-1"))
	assert.True(t, bl.IsRevoked("token-2"))
	assert.False(t, bl.IsRevoked("token-3"), "unrevoked token should not be affected")
}

func TestMemoryTokenBlacklist_MultipleTokens_MixedExpiry(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()
	bl.Revoke("active-token", time.Now().Add(1*time.Hour))
	bl.Revoke("expired-token", time.Now().Add(-1*time.Second))

	assert.True(t, bl.IsRevoked("active-token"))
	assert.False(t, bl.IsRevoked("expired-token"))
}

func TestMemoryTokenBlacklist_Concurrent_RevokeAndIsRevoked(t *testing.T) {
	bl := middleware.NewMemoryTokenBlacklist()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Half goroutines revoke tokens
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			token := "concurrent-token"
			if idx%2 == 0 {
				token = "concurrent-even"
			}
			bl.Revoke(token, time.Now().Add(1*time.Hour))
		}(i)
	}

	// Half goroutines check revocation
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = bl.IsRevoked("concurrent-token")
			_ = bl.IsRevoked("concurrent-even")
			_ = bl.IsRevoked("nonexistent")
		}()
	}

	wg.Wait()
	// If we get here without -race detecting issues, the test passes
	assert.True(t, bl.IsRevoked("concurrent-token"))
	assert.True(t, bl.IsRevoked("concurrent-even"))
}

// --- TokenBlacklist wrapper tests ---

func TestTokenBlacklist_NewDefault_UsesMemory(t *testing.T) {
	bl := middleware.NewTokenBlacklist()
	bl.Revoke("wrapper-token", time.Now().Add(1*time.Hour))

	assert.True(t, bl.IsRevoked("wrapper-token"))
	assert.False(t, bl.IsRevoked("other-token"))
}

func TestTokenBlacklist_WithCustomStore(t *testing.T) {
	store := &mockBlacklistStore{tokens: make(map[string]bool)}
	bl := middleware.NewTokenBlacklistWithStore(store)

	bl.Revoke("custom-token", time.Now().Add(1*time.Hour))
	assert.True(t, bl.IsRevoked("custom-token"))
	assert.True(t, store.revokeCalled, "should delegate to custom store")
}

// --- CompositeTokenBlacklist tests ---

func TestCompositeBlacklist_Revoke_WritesToBoth(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	composite.Revoke("tok-1", time.Now().Add(1*time.Hour))

	assert.True(t, primary.tokens["tok-1"], "primary should have the token")
	assert.True(t, fallback.tokens["tok-1"], "fallback should have the token")
}

func TestCompositeBlacklist_IsRevoked_FallbackFirst(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	// Token only in fallback
	fallback.tokens["fb-only"] = true

	assert.True(t, composite.IsRevoked("fb-only"), "should find token in fallback")
	assert.False(t, primary.isRevokedCalled, "primary should NOT be checked if fallback returns true")
}

func TestCompositeBlacklist_IsRevoked_FallsThroughToPrimary(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	// Token only in primary
	primary.tokens["primary-only"] = true

	assert.True(t, composite.IsRevoked("primary-only"), "should find token in primary after fallback miss")
	assert.True(t, primary.isRevokedCalled, "primary should be checked when fallback returns false")
}

func TestCompositeBlacklist_IsRevoked_NeitherStore_ReturnsFalse(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	assert.False(t, composite.IsRevoked("ghost-token"), "should return false if not in either store")
}

func TestCompositeBlacklist_IsRevoked_InBothStores_ReturnsTrue(t *testing.T) {
	primary := &mockBlacklistStore{tokens: make(map[string]bool)}
	fallback := &mockBlacklistStore{tokens: make(map[string]bool)}
	composite := middleware.NewCompositeTokenBlacklist(primary, fallback)

	primary.tokens["both"] = true
	fallback.tokens["both"] = true

	assert.True(t, composite.IsRevoked("both"), "should return true when token is in both stores")
}

// --- mock store ---

type mockBlacklistStore struct {
	tokens          map[string]bool
	revokeCalled    bool
	isRevokedCalled bool
}

func (m *mockBlacklistStore) Revoke(tokenHash string, _ time.Time) {
	m.revokeCalled = true
	m.tokens[tokenHash] = true
}

func (m *mockBlacklistStore) IsRevoked(tokenHash string) bool {
	m.isRevokedCalled = true
	return m.tokens[tokenHash]
}
