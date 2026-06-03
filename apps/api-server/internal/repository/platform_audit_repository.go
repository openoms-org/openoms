package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PlatformAuditRepository persists platform-admin audit records to the
// dedicated, non-tenant-scoped platform_audit_log table.
type PlatformAuditRepository struct {
	pool *pgxpool.Pool
}

// NewPlatformAuditRepository creates a PlatformAuditRepository backed by the pool.
func NewPlatformAuditRepository(pool *pgxpool.Pool) *PlatformAuditRepository {
	return &PlatformAuditRepository{pool: pool}
}

// Log records a single platform-admin action.
func (r *PlatformAuditRepository) Log(ctx context.Context, entry model.PlatformAuditEntry) error {
	metaJSON, err := json.Marshal(entry.Metadata)
	if err != nil || entry.Metadata == nil {
		metaJSON = []byte("{}")
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO platform_audit_log (actor_user_id, action, target_type, target_id, metadata, ip_address)
		 VALUES ($1, $2, $3, $4, $5::jsonb, $6::inet)`,
		nilIfUUIDEmpty(entry.ActorUserID), entry.Action, nilIfEmpty(entry.TargetType),
		nilIfEmpty(entry.TargetID), string(metaJSON), nilIfEmpty(entry.IPAddress),
	)
	if err != nil {
		return fmt.Errorf("platform audit log: %w", err)
	}
	return nil
}
