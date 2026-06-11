package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// DropshipOrderRepository implements DropshipOrderRepo.
type DropshipOrderRepository struct{}

// NewDropshipOrderRepository creates a new DropshipOrderRepository.
func NewDropshipOrderRepository() *DropshipOrderRepository {
	return &DropshipOrderRepository{}
}

// List returns a paginated list of dropship orders matching the filter.
func (r *DropshipOrderRepository) List(ctx context.Context, tx pgx.Tx, filter model.DropshipOrderListFilter) ([]model.DropshipOrder, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.SupplierID != nil {
		qb.Add("supplier_id = $%d", *filter.SupplierID)
	}
	if filter.OrderID != nil {
		qb.Add("order_id = $%d", *filter.OrderID)
	}

	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM dropship_orders %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count dropship orders: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":    "created_at",
		"supplier_name": "supplier_name",
		"status":        "status",
		"total_cost":    "total_cost",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, order_id, supplier_id, supplier_name, status,
		        supplier_reference, tracking_number, carrier, notes,
		        total_cost, currency, sent_at, confirmed_at, shipped_at, delivered_at, submit_attempted_at,
		        created_at, updated_at
		 FROM dropship_orders %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list dropship orders: %w", err)
	}
	defer rows.Close()

	var orders []model.DropshipOrder
	for rows.Next() {
		var d model.DropshipOrder
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.OrderID, &d.SupplierID, &d.SupplierName, &d.Status,
			&d.SupplierReference, &d.TrackingNumber, &d.Carrier, &d.Notes,
			&d.TotalCost, &d.Currency, &d.SentAt, &d.ConfirmedAt, &d.ShippedAt, &d.DeliveredAt, &d.SubmitAttemptedAt,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan dropship order: %w", err)
		}
		orders = append(orders, d)
	}
	return orders, total, rows.Err()
}

// FindByID returns a dropship order by its ID.
func (r *DropshipOrderRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.DropshipOrder, error) {
	var d model.DropshipOrder
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, supplier_id, supplier_name, status,
		        supplier_reference, tracking_number, carrier, notes,
		        total_cost, currency, sent_at, confirmed_at, shipped_at, delivered_at, submit_attempted_at,
		        created_at, updated_at
		 FROM dropship_orders WHERE id = $1`, id,
	).Scan(
		&d.ID, &d.TenantID, &d.OrderID, &d.SupplierID, &d.SupplierName, &d.Status,
		&d.SupplierReference, &d.TrackingNumber, &d.Carrier, &d.Notes,
		&d.TotalCost, &d.Currency, &d.SentAt, &d.ConfirmedAt, &d.ShippedAt, &d.DeliveredAt, &d.SubmitAttemptedAt,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find dropship order by id: %w", err)
	}
	return &d, nil
}

// FindByIDForUpdate returns a dropship order by its ID, locking the row (SELECT ... FOR
// UPDATE) for the duration of the transaction. Both supplier-submit paths (the orchestration
// handler and the manual status transition) must lock the row BEFORE checking
// supplier_reference so concurrent submitters serialize and the second one observes the
// first one's outcome (OPE-518).
func (r *DropshipOrderRepository) FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.DropshipOrder, error) {
	var d model.DropshipOrder
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, supplier_id, supplier_name, status,
		        supplier_reference, tracking_number, carrier, notes,
		        total_cost, currency, sent_at, confirmed_at, shipped_at, delivered_at, submit_attempted_at,
		        created_at, updated_at
		 FROM dropship_orders WHERE id = $1 FOR UPDATE`, id,
	).Scan(
		&d.ID, &d.TenantID, &d.OrderID, &d.SupplierID, &d.SupplierName, &d.Status,
		&d.SupplierReference, &d.TrackingNumber, &d.Carrier, &d.Notes,
		&d.TotalCost, &d.Currency, &d.SentAt, &d.ConfirmedAt, &d.ShippedAt, &d.DeliveredAt, &d.SubmitAttemptedAt,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find dropship order by id for update: %w", err)
	}
	return &d, nil
}

