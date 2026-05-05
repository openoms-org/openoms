package service

import (
	"context"
	"errors"
	"time"
)

var errRefreshTokenFamilyNotFound = errors.New("refresh token family not found")

// RefreshTokenFamily tracks a chain of rotated refresh tokens for one login session.
type RefreshTokenFamily struct {
	FamilyID         string    `json:"family_id"`
	UserID           string    `json:"user_id"`
	TenantID         string    `json:"tenant_id"`
	CurrentTokenHash string    `json:"current_token_hash"`
	CreatedAt        time.Time `json:"created_at"`
}

// RefreshTokenEntry represents a single refresh token in a family.
type RefreshTokenEntry struct {
	FamilyID string `json:"family_id"`
	Used     bool   `json:"used"`
}

// RefreshTokenStore defines the interface for refresh token rotation tracking.
type RefreshTokenStore interface {
	// StoreFamily persists a token family with the given TTL.
	StoreFamily(ctx context.Context, family *RefreshTokenFamily, ttl time.Duration) error
	// GetFamily retrieves a token family by its ID. Returns nil if not found.
	GetFamily(ctx context.Context, familyID string) (*RefreshTokenFamily, error)
	// UpdateFamily overwrites an existing token family with a new TTL.
	// It must not create a missing family.
	UpdateFamily(ctx context.Context, family *RefreshTokenFamily, ttl time.Duration) error
	// DeleteFamily removes a token family by its ID.
	DeleteFamily(ctx context.Context, familyID string) error
	// StoreToken persists a refresh token entry keyed by token hash with the given TTL.
	StoreToken(ctx context.Context, tokenHash string, entry *RefreshTokenEntry, ttl time.Duration) error
	// GetToken retrieves a refresh token entry by its hash. Returns nil if not found.
	GetToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error)
	// MarkTokenUsed sets the Used flag to true for the given token hash.
	MarkTokenUsed(ctx context.Context, tokenHash string) error
	// ConsumeToken atomically marks an unused token as used.
	// It returns the token entry, whether this call consumed it, and any store error.
	ConsumeToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, bool, error)
	// DeleteToken removes a refresh token entry by its hash.
	DeleteToken(ctx context.Context, tokenHash string) error
	// IsPersistent returns true if the store survives server restarts (e.g. Redis).
	// Used to decide fail-open vs fail-closed on token-not-found.
	IsPersistent() bool
}
