package handler

import (
	"context"
	"time"
)

// OAuthStateStore abstracts storage for OAuth state parameters.
type OAuthStateStore interface {
	Save(ctx context.Context, state string, data *OAuthState, ttl time.Duration) error
	Load(ctx context.Context, state string) (*OAuthState, error) // Load and delete atomically
}
