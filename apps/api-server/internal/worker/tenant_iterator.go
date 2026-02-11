package worker

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
		if err := rows.Scan(
			&ti.TenantID,
			&ti.IntegrationID,
			&ti.Provider,
			&ti.SyncCursor,
			&ti.Credentials,
			&ti.Settings,
		); err != nil {
			return nil, err
		}
		result = append(result, ti)
	}
	return result, rows.Err()
}

// ListAllActiveMarketplaceIntegrations queries all active marketplace integrations
// across all providers. This is used by the stock sync worker.
func ListAllActiveMarketplaceIntegrations(ctx context.Context, pool *pgxpool.Pool) ([]TenantIntegration, error) {
	rows, err := pool.Query(ctx,
		`SELECT tenant_id, id, provider, sync_cursor, credentials, settings
		   FROM integrations
		  WHERE status = 'active'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TenantIntegration
	for rows.Next() {
		var ti TenantIntegration
		if err := rows.Scan(
			&ti.TenantID,
			&ti.IntegrationID,
			&ti.Provider,
			&ti.SyncCursor,
			&ti.Credentials,
			&ti.Settings,
		); err != nil {
			return nil, err
		}
		result = append(result, ti)
	}
	return result, rows.Err()
}
