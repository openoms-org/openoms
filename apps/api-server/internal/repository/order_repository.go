package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) List(ctx context.Context, tx pgx.Tx, filter model.OrderListFilter) ([]model.Order, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Source != nil {
		where += fmt.Sprintf(" AND source = $%d", argIdx)
		args = append(args, *filter.Source)
		argIdx++
	}
	if filter.Search != nil {
		where += fmt.Sprintf(" AND (customer_name ILIKE $%d OR customer_email ILIKE $%d OR customer_phone ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}
	if filter.PaymentStatus != nil {
		where += fmt.Sprintf(" AND payment_status = $%d", argIdx)
		args = append(args, *filter.PaymentStatus)
		argIdx++
	}
	if filter.Tag != nil {
		where += fmt.Sprintf(" AND tags @> ARRAY[$%d]::text[]", argIdx)
		args = append(args, *filter.Tag)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM orders " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"customer_name":  "customer_name",
		"total_amount":   "total_amount",
		"status":         "status",
		"source":         "source",
		"payment_status": "payment_status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, external_id, source, integration_id, status,
		        customer_name, customer_email, customer_phone,
		        shipping_address, billing_address, items,
		        total_amount, currency, notes, metadata, tags,
		        ordered_at, shipped_at, delivered_at,
		        delivery_method, pickup_point_id,
		        payment_status, payment_method, paid_at, created_at, updated_at
		 FROM orders %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.ExternalID, &o.Source, &o.IntegrationID, &o.Status,
			&o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
			&o.ShippingAddress, &o.BillingAddress, &o.Items,
			&o.TotalAmount, &o.Currency, &o.Notes, &o.Metadata, &o.Tags,
			&o.OrderedAt, &o.ShippedAt, &o.DeliveredAt,
			&o.DeliveryMethod, &o.PickupPointID,
			&o.PaymentStatus, &o.PaymentMethod, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

func (r *OrderRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, external_id, source, integration_id, status,
		        customer_name, customer_email, customer_phone,
		        shipping_address, billing_address, items,
		        total_amount, currency, notes, metadata, tags,
		        ordered_at, shipped_at, delivered_at,
		        delivery_method, pickup_point_id,
		        payment_status, payment_method, paid_at, created_at, updated_at
		 FROM orders WHERE id = $1`, id,
	).Scan(
		&o.ID, &o.TenantID, &o.ExternalID, &o.Source, &o.IntegrationID, &o.Status,
		&o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.ShippingAddress, &o.BillingAddress, &o.Items,
		&o.TotalAmount, &o.Currency, &o.Notes, &o.Metadata, &o.Tags,
		&o.OrderedAt, &o.ShippedAt, &o.DeliveredAt,
		&o.DeliveryMethod, &o.PickupPointID,
		&o.PaymentStatus, &o.PaymentMethod, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find order by id: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) Create(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	tags := order.Tags
	if tags == nil {
		tags = []string{}
	}
	return tx.QueryRow(ctx,
		`INSERT INTO orders (
			id, tenant_id, external_id, source, integration_id, status,
			customer_name, customer_email, customer_phone,
			shipping_address, billing_address, items,
			total_amount, currency, notes, metadata, tags, ordered_at,
			delivery_method, pickup_point_id,
			payment_status, payment_method
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING created_at, updated_at`,
		order.ID, order.TenantID, order.ExternalID, order.Source, order.IntegrationID, order.Status,
		order.CustomerName, order.CustomerEmail, order.CustomerPhone,
		order.ShippingAddress, order.BillingAddress, order.Items,
		order.TotalAmount, order.Currency, order.Notes, order.Metadata, tags, order.OrderedAt,
		order.DeliveryMethod, order.PickupPointID,
		order.PaymentStatus, order.PaymentMethod,
	).Scan(&order.CreatedAt, &order.UpdatedAt)
}

func (r *OrderRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateOrderRequest) error {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.ExternalID != nil {
		setClauses = append(setClauses, fmt.Sprintf("external_id = $%d", argIdx))
		args = append(args, *req.ExternalID)
		argIdx++
	}
	if req.CustomerName != nil {
		setClauses = append(setClauses, fmt.Sprintf("customer_name = $%d", argIdx))
		args = append(args, *req.CustomerName)
		argIdx++
	}
	if req.CustomerEmail != nil {
		setClauses = append(setClauses, fmt.Sprintf("customer_email = $%d", argIdx))
		args = append(args, *req.CustomerEmail)
		argIdx++
	}
	if req.CustomerPhone != nil {
		setClauses = append(setClauses, fmt.Sprintf("customer_phone = $%d", argIdx))
		args = append(args, *req.CustomerPhone)
		argIdx++
	}
	if req.ShippingAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("shipping_address = $%d", argIdx))
		args = append(args, req.ShippingAddress)
		argIdx++
	}
	if req.BillingAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("billing_address = $%d", argIdx))
		args = append(args, req.BillingAddress)
		argIdx++
	}
	if req.Items != nil {
		setClauses = append(setClauses, fmt.Sprintf("items = $%d", argIdx))
		args = append(args, req.Items)
		argIdx++
	}
	if req.TotalAmount != nil {
		setClauses = append(setClauses, fmt.Sprintf("total_amount = $%d", argIdx))
		args = append(args, *req.TotalAmount)
		argIdx++
	}
	if req.Currency != nil {
		setClauses = append(setClauses, fmt.Sprintf("currency = $%d", argIdx))
		args = append(args, *req.Currency)
		argIdx++
	}
	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.Metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
		args = append(args, req.Metadata)
		argIdx++
	}
	if req.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *req.Tags)
		argIdx++
	}
	if req.DeliveryMethod != nil {
		setClauses = append(setClauses, fmt.Sprintf("delivery_method = $%d", argIdx))
		args = append(args, *req.DeliveryMethod)
		argIdx++
	}
	if req.PickupPointID != nil {
		setClauses = append(setClauses, fmt.Sprintf("pickup_point_id = $%d", argIdx))
		args = append(args, *req.PickupPointID)
		argIdx++
	}
	if req.PaymentStatus != nil {
		setClauses = append(setClauses, fmt.Sprintf("payment_status = $%d", argIdx))
		args = append(args, *req.PaymentStatus)
		argIdx++
	}
	if req.PaymentMethod != nil {
		setClauses = append(setClauses, fmt.Sprintf("payment_method = $%d", argIdx))
		args = append(args, *req.PaymentMethod)
		argIdx++
	}
	if req.PaidAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("paid_at = $%d", argIdx))
		args = append(args, *req.PaidAt)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE orders SET %s WHERE id = $%d",
		JoinStrings(setClauses, ", "), argIdx)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

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

func (r *OrderRepository) FindByExternalID(ctx context.Context, tx pgx.Tx, source, externalID string) (*model.Order, error) {
	var o model.Order
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, external_id, source, integration_id, status,
		        customer_name, customer_email, customer_phone,
		        shipping_address, billing_address, items,
		        total_amount, currency, notes, metadata, tags,
		        ordered_at, shipped_at, delivered_at,
		        delivery_method, pickup_point_id,
		        payment_status, payment_method, paid_at, created_at, updated_at
		 FROM orders WHERE source = $1 AND metadata->>'external_id' = $2`, source, externalID,
	).Scan(
		&o.ID, &o.TenantID, &o.ExternalID, &o.Source, &o.IntegrationID, &o.Status,
		&o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.ShippingAddress, &o.BillingAddress, &o.Items,
		&o.TotalAmount, &o.Currency, &o.Notes, &o.Metadata, &o.Tags,
		&o.OrderedAt, &o.ShippedAt, &o.DeliveredAt,
		&o.DeliveryMethod, &o.PickupPointID,
		&o.PaymentStatus, &o.PaymentMethod, &o.PaidAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find order by external id: %w", err)
	}
	return &o, nil
}

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

