package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ShipmentRepository handles persistence for shipments.
type ShipmentRepository struct{}

// NewShipmentRepository creates a new ShipmentRepository.
func NewShipmentRepository() *ShipmentRepository {
	return &ShipmentRepository{}
}

// shipmentColumns is the canonical column list for all SELECT queries.
const shipmentColumns = `id, tenant_id, order_id, provider, integration_id,
		        external_id, tracking_number, status, label_url, carrier_data,
		        warehouse_id, package_number, weight, dimensions_length,
		        dimensions_width, dimensions_height, notes,
		        carbon_kg, distance_km, carbon_method,
		        created_at, updated_at`

// scanShipment scans a row into a Shipment struct using the canonical column order.
func scanShipment(row interface{ Scan(dest ...any) error }) (model.Shipment, error) {
	var s model.Shipment
	err := row.Scan(
		&s.ID, &s.TenantID, &s.OrderID, &s.Provider, &s.IntegrationID,
		&s.ExternalID, &s.TrackingNumber, &s.Status, &s.LabelURL, &s.CarrierData,
		&s.WarehouseID, &s.PackageNumber, &s.Weight, &s.Length,
		&s.Width, &s.Height, &s.Notes,
		&s.CarbonKg, &s.DistanceKm, &s.CarbonMethod,
		&s.CreatedAt, &s.UpdatedAt,
	)
	return s, err
}

// List returns a paginated list of shipments matching the filter.
func (r *ShipmentRepository) List(ctx context.Context, tx pgx.Tx, filter model.ShipmentListFilter) ([]model.Shipment, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.Provider != nil {
		qb.Add("provider = $%d", *filter.Provider)
	}
	if filter.OrderID != nil {
		qb.Add("order_id = $%d", *filter.OrderID)
	}
	if len(filter.OrderIDs) > 0 {
		qb.Add("order_id = ANY($%d)", filter.OrderIDs)
	}

	where := qb.WhereClause()

	var total int
	countQuery := "SELECT COUNT(*) FROM shipments " + where
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count shipments: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"provider":       "provider",
		"status":         "status",
		"package_number": "package_number",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s
		 FROM shipments %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		shipmentColumns, where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list shipments: %w", err)
	}
	defer rows.Close()

	var shipments []model.Shipment
	for rows.Next() {
		s, err := scanShipment(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan shipment: %w", err)
		}
		shipments = append(shipments, s)
	}
	return shipments, total, rows.Err()
}

// FindByID returns a shipment by its ID.
func (r *ShipmentRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Shipment, error) {
	row := tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM shipments WHERE id = $1`, shipmentColumns), id,
	)
	s, err := scanShipment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find shipment by id: %w", err)
	}
	return &s, nil
}

// FindByExternalID looks up a shipment by its external system ID (e.g. Allegro shipment ID).
func (r *ShipmentRepository) FindByExternalID(ctx context.Context, tx pgx.Tx, externalID string) (*model.Shipment, error) {
	row := tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM shipments WHERE external_id = $1`, shipmentColumns), externalID,
	)
	s, err := scanShipment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find shipment by external id: %w", err)
	}
	return &s, nil
}

// CountByOrder returns the number of shipments for the given order.
func (r *ShipmentRepository) CountByOrder(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM shipments WHERE order_id = $1", orderID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count shipments by order: %w", err)
	}
	return count, nil
}

// Create inserts a new shipment.
func (r *ShipmentRepository) Create(ctx context.Context, tx pgx.Tx, shipment *model.Shipment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO shipments (
			id, tenant_id, order_id, provider, integration_id,
			external_id, tracking_number, status, label_url, carrier_data,
			warehouse_id, package_number, weight, dimensions_length,
			dimensions_width, dimensions_height, notes,
			carbon_kg, distance_km, carbon_method
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING created_at, updated_at`,
		shipment.ID, shipment.TenantID, shipment.OrderID, shipment.Provider, shipment.IntegrationID,
		shipment.ExternalID, shipment.TrackingNumber, shipment.Status, shipment.LabelURL, shipment.CarrierData,
		shipment.WarehouseID, shipment.PackageNumber, shipment.Weight, shipment.Length,
		shipment.Width, shipment.Height, shipment.Notes,
		shipment.CarbonKg, shipment.DistanceKm, shipment.CarbonMethod,
	).Scan(&shipment.CreatedAt, &shipment.UpdatedAt)
}

// Update applies partial updates to a shipment.
func (r *ShipmentRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateShipmentRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "tracking_number", req.TrackingNumber)
	SetPtr(ub, "label_url", req.LabelURL)
	if req.CarrierData != nil {
		ub.Set("carrier_data", req.CarrierData)
	}
	SetPtr(ub, "weight", req.Weight)
	SetPtr(ub, "dimensions_length", req.Length)
	SetPtr(ub, "dimensions_width", req.Width)
	SetPtr(ub, "dimensions_height", req.Height)
	SetPtr(ub, "notes", req.Notes)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	args := append(ub.Args(), id)

	query := fmt.Sprintf("UPDATE shipments SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update shipment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("shipment not found")
	}
	return nil
}

// UpdateStatus sets the status of a shipment.
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

// UpdateStatusIfCurrent atomically transitions a shipment's status only if it
// currently equals expected. Returns true on success, false if the row was not
// in the expected state (prevents concurrent label generation races).
func (r *ShipmentRepository) UpdateStatusIfCurrent(ctx context.Context, tx pgx.Tx, id uuid.UUID, expected, newStatus string) (bool, error) {
	ct, err := tx.Exec(ctx,
		"UPDATE shipments SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3",
		newStatus, id, expected,
	)
	if err != nil {
		return false, fmt.Errorf("update shipment status (CAS): %w", err)
	}
	return ct.RowsAffected() == 1, nil
}

// Delete removes a shipment by its ID.
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
