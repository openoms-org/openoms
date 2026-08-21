package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// APITokenPrefix is prepended to newly minted long-lived API tokens so they
// are distinguishable from JWTs (which always contain two dots).
const APITokenPrefix = "oms_"

// APIToken is a long-lived owner credential. The raw secret is never stored;
// only TokenHash is persisted. List responses omit TokenHash via json:"-".
type APIToken struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"-"`
	UserID     uuid.UUID  `json:"-"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreatedAPIToken is returned only from create. Token is the raw secret.
type CreatedAPIToken struct {
	APIToken
	Token string `json:"token"`
}

// CreateAPITokenRequest is the body of POST /v1/api-tokens.
type CreateAPITokenRequest struct {
	Name string `json:"name"`
}

// Validate validates the create API token request.
func (r *CreateAPITokenRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	return validateMaxLength("name", r.Name, MaxNameLength)
}

// HashAPIToken returns the hex SHA-256 of a raw token (the stored token_hash).
func HashAPIToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
