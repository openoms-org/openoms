package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// AuditRepository handles persistence for audit log entries.
type AuditRepository struct{}

// NewAuditRepository creates a new AuditRepository.
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
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::inet)`,
		entry.TenantID, nilIfUUIDEmpty(entry.UserID), entry.Action, entry.EntityType, entry.EntityID,
		string(changesJSON), nilIfEmpty(entry.IPAddress),
	)
	if err != nil {
		return fmt.Errorf("audit log: %w", err)
	}
	return nil
}

// ListByEntity returns audit log entries for a specific entity.
func (r *AuditRepository) ListByEntity(ctx context.Context, tx pgx.Tx, entityType string, entityID uuid.UUID) ([]model.AuditLogEntry, error) {
	rows, err := tx.Query(ctx,
		`SELECT a.id, u.name, a.action, a.entity_type, a.entity_id::text, a.changes, a.ip_address::text, a.created_at
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
		if err := rows.Scan(&entry.ID, &entry.UserName, &entry.Action, &entry.EntityType, &entry.EntityID, &changesJSON, &entry.IPAddress, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if len(changesJSON) > 0 {
			if err := json.Unmarshal(changesJSON, &entry.Changes); err != nil {
				slog.Warn("failed to unmarshal audit entry changes", "error", err, "entry_id", entry.ID)
			}
		}
		if entry.Changes == nil {
			entry.Changes = map[string]string{}
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

// List returns a paginated, filtered list of audit log entries across all entities.
func (r *AuditRepository) List(ctx context.Context, tx pgx.Tx, filter model.AuditListFilter) ([]model.AuditLogEntry, int, error) {
	qb := NewQueryBuilder()

	if filter.EntityType != nil {
		qb.Add("a.entity_type = $%d", *filter.EntityType)
	}
	if filter.Action != nil {
		qb.Add("a.action = $%d", *filter.Action)
	}
	if filter.UserID != nil {
		qb.Add("a.user_id = $%d", *filter.UserID)
	}

	where := qb.WhereClause()

	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) FROM audit_log a %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit log: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	argIdx := qb.AddArgs(limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT a.id, u.name, a.action, a.entity_type, a.entity_id::text, a.changes, a.ip_address::text, a.created_at
		 FROM audit_log a
		 LEFT JOIN users u ON a.user_id = u.id
		 %s
		 ORDER BY a.created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditLogEntry
	for rows.Next() {
		var entry model.AuditLogEntry
		var changesJSON []byte
		if err := rows.Scan(&entry.ID, &entry.UserName, &entry.Action, &entry.EntityType, &entry.EntityID, &changesJSON, &entry.IPAddress, &entry.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan audit entry: %w", err)
		}
		if len(changesJSON) > 0 {
			if err := json.Unmarshal(changesJSON, &entry.Changes); err != nil {
				slog.Warn("failed to unmarshal audit entry changes", "error", err, "entry_id", entry.ID)
			}
		}
		if entry.Changes == nil {
			entry.Changes = map[string]string{}
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilIfUUIDEmpty(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
