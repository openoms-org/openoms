package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
)

var (
	// ErrProviderWebhookIntegrationNotFound is returned when a scoped provider webhook targets no active integration.
	ErrProviderWebhookIntegrationNotFound = errors.New("provider webhook integration not found")
)

// ProviderWebhookScope describes the tenant/integration proven by webhook signature verification.
type ProviderWebhookScope struct {
	TenantID      uuid.UUID
	IntegrationID uuid.UUID
	Provider      string
	Legacy        bool
}

// ProviderWebhookSecretResolver resolves tenant-scoped provider webhook secrets from encrypted integrations.
type ProviderWebhookSecretResolver struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
}

// NewProviderWebhookSecretResolver creates a resolver backed by the privileged worker pool.
func NewProviderWebhookSecretResolver(pool *pgxpool.Pool, encryptionKey []byte) *ProviderWebhookSecretResolver {
	return &ProviderWebhookSecretResolver{
		pool:          pool,
		encryptionKey: encryptionKey,
	}
}

// Resolve loads the active integration and returns its encrypted webhook signing secret.
func (r *ProviderWebhookSecretResolver) Resolve(ctx context.Context, provider string, integrationID uuid.UUID) (string, *ProviderWebhookScope, error) {
	if r == nil || r.pool == nil {
		return "", nil, ErrWebhookSecretNotConfigured
	}

	var scope ProviderWebhookScope
	var rawCredentials json.RawMessage
	err := r.pool.QueryRow(ctx,
		`SELECT tenant_id, id, provider, credentials
		   FROM integrations
		  WHERE id = $1 AND provider = $2 AND status = 'active'`,
		integrationID,
		provider,
	).Scan(&scope.TenantID, &scope.IntegrationID, &scope.Provider, &rawCredentials)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrProviderWebhookIntegrationNotFound
		}
		return "", nil, fmt.Errorf("resolve provider webhook integration: %w", err)
	}

	encrypted, err := decodeEncryptedCredentials(rawCredentials)
	if err != nil {
		return "", nil, err
	}
	if encrypted == "" {
		return "", nil, ErrWebhookSecretNotConfigured
	}

	credentials, err := crypto.Decrypt(encrypted, r.encryptionKey)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt provider webhook credentials: %w", err)
	}

	secret, ok := ExtractProviderWebhookSecret(credentials)
	if !ok {
		return "", nil, ErrWebhookSecretNotConfigured
	}

	return secret, &scope, nil
}

// ExtractProviderWebhookSecret returns the tenant-scoped webhook secret from decrypted integration credentials.
func ExtractProviderWebhookSecret(credentials []byte) (string, bool) {
	var values map[string]any
	if err := json.Unmarshal(credentials, &values); err != nil {
		return "", false
	}

	raw, ok := values["webhook_secret"]
	if !ok {
		return "", false
	}

	secret, ok := raw.(string)
	if !ok {
		return "", false
	}

	secret = strings.TrimSpace(secret)
	return secret, secret != ""
}

func decodeEncryptedCredentials(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var encrypted string
	if err := json.Unmarshal(raw, &encrypted); err != nil {
		return "", fmt.Errorf("decode encrypted integration credentials: %w", err)
	}
	return encrypted, nil
}
