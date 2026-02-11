package middleware

import (
	"sync"
	"time"
)

// TokenBlacklist provides in-memory token revocation.
// Tokens are automatically cleaned up after their TTL expires.
type TokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // token hash -> expiry
}

// NewTokenBlacklist creates a new TokenBlacklist and starts a background
// goroutine that periodically removes expired entries.
func NewTokenBlacklist() *TokenBlacklist {
	bl := &TokenBlacklist{
		tokens: make(map[string]time.Time),
	}
	go bl.cleanup()
	return bl
}

// Revoke adds a token hash to the blacklist. It will be automatically removed
// after expiresAt has passed.
func (bl *TokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	bl.mu.Lock()
	bl.tokens[tokenHash] = expiresAt
	bl.mu.Unlock()
}

// IsRevoked returns true if the given token hash has been revoked and has not
// yet expired from the blacklist.
func (bl *TokenBlacklist) IsRevoked(tokenHash string) bool {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	_, exists := bl.tokens[tokenHash]
	return exists
}

// cleanup runs every 5 minutes and removes expired tokens from the blacklist.
func (bl *TokenBlacklist) cleanup() {
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
