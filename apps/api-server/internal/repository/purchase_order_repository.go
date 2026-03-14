package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// PurchaseOrderRepository implements PurchaseOrderRepo.
type PurchaseOrderRepository struct{}

// NewPurchaseOrderRepository creates a new PurchaseOrderRepository.
func NewPurchaseOrderRepository() *PurchaseOrderRepository {
	return &PurchaseOrderRepository{}
}

// List returns a paginated list of purchase orders matching the filter.
func (r *PurchaseOrderRepository) List(ctx context.Context, tx pgx.Tx, filter model.PurchaseOrderListFilter) ([]model.PurchaseOrder, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.SupplierID != nil {
		qb.Add("supplier_id = $%d", *filter.SupplierID)
	}

	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM purchase_orders %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count purchase orders: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":             "created_at",
		"po_number":              "po_number",
		"supplier_name":          "supplier_name",
		"status":                 "status",
		"total_amount":           "total_amount",
		"expected_delivery_date": "expected_delivery_date",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT id, tenant_id, po_number, supplier_id, supplier_name, warehouse_id,
		        status, notes, expected_delivery_date, total_amount, currency,
		        created_by, created_at, updated_at
		 FROM purchase_orders %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list purchase orders: %w", err)
	}
	defer rows.Close()

	var orders []model.PurchaseOrder
	for rows.Next() {
		var po model.PurchaseOrder
		if err := rows.Scan(
			&po.ID, &po.TenantID, &po.PONumber, &po.SupplierID, &po.SupplierName,
			&po.WarehouseID, &po.Status, &po.Notes, &po.ExpectedDeliveryDate,
			&po.TotalAmount, &po.Currency, &po.CreatedBy,
			&po.CreatedAt, &po.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan purchase order: %w", err)
		}
		orders = append(orders, po)
	}
	return orders, total, rows.Err()
}

// FindByID returns a purchase order by its ID.
func (r *PurchaseOrderRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PurchaseOrder, error) {
	var po model.PurchaseOrder
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, po_number, supplier_id, supplier_name, warehouse_id,
		        status, notes, expected_delivery_date, total_amount, currency,
		        created_by, created_at, updated_at
		 FROM purchase_orders WHERE id = $1`, id,
	).Scan(
		&po.ID, &po.TenantID, &po.PONumber, &po.SupplierID, &po.SupplierName,
		&po.WarehouseID, &po.Status, &po.Notes, &po.ExpectedDeliveryDate,
		&po.TotalAmount, &po.Currency, &po.CreatedBy,
		&po.CreatedAt, &po.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find purchase order by id: %w", err)
	}
	return &po, nil
}

// Create inserts a new purchase order.
func (r *PurchaseOrderRepository) Create(ctx context.Context, tx pgx.Tx, po *model.PurchaseOrder) error {
	return tx.QueryRow(ctx,
		`INSERT INTO purchase_orders (id, tenant_id, po_number, supplier_id, supplier_name,
		        warehouse_id, status, notes, expected_delivery_date, total_amount, currency, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING created_at, updated_at`,
		po.ID, po.TenantID, po.PONumber, po.SupplierID, po.SupplierName,
		po.WarehouseID, po.Status, po.Notes, po.ExpectedDeliveryDate,
		po.TotalAmount, po.Currency, po.CreatedBy,
	).Scan(&po.CreatedAt, &po.UpdatedAt)
}

// Update applies partial updates to a purchase order.
func (r *PurchaseOrderRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdatePurchaseOrderRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.SupplierID != nil {
		setClauses = append(setClauses, fmt.Sprintf("supplier_id = $%d", argIdx))
		args = append(args, *req.SupplierID)
		argIdx++
	}
	if req.SupplierName != nil {
		setClauses = append(setClauses, fmt.Sprintf("supplier_name = $%d", argIdx))
		args = append(args, *req.SupplierName)
		argIdx++
	}
	if req.WarehouseID != nil {
		setClauses = append(setClauses, fmt.Sprintf("warehouse_id = $%d", argIdx))
		args = append(args, *req.WarehouseID)
		argIdx++
	}
	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.ExpectedDeliveryDate != nil {
		if *req.ExpectedDeliveryDate == "" {
			setClauses = append(setClauses, "expected_delivery_date = NULL")
		} else {
			setClauses = append(setClauses, fmt.Sprintf("expected_delivery_date = $%d", argIdx))
			args = append(args, *req.ExpectedDeliveryDate)
			argIdx++
		}
	}
	if req.Currency != nil {
		setClauses = append(setClauses, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *req.Currency)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE purchase_orders SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update purchase order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("purchase order not found")
	}
	return nil
}

// UpdateStatus sets the status of a purchase order.
func (r *PurchaseOrderRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update purchase order status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("purchase order not found")
	}
	return nil
}

// UpdateTotalAmount sets the total amount of a purchase order.
func (r *PurchaseOrderRepository) UpdateTotalAmount(ctx context.Context, tx pgx.Tx, id uuid.UUID, amount float64) error {
	ct, err := tx.Exec(ctx,
		`UPDATE purchase_orders SET total_amount = $1, updated_at = NOW() WHERE id = $2`,
		amount, id,
	)
	if err != nil {
		return fmt.Errorf("update purchase order total: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("purchase order not found")
	}
	return nil
}

// Delete removes a purchase order by its ID.
func (r *PurchaseOrderRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM purchase_orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete purchase order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("purchase order not found")
	}
	return nil
}

// GeneratePONumber generates the next PO number in PO-YYYYMMDD-NNN format.
func (r *PurchaseOrderRepository) GeneratePONumber(ctx context.Context, tx pgx.Tx, _ uuid.UUID) (string, error) {
	today := time.Now().Format("20060102")
	prefix := "PO-" + today + "-"

	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM purchase_orders WHERE po_number LIKE $1`,
		prefix+"%",
	).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("count POs for today: %w", err)
	}

	return fmt.Sprintf("%s%03d", prefix, count+1), nil
}

