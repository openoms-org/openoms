package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type AuditRepository struct{}

func NewAuditRepository() *AuditRepository {
	return &AuditRepository{}
}

// Log creates an audit log entry within a WithTenant transaction.
func (r *AuditRepository) Log(ctx context.Context, tx pgx.Tx, entry model.AuditEntry) error {
	changesJSON, err := json.Marshal(entry.Changes)
	if err != nil {
		changesJSON = []byte("{}")
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO audit_log (tenant_id, user_id, action, entity_type, entity_id, changes, ip_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::inet)`,
		entry.TenantID, entry.UserID, entry.Action, entry.EntityType, entry.EntityID,
		changesJSON, nilIfEmpty(entry.IPAddress),
	)
	if err != nil {
		return fmt.Errorf("audit log: %w", err)
	}
	return nil
}

// ListByEntity returns audit log entries for a specific entity.
func (r *AuditRepository) ListByEntity(ctx context.Context, tx pgx.Tx, entityType string, entityID uuid.UUID) ([]model.AuditLogEntry, error) {
	rows, err := tx.Query(ctx,
		`SELECT a.id, u.name, a.action, a.changes, a.ip_address::text, a.created_at
		 FROM audit_log a
		 LEFT JOIN users u ON u.id = a.user_id
		 WHERE a.entity_type = $1 AND a.entity_id = $2
		 ORDER BY a.created_at DESC
		 LIMIT 50`, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("list audit by entity: %w", err)
	}
	defer rows.Close()

	result := []model.AuditLogEntry{}
	for rows.Next() {
		var entry model.AuditLogEntry
		var changesJSON []byte
		if err := rows.Scan(&entry.ID, &entry.UserName, &entry.Action, &changesJSON, &entry.IPAddress, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if len(changesJSON) > 0 {
			_ = json.Unmarshal(changesJSON, &entry.Changes)
		}
		if entry.Changes == nil {
			entry.Changes = map[string]string{}
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
