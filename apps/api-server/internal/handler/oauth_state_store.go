package handler

import (
	"context"
	"time"
)

// OAuthStateStore abstracts storage for OAuth state parameters.
type OAuthStateStore interface {
	Save(ctx context.Context, state string, data *allegroOAuthState, ttl time.Duration) error
	Load(ctx context.Context, state string) (*allegroOAuthState, error) // Load and delete atomically
}
