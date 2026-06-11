package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// RecurringOrderRepository handles persistence for recurring orders.
type RecurringOrderRepository struct{}

// NewRecurringOrderRepository creates a new RecurringOrderRepository.
func NewRecurringOrderRepository() *RecurringOrderRepository {
	return &RecurringOrderRepository{}
}

var recurringOrderColumns = `id, tenant_id, customer_id, customer_name, customer_email,
	status, frequency, interval_days, next_order_date, last_order_date,
	end_date, total_orders_created, max_orders, shipping_address, notes,
	created_by, created_at, updated_at`

func scanRecurringOrder(row interface{ Scan(dest ...any) error }) (*model.RecurringOrder, error) {
	var ro model.RecurringOrder
	err := row.Scan(
		&ro.ID, &ro.TenantID, &ro.CustomerID, &ro.CustomerName, &ro.CustomerEmail,
		&ro.Status, &ro.Frequency, &ro.IntervalDays, &ro.NextOrderDate, &ro.LastOrderDate,
		&ro.EndDate, &ro.TotalOrdersCreated, &ro.MaxOrders, &ro.ShippingAddress, &ro.Notes,
		&ro.CreatedBy, &ro.CreatedAt, &ro.UpdatedAt,
	)
	return &ro, err
}

// List returns a paginated list of recurring orders matching the filter.
func (r *RecurringOrderRepository) List(ctx context.Context, tx pgx.Tx, filter model.RecurringOrderListFilter) ([]model.RecurringOrder, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil && *filter.Status != "" {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.CustomerID != nil && *filter.CustomerID != "" {
		qb.Add("customer_id = $%d", *filter.CustomerID)
	}
	if filter.NextDateBefore != nil && *filter.NextDateBefore != "" {
		qb.Add("next_order_date <= $%d", *filter.NextDateBefore)
	}

	where := qb.WhereClause()

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM recurring_orders %s", where)
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count recurring orders: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":      "created_at",
		"next_order_date": "next_order_date",
		"customer_name":   "customer_name",
		"status":          "status",
		"frequency":       "frequency",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s FROM recurring_orders %s %s LIMIT $%d OFFSET $%d`,
		recurringOrderColumns, where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list recurring orders: %w", err)
	}
	defer rows.Close()

	var orders []model.RecurringOrder
	for rows.Next() {
		ro, err := scanRecurringOrder(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan recurring order: %w", err)
		}
		orders = append(orders, *ro)
	}
	return orders, total, rows.Err()
}

// FindByID returns a recurring order by its ID.
func (r *RecurringOrderRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.RecurringOrder, error) {
	ro, err := scanRecurringOrder(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM recurring_orders WHERE id = $1", recurringOrderColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find recurring order by id: %w", err)
	}
	return ro, nil
}

// Create inserts a new recurring order.
func (r *RecurringOrderRepository) Create(ctx context.Context, tx pgx.Tx, ro *model.RecurringOrder) error {
	return tx.QueryRow(ctx,
		`INSERT INTO recurring_orders (id, tenant_id, customer_id, customer_name, customer_email,
		 status, frequency, interval_days, next_order_date, end_date,
		 max_orders, shipping_address, notes, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 RETURNING created_at, updated_at`,
		ro.ID, ro.TenantID, ro.CustomerID, ro.CustomerName, ro.CustomerEmail,
		ro.Status, ro.Frequency, ro.IntervalDays, ro.NextOrderDate, ro.EndDate,
		ro.MaxOrders, ro.ShippingAddress, ro.Notes, ro.CreatedBy,
	).Scan(&ro.CreatedAt, &ro.UpdatedAt)
}

// Update applies partial updates to a recurring order.
func (r *RecurringOrderRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateRecurringOrderRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "customer_id", req.CustomerID)
	SetPtr(ub, "customer_name", req.CustomerName)
	SetPtr(ub, "customer_email", req.CustomerEmail)
	if req.Frequency != nil {
		ub.Set("frequency", *req.Frequency)
		ub.Set("interval_days", model.FrequencyToIntervalDays(*req.Frequency))
	}
	SetPtr(ub, "next_order_date", req.NextOrderDate)
	SetPtr(ub, "end_date", req.EndDate)
	SetPtr(ub, "max_orders", req.MaxOrders)
	if req.ShippingAddress != nil {
		ub.Set("shipping_address", req.ShippingAddress)
	}
	SetPtr(ub, "notes", req.Notes)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	query := fmt.Sprintf("UPDATE recurring_orders SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update recurring order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("recurring order not found")
	}
	return nil
}

// UpdateStatus sets the status of a recurring order.
func (r *RecurringOrderRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		"UPDATE recurring_orders SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update recurring order status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("recurring order not found")
	}
	return nil
}

// Delete removes a recurring order by its ID.
func (r *RecurringOrderRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM recurring_orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete recurring order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("recurring order not found")
	}
	return nil
}

// FindDue returns all active recurring orders where next_order_date <= today.
func (r *RecurringOrderRepository) FindDue(ctx context.Context, tx pgx.Tx, today time.Time) ([]model.RecurringOrder, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM recurring_orders WHERE status = 'active' AND next_order_date <= $1`,
		recurringOrderColumns,
	)
	rows, err := tx.Query(ctx, query, today)
	if err != nil {
		return nil, fmt.Errorf("find due recurring orders: %w", err)
	}
	defer rows.Close()

	var orders []model.RecurringOrder
	for rows.Next() {
		ro, err := scanRecurringOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan due recurring order: %w", err)
		}
		orders = append(orders, *ro)
	}
	return orders, rows.Err()
}

