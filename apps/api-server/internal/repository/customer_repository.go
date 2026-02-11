package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type CustomerRepository struct{}

func NewCustomerRepository() *CustomerRepository {
	return &CustomerRepository{}
}

var customerColumns = `id, tenant_id, email, phone, name, company_name, nip,
	default_shipping_address, default_billing_address, tags, notes,
	total_orders, total_spent, price_list_id, created_at, updated_at`

func scanCustomer(row interface{ Scan(dest ...any) error }) (*model.Customer, error) {
	var c model.Customer
	err := row.Scan(
		&c.ID, &c.TenantID, &c.Email, &c.Phone, &c.Name, &c.CompanyName, &c.NIP,
		&c.DefaultShippingAddress, &c.DefaultBillingAddress, &c.Tags, &c.Notes,
		&c.TotalOrders, &c.TotalSpent, &c.PriceListID, &c.CreatedAt, &c.UpdatedAt,
	)
	return &c, err
}

func (r *CustomerRepository) List(ctx context.Context, tx pgx.Tx, filter model.CustomerListFilter) ([]model.Customer, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(name ILIKE $%d OR email ILIKE $%d OR phone ILIKE $%d)",
			argIdx, argIdx, argIdx,
		))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}
	if filter.Tags != nil && *filter.Tags != "" {
		conditions = append(conditions, fmt.Sprintf("tags @> ARRAY[$%d]::text[]", argIdx))
		args = append(args, *filter.Tags)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM customers %s", where)
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":   "created_at",
		"name":         "name",
		"email":        "email",
		"total_orders": "total_orders",
		"total_spent":  "total_spent",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT %s FROM customers %s %s LIMIT $%d OFFSET $%d`,
		customerColumns, where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []model.Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, *c)
	}
	return customers, total, rows.Err()
}

func (r *CustomerRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Customer, error) {
	c, err := scanCustomer(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM customers WHERE id = $1", customerColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find customer by id: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) FindByEmail(ctx context.Context, tx pgx.Tx, email string) (*model.Customer, error) {
	c, err := scanCustomer(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM customers WHERE email = $1 LIMIT 1", customerColumns), email,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find customer by email: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) Create(ctx context.Context, tx pgx.Tx, customer *model.Customer) error {
	tags := customer.Tags
	if tags == nil {
		tags = []string{}
	}
	return tx.QueryRow(ctx,
		`INSERT INTO customers (id, tenant_id, email, phone, name, company_name, nip,
		 default_shipping_address, default_billing_address, tags, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING created_at, updated_at`,
		customer.ID, customer.TenantID, customer.Email, customer.Phone,
		customer.Name, customer.CompanyName, customer.NIP,
		customer.DefaultShippingAddress, customer.DefaultBillingAddress,
		tags, customer.Notes,
	).Scan(&customer.CreatedAt, &customer.UpdatedAt)
}

func (r *CustomerRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateCustomerRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *req.Email)
		argIdx++
	}
	if req.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *req.Phone)
		argIdx++
	}
	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.CompanyName != nil {
		setClauses = append(setClauses, fmt.Sprintf("company_name = $%d", argIdx))
		args = append(args, *req.CompanyName)
		argIdx++
	}
	if req.NIP != nil {
		setClauses = append(setClauses, fmt.Sprintf("nip = $%d", argIdx))
		args = append(args, *req.NIP)
		argIdx++
	}
	if req.DefaultShippingAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_shipping_address = $%d", argIdx))
		args = append(args, req.DefaultShippingAddress)
		argIdx++
	}
	if req.DefaultBillingAddress != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_billing_address = $%d", argIdx))
		args = append(args, req.DefaultBillingAddress)
		argIdx++
	}
	if req.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *req.Tags)
		argIdx++
	}
	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}
	if req.PriceListID != nil {
		setClauses = append(setClauses, fmt.Sprintf("price_list_id = $%d", argIdx))
		args = append(args, *req.PriceListID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE customers SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update customer: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *CustomerRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM customers WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *CustomerRepository) IncrementOrderStats(ctx context.Context, tx pgx.Tx, id uuid.UUID, amount float64) error {
	_, err := tx.Exec(ctx,
		`UPDATE customers SET total_orders = total_orders + 1, total_spent = total_spent + $1, updated_at = NOW() WHERE id = $2`,
		amount, id,
	)
	if err != nil {
		return fmt.Errorf("increment customer order stats: %w", err)
	}
	return nil
}

func (r *CustomerRepository) ListOrdersByCustomerID(ctx context.Context, tx pgx.Tx, customerID uuid.UUID, filter model.OrderListFilter) ([]model.Order, int, error) {
	where := "WHERE customer_id = $1"
	args := []any{customerID}
	argIdx := 2

	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM orders " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count customer orders: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":   "created_at",
		"total_amount": "total_amount",
		"status":       "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, external_id, source, integration_id, status,
		        customer_name, customer_email, customer_phone,
		        shipping_address, billing_address, items,
		        total_amount, currency, notes, metadata, tags,
		        ordered_at, shipped_at, delivered_at,
		        delivery_method, pickup_point_id,
		        payment_status, payment_method, paid_at, customer_id, created_at, updated_at
		 FROM orders %s %s LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list customer orders: %w", err)
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
			&o.PaymentStatus, &o.PaymentMethod, &o.PaidAt, &o.CustomerID, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan customer order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}