// PurchaseOrderItemRepository handles purchase order item persistence.
type PurchaseOrderItemRepository struct{}

// NewPurchaseOrderItemRepository creates a new PurchaseOrderItemRepository.
func NewPurchaseOrderItemRepository() *PurchaseOrderItemRepository {
	return &PurchaseOrderItemRepository{}
}

// CreateItem inserts a new purchase order item.
func (r *PurchaseOrderItemRepository) CreateItem(ctx context.Context, tx pgx.Tx, item *model.PurchaseOrderItem) error {
	return tx.QueryRow(ctx,
		`INSERT INTO purchase_order_items (id, tenant_id, purchase_order_id, product_id, sku, product_name,
		        quantity_ordered, quantity_received, unit_cost, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING total_cost, created_at`,
		item.ID, item.TenantID, item.PurchaseOrderID, item.ProductID, item.SKU,
		item.ProductName, item.QuantityOrdered, item.QuantityReceived, item.UnitCost, item.Notes,
	).Scan(&item.TotalCost, &item.CreatedAt)
}

// ListByPOID returns all items for the given purchase order.
func (r *PurchaseOrderItemRepository) ListByPOID(ctx context.Context, tx pgx.Tx, poID uuid.UUID) ([]model.PurchaseOrderItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, purchase_order_id, product_id, sku, product_name,
		        quantity_ordered, quantity_received, unit_cost, total_cost, notes, created_at
		 FROM purchase_order_items WHERE purchase_order_id = $1 ORDER BY created_at ASC`, poID,
	)
	if err != nil {
		return nil, fmt.Errorf("list PO items: %w", err)
	}
	defer rows.Close()

	var items []model.PurchaseOrderItem
	for rows.Next() {
		var item model.PurchaseOrderItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.PurchaseOrderID, &item.ProductID,
			&item.SKU, &item.ProductName, &item.QuantityOrdered, &item.QuantityReceived,
			&item.UnitCost, &item.TotalCost, &item.Notes, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan PO item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// FindByID returns a purchase order item by its ID.
func (r *PurchaseOrderItemRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PurchaseOrderItem, error) {
	var item model.PurchaseOrderItem
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, purchase_order_id, product_id, sku, product_name,
		        quantity_ordered, quantity_received, unit_cost, total_cost, notes, created_at
		 FROM purchase_order_items WHERE id = $1`, id,
	).Scan(
		&item.ID, &item.TenantID, &item.PurchaseOrderID, &item.ProductID,
		&item.SKU, &item.ProductName, &item.QuantityOrdered, &item.QuantityReceived,
		&item.UnitCost, &item.TotalCost, &item.Notes, &item.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find PO item by id: %w", err)
	}
	return &item, nil
}

// UpdateReceived sets the quantity received for a purchase order item.
func (r *PurchaseOrderItemRepository) UpdateReceived(ctx context.Context, tx pgx.Tx, id uuid.UUID, quantityReceived int) error {
	ct, err := tx.Exec(ctx,
		`UPDATE purchase_order_items SET quantity_received = $1 WHERE id = $2`,
		quantityReceived, id,
	)
	if err != nil {
		return fmt.Errorf("update PO item received: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("purchase order item not found")
	}
	return nil
}

// DeleteByPOID deletes all items for the given purchase order.
func (r *PurchaseOrderItemRepository) DeleteByPOID(ctx context.Context, tx pgx.Tx, poID uuid.UUID) error {
	_, err := tx.Exec(ctx, "DELETE FROM purchase_order_items WHERE purchase_order_id = $1", poID)
	if err != nil {
		return fmt.Errorf("delete PO items: %w", err)
	}
	return nil
}
