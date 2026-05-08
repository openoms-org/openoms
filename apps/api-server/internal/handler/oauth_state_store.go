package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OAuthState holds the state + credentials needed to complete an OAuth flow.
// Shared by all OAuth providers (Allegro, OLX, Amazon, eBay, etc.).
type OAuthState struct {
	ExpiresAt     time.Time
	TenantID      uuid.UUID
	UserID        uuid.UUID
	Provider      string
	ClientID      string
	ClientSecret  string
	Sandbox       bool
	ApplicationID string // Amazon SP-API: application_id (separate from LWA client_id)
	MarketplaceID string // Amazon SP-API: marketplace_id for Seller Central URL
	DevID         string // eBay: developer ID (optional)
}

// OAuthStateStore abstracts storage for OAuth state parameters.
type OAuthStateStore interface {
	Save(ctx context.Context, state string, data *OAuthState, ttl time.Duration) error
	Load(ctx context.Context, state string) (*OAuthState, error) // Load and delete atomically
}
