package middleware

import "time"

// CompositeTokenBlacklist writes to both stores and reads from either.
// This ensures revoked tokens stay blocked even when the primary store (Redis) is down,
// because the in-memory fallback always catches recently-revoked tokens.
type CompositeTokenBlacklist struct {
	primary  TokenBlacklistStore // Redis (authoritative, shared across instances)
	fallback TokenBlacklistStore // In-memory (always available, instance-local)
}

// NewCompositeTokenBlacklist creates a composite store that writes to both backends.
func NewCompositeTokenBlacklist(primary, fallback TokenBlacklistStore) *CompositeTokenBlacklist {
	return &CompositeTokenBlacklist{primary: primary, fallback: fallback}
}

// Revoke adds a token hash to both stores.
func (c *CompositeTokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	c.primary.Revoke(tokenHash, expiresAt)
	c.fallback.Revoke(tokenHash, expiresAt)
}

// IsRevoked checks the in-memory fallback first (cheap, never fails),
// then the primary store. Returns true if either reports the token as revoked.
func (c *CompositeTokenBlacklist) IsRevoked(tokenHash string) bool {
	if c.fallback.IsRevoked(tokenHash) {
		return true
	}
	return c.primary.IsRevoked(tokenHash)
}
