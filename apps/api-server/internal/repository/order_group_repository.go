package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type OrderGroupRepository struct{}

func NewOrderGroupRepository() *OrderGroupRepository {
	return &OrderGroupRepository{}
}

func (r *OrderGroupRepository) Create(ctx context.Context, tx pgx.Tx, group *model.OrderGroup) error {
	return tx.QueryRow(ctx,
		`INSERT INTO order_groups (id, tenant_id, group_type, source_order_ids, target_order_ids, notes, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		group.ID, group.TenantID, group.GroupType, group.SourceOrderIDs, group.TargetOrderIDs,
		group.Notes, group.CreatedBy,
	).Scan(&group.CreatedAt)
}

func (r *OrderGroupRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.OrderGroup, error) {
	var g model.OrderGroup
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, group_type, source_order_ids, target_order_ids, notes, created_by, created_at
		 FROM order_groups WHERE id = $1`, id,
	).Scan(&g.ID, &g.TenantID, &g.GroupType, &g.SourceOrderIDs, &g.TargetOrderIDs,
		&g.Notes, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find order group by id: %w", err)
	}
	return &g, nil
}

func (r *OrderGroupRepository) ListByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.OrderGroup, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, group_type, source_order_ids, target_order_ids, notes, created_by, created_at
		 FROM order_groups
		 WHERE $1 = ANY(source_order_ids) OR $1 = ANY(target_order_ids)
		 ORDER BY created_at DESC`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("list order groups by order id: %w", err)
	}
	defer rows.Close()

	var groups []model.OrderGroup
	for rows.Next() {
		var g model.OrderGroup
		if err := rows.Scan(&g.ID, &g.TenantID, &g.GroupType, &g.SourceOrderIDs, &g.TargetOrderIDs,
			&g.Notes, &g.CreatedBy, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}
