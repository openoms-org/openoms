package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// OrderRepository handles persistence for orders.
type OrderRepository struct{}

// NewOrderRepository creates a new OrderRepository.
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

// orderSelectColumns is the canonical list of columns selected from orders.
const orderSelectColumns = `id, tenant_id, external_id, source, integration_id, status,
		        customer_name, customer_email, customer_phone,
		        shipping_address, billing_address, items,
		        total_amount, currency, notes, metadata, tags,
		        ordered_at, shipped_at, delivered_at,
		        delivery_method, pickup_point_id,
		        payment_status, payment_method, paid_at, customer_id, merged_into, split_from,
		        internal_notes, priority, created_at, updated_at`

// scanOrder scans a row into a model.Order using the orderSelectColumns column order.
func scanOrder(row pgx.Row) (model.Order, error) {
	var o model.Order
	err := row.Scan(
		&o.ID, &o.TenantID, &o.ExternalID, &o.Source, &o.IntegrationID, &o.Status,
		&o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.ShippingAddress, &o.BillingAddress, &o.Items,
		&o.TotalAmount, &o.Currency, &o.Notes, &o.Metadata, &o.Tags,
		&o.OrderedAt, &o.ShippedAt, &o.DeliveredAt,
		&o.DeliveryMethod, &o.PickupPointID,
		&o.PaymentStatus, &o.PaymentMethod, &o.PaidAt, &o.CustomerID, &o.MergedInto, &o.SplitFrom,
		&o.InternalNotes, &o.Priority, &o.CreatedAt, &o.UpdatedAt,
	)
	return o, err
}

// List returns a paginated list of orders matching the filter.
func (r *OrderRepository) List(ctx context.Context, tx pgx.Tx, filter model.OrderListFilter) ([]model.Order, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.Source != nil {
		qb.Add("source = $%d", *filter.Source)
	}
	if filter.Search != nil {
		qb.AddMultiRef("(customer_name ILIKE $%d OR customer_email ILIKE $%d OR customer_phone ILIKE $%d)", 3, "%"+*filter.Search+"%")
	}
	if filter.PaymentStatus != nil {
		qb.Add("payment_status = $%d", *filter.PaymentStatus)
	}
	if filter.Tag != nil {
		qb.Add("tags @> ARRAY[$%d]::text[]", *filter.Tag)
	}
	if filter.Priority != nil {
		qb.Add("priority = $%d", *filter.Priority)
	}

	where := qb.WhereClause()

	var total int
	countQuery := "SELECT COUNT(*) FROM orders " + where
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"ordered_at":     "ordered_at",
		"customer_name":  "customer_name",
		"total_amount":   "total_amount",
		"status":         "status",
		"source":         "source",
		"payment_status": "payment_status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s
		 FROM orders %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		orderSelectColumns, where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

// FindByIDs returns orders by their IDs as a map keyed by order ID.
func (r *OrderRepository) FindByIDs(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) (map[uuid.UUID]*model.Order, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*model.Order), nil
	}
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM orders WHERE id = ANY($1)`, orderSelectColumns), ids,
	)
	if err != nil {
		return nil, fmt.Errorf("find orders by ids: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]*model.Order, len(ids))
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("find orders by ids: scan: %w", err)
		}
		result[o.ID] = &o
	}
	return result, rows.Err()
}

// FindByID returns an order by its ID.
func (r *OrderRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error) {
	o, err := scanOrder(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM orders WHERE id = $1`, orderSelectColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find order by id: %w", err)
	}
	return &o, nil
}

// Create inserts a new order.
func (r *OrderRepository) Create(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	_, err := r.insert(ctx, tx, order, "")
	return err
}

// CreateIfExternalIDNotExists inserts an order atomically unless the tenant/source/external_id
// tuple is already present. It returns false,nil for duplicate external orders.
func (r *OrderRepository) CreateIfExternalIDNotExists(ctx context.Context, tx pgx.Tx, order *model.Order) (bool, error) {
	return r.insert(ctx, tx, order, `ON CONFLICT (tenant_id, source, external_id)
		WHERE external_id IS NOT NULL AND external_id <> '' DO NOTHING`)
}