// UpdateAfterCreation updates next_order_date and increments the total_orders_created counter.
func (r *RecurringOrderRepository) UpdateAfterCreation(ctx context.Context, tx pgx.Tx, id uuid.UUID, nextDate time.Time, count int) error {
	_, err := tx.Exec(ctx,
		`UPDATE recurring_orders
		 SET next_order_date = $1,
		     last_order_date = CURRENT_DATE,
		     total_orders_created = $2,
		     updated_at = NOW()
		 WHERE id = $3`,
		nextDate, count, id,
	)
	if err != nil {
		return fmt.Errorf("update recurring order after creation: %w", err)
	}
	return nil
}

// --- Recurring Order Items ---

var recurringOrderItemColumns = `id, tenant_id, recurring_order_id, product_id, sku, product_name, quantity, unit_price, created_at`

func scanRecurringOrderItem(row interface{ Scan(dest ...any) error }) (*model.RecurringOrderItem, error) {
	var item model.RecurringOrderItem
	err := row.Scan(
		&item.ID, &item.TenantID, &item.RecurringOrderID, &item.ProductID,
		&item.SKU, &item.ProductName, &item.Quantity, &item.UnitPrice, &item.CreatedAt,
	)
	return &item, err
}

// ListItems returns all items for the given recurring order.
func (r *RecurringOrderRepository) ListItems(ctx context.Context, tx pgx.Tx, recurringOrderID uuid.UUID) ([]model.RecurringOrderItem, error) {
	query := fmt.Sprintf(
		`SELECT %s FROM recurring_order_items WHERE recurring_order_id = $1 ORDER BY created_at ASC`,
		recurringOrderItemColumns,
	)
	rows, err := tx.Query(ctx, query, recurringOrderID)
	if err != nil {
		return nil, fmt.Errorf("list recurring order items: %w", err)
	}
	defer rows.Close()

	var items []model.RecurringOrderItem
	for rows.Next() {
		item, err := scanRecurringOrderItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recurring order item: %w", err)
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

// CreateItem inserts a new item into a recurring order.
func (r *RecurringOrderRepository) CreateItem(ctx context.Context, tx pgx.Tx, item *model.RecurringOrderItem) error {
	return tx.QueryRow(ctx,
		`INSERT INTO recurring_order_items (id, tenant_id, recurring_order_id, product_id, sku, product_name, quantity, unit_price)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING created_at`,
		item.ID, item.TenantID, item.RecurringOrderID, item.ProductID,
		item.SKU, item.ProductName, item.Quantity, item.UnitPrice,
	).Scan(&item.CreatedAt)
}

// DeleteItemsByRecurringOrderID deletes all items for the given recurring order.
func (r *RecurringOrderRepository) DeleteItemsByRecurringOrderID(ctx context.Context, tx pgx.Tx, recurringOrderID uuid.UUID) error {
	_, err := tx.Exec(ctx, "DELETE FROM recurring_order_items WHERE recurring_order_id = $1", recurringOrderID)
	if err != nil {
		return fmt.Errorf("delete recurring order items: %w", err)
	}
	return nil
}
