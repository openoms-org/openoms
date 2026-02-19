package middleware

import (
	"sync"
	"time"
)

// TokenBlacklistStore defines the interface for token revocation backends.
type TokenBlacklistStore interface {
	Revoke(tokenHash string, expiresAt time.Time)
	IsRevoked(tokenHash string) bool
}

// TokenBlacklist wraps a TokenBlacklistStore. Preserves backward compatibility
// with existing code that uses *TokenBlacklist directly.
type TokenBlacklist struct {
	store TokenBlacklistStore
}

// NewTokenBlacklist creates a TokenBlacklist with in-memory backend (default).
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{store: NewMemoryTokenBlacklist()}
}

// NewTokenBlacklistWithStore creates a TokenBlacklist with a custom backend.
func NewTokenBlacklistWithStore(store TokenBlacklistStore) *TokenBlacklist {
	return &TokenBlacklist{store: store}
}

// Revoke delegates to the underlying store.
func (bl *TokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	bl.store.Revoke(tokenHash, expiresAt)
}

// IsRevoked delegates to the underlying store.
func (bl *TokenBlacklist) IsRevoked(tokenHash string) bool {
	return bl.store.IsRevoked(tokenHash)
}

// --- Memory implementation ---

// MemoryTokenBlacklist is an in-memory token blacklist for development/single-instance use.
type MemoryTokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
}

// NewMemoryTokenBlacklist creates a new in-memory token blacklist with background cleanup.
func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {
	bl := &MemoryTokenBlacklist{tokens: make(map[string]time.Time)}
	go bl.cleanup()
	return bl
}

// Revoke adds a token hash to the in-memory blacklist.
func (bl *MemoryTokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	bl.mu.Lock()
	bl.tokens[tokenHash] = expiresAt
	bl.mu.Unlock()
}

// IsRevoked checks if a token hash is in the in-memory blacklist.
func (bl *MemoryTokenBlacklist) IsRevoked(tokenHash string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	expiry, exists := bl.tokens[tokenHash]
	if !exists {
		return false
	}
	return time.Now().Before(expiry)
}

func (bl *MemoryTokenBlacklist) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		bl.mu.Lock()
		now := time.Now()
		for token, expiry := range bl.tokens {
			if now.After(expiry) {
				delete(bl.tokens, token)
			}
		}
		bl.mu.Unlock()
	}
}