func (r *OrderRepository) insert(ctx context.Context, tx pgx.Tx, order *model.Order, conflictClause string) (bool, error) {
	tags := order.Tags
	if tags == nil {
		tags = []string{}
	}

	query := `INSERT INTO orders (
			id, tenant_id, external_id, source, integration_id, status,
			customer_name, customer_email, customer_phone,
			shipping_address, billing_address, items,
			total_amount, currency, notes, metadata, tags, ordered_at,
			delivery_method, pickup_point_id,
			payment_status, payment_method,
			internal_notes, priority, customer_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
		` + conflictClause + `
		RETURNING created_at, updated_at`

	err := tx.QueryRow(ctx, query,
		order.ID, order.TenantID, order.ExternalID, order.Source, order.IntegrationID, order.Status,
		order.CustomerName, order.CustomerEmail, order.CustomerPhone,
		order.ShippingAddress, order.BillingAddress, order.Items,
		order.TotalAmount, order.Currency, order.Notes, order.Metadata, tags, order.OrderedAt,
		order.DeliveryMethod, order.PickupPointID,
		order.PaymentStatus, order.PaymentMethod,
		order.InternalNotes, order.Priority, order.CustomerID,
	).Scan(&order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if conflictClause != "" && err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("insert order: %w", err)
	}
	return true, nil
}

// Update applies partial updates to an order.
func (r *OrderRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateOrderRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "external_id", req.ExternalID)
	SetPtr(ub, "customer_name", req.CustomerName)
	SetPtr(ub, "customer_email", req.CustomerEmail)
	SetPtr(ub, "customer_phone", req.CustomerPhone)
	if req.ShippingAddress != nil {
		ub.Set("shipping_address", req.ShippingAddress)
	}
	if req.BillingAddress != nil {
		ub.Set("billing_address", req.BillingAddress)
	}
	if req.Items != nil {
		ub.Set("items", req.Items)
	}
	SetPtr(ub, "total_amount", req.TotalAmount)
	SetPtr(ub, "currency", req.Currency)
	SetPtr(ub, "notes", req.Notes)
	if req.Metadata != nil {
		ub.Set("metadata", req.Metadata)
	}
	SetPtr(ub, "tags", req.Tags)
	SetPtr(ub, "delivery_method", req.DeliveryMethod)
	SetPtr(ub, "pickup_point_id", req.PickupPointID)
	SetPtr(ub, "payment_status", req.PaymentStatus)
	SetPtr(ub, "payment_method", req.PaymentMethod)
	SetPtr(ub, "paid_at", req.PaidAt)
	SetPtr(ub, "internal_notes", req.InternalNotes)
	SetPtr(ub, "priority", req.Priority)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("updated_at = NOW()")
	args := append(ub.Args(), id)

	query := fmt.Sprintf("UPDATE orders SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

// UpdateStatus sets the status and optional timestamps of an order.
func (r *OrderRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, shippedAt, deliveredAt *time.Time) error {
	ct, err := tx.Exec(ctx,
		`UPDATE orders SET status = $1, shipped_at = COALESCE($2, shipped_at),
		 delivered_at = COALESCE($3, delivered_at), updated_at = NOW()
		 WHERE id = $4`,
		status, shippedAt, deliveredAt, id,
	)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

// FindByExternalID returns the order matching the given source and external ID.
func (r *OrderRepository) FindByExternalID(ctx context.Context, tx pgx.Tx, source, externalID string) (*model.Order, error) {
	o, err := scanOrder(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM orders WHERE source = $1 AND external_id = $2`, orderSelectColumns), source, externalID,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find order by external id: %w", err)
	}
	return &o, nil
}

// Delete removes an order by its ID.
func (r *OrderRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM orders WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

// CountThisMonth returns the number of orders created in the current calendar month.
func (r *OrderRepository) CountThisMonth(ctx context.Context, tx pgx.Tx) (int, error) {
	var count int
	err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM orders WHERE created_at >= date_trunc('month', now())`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count orders this month: %w", err)
	}
	return count, nil
}