// FindByOrderID returns all dropship orders for the given order.
func (r *DropshipOrderRepository) FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.DropshipOrder, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, order_id, supplier_id, supplier_name, status,
		        supplier_reference, tracking_number, carrier, notes,
		        total_cost, currency, sent_at, confirmed_at, shipped_at, delivered_at, submit_attempted_at,
		        created_at, updated_at
		 FROM dropship_orders WHERE order_id = $1 ORDER BY created_at ASC`, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("find dropship orders by order: %w", err)
	}
	defer rows.Close()

	var orders []model.DropshipOrder
	for rows.Next() {
		var d model.DropshipOrder
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.OrderID, &d.SupplierID, &d.SupplierName, &d.Status,
			&d.SupplierReference, &d.TrackingNumber, &d.Carrier, &d.Notes,
			&d.TotalCost, &d.Currency, &d.SentAt, &d.ConfirmedAt, &d.ShippedAt, &d.DeliveredAt, &d.SubmitAttemptedAt,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dropship order: %w", err)
		}
		orders = append(orders, d)
	}
	return orders, rows.Err()
}

// Create inserts a new dropship order.
func (r *DropshipOrderRepository) Create(ctx context.Context, tx pgx.Tx, d *model.DropshipOrder) error {
	return tx.QueryRow(ctx,
		`INSERT INTO dropship_orders (id, tenant_id, order_id, supplier_id, supplier_name,
		        status, supplier_reference, tracking_number, carrier, notes,
		        total_cost, currency)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING created_at, updated_at`,
		d.ID, d.TenantID, d.OrderID, d.SupplierID, d.SupplierName,
		d.Status, d.SupplierReference, d.TrackingNumber, d.Carrier, d.Notes,
		d.TotalCost, d.Currency,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
}

// UpdateStatus sets the status of a dropship order.
func (r *DropshipOrderRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		`UPDATE dropship_orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update dropship order status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("dropship order not found")
	}
	return nil
}

// UpdateFields updates mutable fields on a dropship order.
func (r *DropshipOrderRepository) UpdateFields(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateDropshipStatusRequest) error {
	ub := NewUpdateBuilder()
	ub.Set("status", req.Status)
	SetPtr(ub, "tracking_number", req.TrackingNumber)
	SetPtr(ub, "carrier", req.Carrier)
	SetPtr(ub, "supplier_reference", req.SupplierReference)
	SetPtr(ub, "notes", req.Notes)

	// Set timestamp columns based on status
	switch req.Status {
	case "sent":
		ub.SetRaw("sent_at = NOW()")
	case "confirmed":
		ub.SetRaw("confirmed_at = NOW()")
	case "shipped":
		ub.SetRaw("shipped_at = NOW()")
	case "delivered":
		ub.SetRaw("delivered_at = NOW()")
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE dropship_orders SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update dropship order fields: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("dropship order not found")
	}
	return nil
}

// MarkSubmitAttempted sets the two-phase submit-intent marker (OPE-517). The caller must
// hold the row lock (FindByIDForUpdate) in the same transaction and commit BEFORE calling
// the supplier's CreateOrder. The marker is never cleared — it is historical evidence that
// a submission was attempted.
func (r *DropshipOrderRepository) MarkSubmitAttempted(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`UPDATE dropship_orders SET submit_attempted_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark dropship order submit attempted: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("dropship order not found")
	}
	return nil
}

// Delete removes a dropship order by its ID.
func (r *DropshipOrderRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM dropship_orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete dropship order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("dropship order not found")
	}
	return nil
}

// DropshipOrderItemRepository handles dropship order item persistence.
type DropshipOrderItemRepository struct{}

// NewDropshipOrderItemRepository creates a new DropshipOrderItemRepository.
func NewDropshipOrderItemRepository() *DropshipOrderItemRepository {
	return &DropshipOrderItemRepository{}
}

// CreateItem inserts a new dropship order item.
func (r *DropshipOrderItemRepository) CreateItem(ctx context.Context, tx pgx.Tx, item *model.DropshipOrderItem) error {
	return tx.QueryRow(ctx,
		`INSERT INTO dropship_order_items (id, tenant_id, dropship_order_id, product_id, sku,
		        product_name, quantity, unit_cost)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		item.ID, item.TenantID, item.DropshipOrderID, item.ProductID, item.SKU,
		item.ProductName, item.Quantity, item.UnitCost,
	).Scan(&item.CreatedAt)
}

// ListByDropshipOrderID returns all items for the given dropship order.
func (r *DropshipOrderItemRepository) ListByDropshipOrderID(ctx context.Context, tx pgx.Tx, dropshipOrderID uuid.UUID) ([]model.DropshipOrderItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, dropship_order_id, product_id, sku, product_name,
		        quantity, unit_cost, created_at
		 FROM dropship_order_items WHERE dropship_order_id = $1 ORDER BY created_at ASC`, dropshipOrderID,
	)
	if err != nil {
		return nil, fmt.Errorf("list dropship order items: %w", err)
	}
	defer rows.Close()

	var items []model.DropshipOrderItem
	for rows.Next() {
		var item model.DropshipOrderItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.DropshipOrderID, &item.ProductID,
			&item.SKU, &item.ProductName, &item.Quantity, &item.UnitCost, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dropship order item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// DeleteByDropshipOrderID deletes all items for the given dropship order.
func (r *DropshipOrderItemRepository) DeleteByDropshipOrderID(ctx context.Context, tx pgx.Tx, dropshipOrderID uuid.UUID) error {
	_, err := tx.Exec(ctx, "DELETE FROM dropship_order_items WHERE dropship_order_id = $1", dropshipOrderID)
	if err != nil {
		return fmt.Errorf("delete dropship order items: %w", err)
	}
	return nil
}
