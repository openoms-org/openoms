package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type ShipmentRepository struct{}

func NewShipmentRepository() *ShipmentRepository {
	return &ShipmentRepository{}
}

func (r *ShipmentRepository) List(ctx context.Context, tx pgx.Tx, filter model.ShipmentListFilter) ([]model.Shipment, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Provider != nil {
		where += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, *filter.Provider)
		argIdx++
	}
	if filter.OrderID != nil {
		where += fmt.Sprintf(" AND order_id = $%d", argIdx)
		args = append(args, *filter.OrderID)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM shipments " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count shipments: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at": "created_at",
		"provider":   "provider",
		"status":     "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, order_id, provider, integration_id,
		        tracking_number, status, label_url, carrier_data,
		        warehouse_id, created_at, updated_at
		 FROM shipments %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list shipments: %w", err)
	}
	defer rows.Close()

	var shipments []model.Shipment
	for rows.Next() {
		var s model.Shipment
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.OrderID, &s.Provider, &s.IntegrationID,
			&s.TrackingNumber, &s.Status, &s.LabelURL, &s.CarrierData,
			&s.WarehouseID, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan shipment: %w", err)
		}
		shipments = append(shipments, s)
	}
	return shipments, total, rows.Err()
}

func (r *ShipmentRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Shipment, error) {
	var s model.Shipment
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, provider, integration_id,
		        tracking_number, status, label_url, carrier_data,
		        warehouse_id, created_at, updated_at
		 FROM shipments WHERE id = $1`, id,
	).Scan(
		&s.ID, &s.TenantID, &s.OrderID, &s.Provider, &s.IntegrationID,
		&s.TrackingNumber, &s.Status, &s.LabelURL, &s.CarrierData,
		&s.WarehouseID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find shipment by id: %w", err)
	}
	return &s, nil
}

func (r *ShipmentRepository) Create(ctx context.Context, tx pgx.Tx, shipment *model.Shipment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO shipments (
			id, tenant_id, order_id, provider, integration_id,
			tracking_number, status, label_url, carrier_data, warehouse_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`,
		shipment.ID, shipment.TenantID, shipment.OrderID, shipment.Provider, shipment.IntegrationID,
		shipment.TrackingNumber, shipment.Status, shipment.LabelURL, shipment.CarrierData,
		shipment.WarehouseID,
	).Scan(&shipment.CreatedAt, &shipment.UpdatedAt)
}

func (r *ShipmentRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateShipmentRequest) error {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.TrackingNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("tracking_number = $%d", argIdx))
		args = append(args, *req.TrackingNumber)
		argIdx++
	}
	if req.LabelURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("label_url = $%d", argIdx))
		args = append(args, *req.LabelURL)
		argIdx++
	}
	if req.CarrierData != nil {
		setClauses = append(setClauses, fmt.Sprintf("carrier_data = $%d", argIdx))
		args = append(args, req.CarrierData)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE shipments SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update shipment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("shipment not found")
	}
	return nil
}

func (r *ShipmentRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		"UPDATE shipments SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update shipment status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("shipment not found")
	}
	return nil
}

func (r *ShipmentRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM shipments WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete shipment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("shipment not found")
	}
	return nil
}
