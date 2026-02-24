package worker

import (
	"context"
	"encoding/json"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// TenantIntegration holds data needed for cross-tenant worker operations.
type TenantIntegration struct {
	TenantID      uuid.UUID
	IntegrationID uuid.UUID
	Provider      string
	SyncCursor    *string
	Credentials   string // encrypted
	Settings      json.RawMessage
}

// ListActiveIntegrations queries all active integrations for a given provider.
// This bypasses RLS -- runs directly on pool without set_config.
func ListActiveIntegrations(ctx context.Context, pool *pgxpool.Pool, provider string) ([]TenantIntegration, error) {
	rows, err := pool.Query(ctx,
		`SELECT tenant_id, id, provider, sync_cursor, credentials, settings
		   FROM integrations
		  WHERE provider = $1 AND status = 'active'`,
		provider,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TenantIntegration
	for rows.Next() {
		var ti TenantIntegration
		var credsJSON json.RawMessage
		if err := rows.Scan(
			&ti.TenantID,
			&ti.IntegrationID,
			&ti.Provider,
			&ti.SyncCursor,
			&credsJSON,
			&ti.Settings,
		); err != nil {
			return nil, err
		}
		// credentials column is JSONB storing a JSON string value (e.g. "base64...").
		// Unmarshal to strip the JSON quotes and get the raw base64 string.
		if len(credsJSON) > 0 {
			_ = json.Unmarshal(credsJSON, &ti.Credentials)
		}
		result = append(result, ti)
	}
	return result, rows.Err()
}

// providerCloser is implemented by providers that hold resources (e.g., rate limiter goroutines).
type providerCloser interface {
	Close()
}

// closeProvider closes a provider if it implements providerCloser.
func closeProvider(provider any) {
	if c, ok := provider.(providerCloser); ok {
		c.Close()
	}
}

// truncateErrorMessage limits error messages stored in the database to maxLen bytes.
// Truncates at a valid UTF-8 boundary to avoid invalid strings in PostgreSQL.
func truncateErrorMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	// Walk back from maxLen to find a valid UTF-8 boundary
	truncated := msg[:maxLen]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

// ListAllActiveMarketplaceIntegrations queries all active marketplace integrations
// across all registered marketplace providers. Only fetches integrations whose
// provider is registered in the marketplace factory (excludes carriers, invoicing, etc.).
func ListAllActiveMarketplaceIntegrations(ctx context.Context, pool *pgxpool.Pool) ([]TenantIntegration, error) {
	providers := integration.RegisteredMarketplaceProviders()
	if len(providers) == 0 {
		return nil, nil
	}

	rows, err := pool.Query(ctx,
		`SELECT tenant_id, id, provider, sync_cursor, credentials, settings
		   FROM integrations
		  WHERE status = 'active' AND provider = ANY($1)`,
		providers,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TenantIntegration
	for rows.Next() {
		var ti TenantIntegration
		var credsJSON json.RawMessage
		if err := rows.Scan(
			&ti.TenantID,
			&ti.IntegrationID,
			&ti.Provider,
			&ti.SyncCursor,
			&credsJSON,
			&ti.Settings,
		); err != nil {
			return nil, err
		}
		if len(credsJSON) > 0 {
			_ = json.Unmarshal(credsJSON, &ti.Credentials)
		}
		result = append(result, ti)
	}
	return result, rows.Err()
}
