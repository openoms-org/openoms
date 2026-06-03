package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProviderPublicationRepository persists the append-only publication-event audit
// trail and the private-beta tenant allowlist.
type ProviderPublicationRepository struct {
	pool *pgxpool.Pool
}

// NewProviderPublicationRepository creates the repository backed by the pool.
func NewProviderPublicationRepository(pool *pgxpool.Pool) *ProviderPublicationRepository {
	return &ProviderPublicationRepository{pool: pool}
}

// LogEvent appends one publication event.
func (r *ProviderPublicationRepository) LogEvent(ctx context.Context, q Querier, versionID uuid.UUID, fromState *string, toState, action string, actorUserID *uuid.UUID, reason string, metadata any) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil || metadata == nil {
		metaJSON = []byte("{}")
	}
	_, err = q.Exec(ctx,
		`INSERT INTO provider_publication_events (provider_version_id, from_state, to_state, action, actor_user_id, reason, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)`,
		versionID, fromState, toState, action, actorUserID, reason, string(metaJSON))
	if err != nil {
		return fmt.Errorf("log publication event: %w", err)
	}
	return nil
}

// ListEvents returns the publication events for a version, newest first.
func (r *ProviderPublicationRepository) ListEvents(ctx context.Context, versionID uuid.UUID) ([]model.ProviderPublicationEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, provider_version_id, from_state, to_state, action, actor_user_id, reason, created_at
		   FROM provider_publication_events WHERE provider_version_id = $1 ORDER BY created_at DESC, id DESC`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list publication events: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderPublicationEvent{}
	for rows.Next() {
		var e model.ProviderPublicationEvent
		if err := rows.Scan(&e.ID, &e.ProviderVersionID, &e.FromState, &e.ToState, &e.Action, &e.ActorUserID, &e.Reason, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan publication event: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// EnableTenant adds (idempotently) a tenant to a version's private-beta allowlist.
func (r *ProviderPublicationRepository) EnableTenant(ctx context.Context, versionID, tenantID uuid.UUID, enabledBy *uuid.UUID) (*model.ProviderTenantEnable, error) {
	var e model.ProviderTenantEnable
	err := r.pool.QueryRow(ctx,
		`INSERT INTO provider_tenant_enables (provider_version_id, tenant_id, enabled_by)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (provider_version_id, tenant_id)
		 DO UPDATE SET created_at = provider_tenant_enables.created_at
		 RETURNING id, provider_version_id, tenant_id, enabled_by, created_at`,
		versionID, tenantID, enabledBy).
		Scan(&e.ID, &e.ProviderVersionID, &e.TenantID, &e.EnabledBy, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("enable tenant: %w", err)
	}
	return &e, nil
}

// ListTenantEnables returns the tenants enabled for a version.
func (r *ProviderPublicationRepository) ListTenantEnables(ctx context.Context, versionID uuid.UUID) ([]model.ProviderTenantEnable, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, provider_version_id, tenant_id, enabled_by, created_at
		   FROM provider_tenant_enables WHERE provider_version_id = $1 ORDER BY created_at DESC`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list tenant enables: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderTenantEnable{}
	for rows.Next() {
		var e model.ProviderTenantEnable
		if err := rows.Scan(&e.ID, &e.ProviderVersionID, &e.TenantID, &e.EnabledBy, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant enable: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// IsTenantEnabled reports whether a tenant is on a version's private-beta allowlist.
func (r *ProviderPublicationRepository) IsTenantEnabled(ctx context.Context, versionID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM provider_tenant_enables WHERE provider_version_id = $1 AND tenant_id = $2)`,
		versionID, tenantID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check tenant enabled: %w", err)
	}
	return exists, nil
}
