package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/google/uuid"
)

// ExternalWorkflowToken is an RBAC-scoped, hashed bearer token authenticating an external
// workflow engine's callbacks for one integration (OPE-421/Phase-13). Mirrors
// supplier_portal_tokens. The raw token is shown once at creation; only token_hash is stored.
type ExternalWorkflowToken struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	IntegrationID uuid.UUID  `json:"integration_id"`
	TokenHash     string     `json:"-"`
	Scopes        []string   `json:"scopes"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ScopeAllows reports whether the token may submit the given callback command. An empty
// scope set means resolve-only (no follow-on command permitted).
func (t ExternalWorkflowToken) ScopeAllows(command string) bool {
	return slices.Contains(t.Scopes, command)
}

// IsExpired reports whether the token has an expiry in the past.
func (t ExternalWorkflowToken) IsExpired(now time.Time) bool {
	return t.ExpiresAt != nil && now.After(*t.ExpiresAt)
}

// whitelistedCallbackCommands are the only typed commands a callback may submit (each is
// applied through the audited orchestration outbox; the external engine never writes state).
var whitelistedCallbackCommands = []string{"set_status", "add_tag", "add_note"}

// IsWhitelistedCallbackCommand reports whether c is a command a callback may submit.
func IsWhitelistedCallbackCommand(c string) bool {
	return slices.Contains(whitelistedCallbackCommands, c)
}

// ExternalWorkflowCallbackRequest is the callback body posted by the external engine.
type ExternalWorkflowCallbackRequest struct {
	CorrelationNonce string `json:"correlation_nonce"`
	Status           string `json:"status"` // "succeeded" | "failed"
	ExternalExecID   string `json:"external_execution_id,omitempty"`
	// Optional single follow-on command, applied only if in the token's scopes.
	Command      string `json:"command,omitempty"`       // set_status | add_tag | add_note
	CommandValue string `json:"command_value,omitempty"` // e.g. the status / tag / note text
}

// BuildRedactedPayload projects an order map onto an allowlist of field paths. Nothing
// outside the allowlist is included — secrets, internal ids and unlisted PII never leave.
func BuildRedactedPayload(order map[string]any, allowlist []string) map[string]any {
	out := make(map[string]any, len(allowlist))
	for _, key := range allowlist {
		if v, ok := order[key]; ok {
			out[key] = v
		}
	}
	return out
}

// HashExternalWorkflowToken returns the hex sha256 of a raw token (the stored token_hash).
func HashExternalWorkflowToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// SignExternalWorkflowBody returns the "sha256=<hex>" HMAC of body under secret (the
// X-Signature-256 header value), matching the outgoing-webhook signing convention.
func SignExternalWorkflowBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyExternalWorkflowSignature constant-time compares a provided "sha256=..." signature
// against the expected HMAC of body under secret.
func VerifyExternalWorkflowSignature(body []byte, provided, secret string) bool {
	expected := SignExternalWorkflowBody(body, secret)
	return hmac.Equal([]byte(provided), []byte(expected))
}
