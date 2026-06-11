package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// CustomerSegmentRepository handles persistence for customer segments.
type CustomerSegmentRepository struct{}

// NewCustomerSegmentRepository creates a new CustomerSegmentRepository.
func NewCustomerSegmentRepository() *CustomerSegmentRepository {
	return &CustomerSegmentRepository{}
}

var segmentColumns = `id, tenant_id, name, description, color, segment_type, rules, customer_count, created_at, updated_at`

func scanSegment(row interface{ Scan(dest ...any) error }) (*model.CustomerSegment, error) {
	var s model.CustomerSegment
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.Description, &s.Color,
		&s.SegmentType, &s.Rules, &s.CustomerCount,
		&s.CreatedAt, &s.UpdatedAt,
	)
	return &s, err
}

// List returns a paginated list of customer segments.
func (r *CustomerSegmentRepository) List(ctx context.Context, tx pgx.Tx, filter model.SegmentListFilter) ([]model.CustomerSegment, int, error) {
	var total int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM customer_segments").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count segments: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":     "created_at",
		"name":           "name",
		"customer_count": "customer_count",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT %s FROM customer_segments %s LIMIT $1 OFFSET $2`,
		segmentColumns, orderByClause,
	)

	rows, err := tx.Query(ctx, query, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list segments: %w", err)
	}
	defer rows.Close()

	var segments []model.CustomerSegment
	for rows.Next() {
		s, err := scanSegment(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan segment: %w", err)
		}
		segments = append(segments, *s)
	}
	return segments, total, rows.Err()
}

// FindByID returns a customer segment by its ID.
func (r *CustomerSegmentRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.CustomerSegment, error) {
	s, err := scanSegment(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM customer_segments WHERE id = $1", segmentColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find segment by id: %w", err)
	}
	return s, nil
}

// Create inserts a new customer segment.
func (r *CustomerSegmentRepository) Create(ctx context.Context, tx pgx.Tx, segment *model.CustomerSegment) error {
	return tx.QueryRow(ctx,
		`INSERT INTO customer_segments (id, tenant_id, name, description, color, segment_type, rules)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		segment.ID, segment.TenantID, segment.Name, segment.Description,
		segment.Color, segment.SegmentType, segment.Rules,
	).Scan(&segment.CreatedAt, &segment.UpdatedAt)
}

// Update applies partial updates to a customer segment.
func (r *CustomerSegmentRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateSegmentRequest) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Color != nil {
		setClauses = append(setClauses, fmt.Sprintf("color = $%d", argIdx))
		args = append(args, *req.Color)
		argIdx++
	}
	if req.Rules != nil {
		setClauses = append(setClauses, fmt.Sprintf("rules = $%d", argIdx))
		args = append(args, req.Rules)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE customer_segments SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update segment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("segment not found")
	}
	return nil
}

// Delete removes a customer segment by its ID.
func (r *CustomerSegmentRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM customer_segments WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete segment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("segment not found")
	}
	return nil
}

// AddMember adds a customer to a segment.
func (r *CustomerSegmentRepository) AddMember(ctx context.Context, tx pgx.Tx, tenantID, segmentID, customerID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO customer_segment_members (tenant_id, segment_id, customer_id)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		tenantID, segmentID, customerID,
	)
	if err != nil {
		return fmt.Errorf("add segment member: %w", err)
	}
	return nil
}

// RemoveMember removes a customer from a segment.
func (r *CustomerSegmentRepository) RemoveMember(ctx context.Context, tx pgx.Tx, segmentID, customerID uuid.UUID) error {
	ct, err := tx.Exec(ctx,
		`DELETE FROM customer_segment_members WHERE segment_id = $1 AND customer_id = $2`,
		segmentID, customerID,
	)
	if err != nil {
		return fmt.Errorf("remove segment member: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

// ListMembers lists all members of a segment with customer details.
func (r *CustomerSegmentRepository) ListMembers(ctx context.Context, tx pgx.Tx, segmentID uuid.UUID, limit, offset int) ([]model.SegmentMember, int, error) {
	var total int
	if err := tx.QueryRow(ctx,
		"SELECT COUNT(*) FROM customer_segment_members WHERE segment_id = $1", segmentID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count segment members: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT m.tenant_id, m.segment_id, m.customer_id, m.added_at,
		        c.name, c.email, c.total_orders, c.total_spent
		 FROM customer_segment_members m
		 JOIN customers c ON c.id = m.customer_id
		 WHERE m.segment_id = $1
		 ORDER BY m.added_at DESC
		 LIMIT $2 OFFSET $3`,
		segmentID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list segment members: %w", err)
	}
	defer rows.Close()

	var members []model.SegmentMember
	for rows.Next() {
		var m model.SegmentMember
		if err := rows.Scan(
			&m.TenantID, &m.SegmentID, &m.CustomerID, &m.AddedAt,
			&m.CustomerName, &m.CustomerEmail, &m.TotalOrders, &m.TotalSpent,
		); err != nil {
			return nil, 0, fmt.Errorf("scan segment member: %w", err)
		}
		members = append(members, m)
	}
	return members, total, rows.Err()
}

// UpdateCustomerCount updates the cached customer_count on a segment.
func (r *CustomerSegmentRepository) UpdateCustomerCount(ctx context.Context, tx pgx.Tx, segmentID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE customer_segments SET customer_count = (
			SELECT COUNT(*) FROM customer_segment_members WHERE segment_id = $1
		), updated_at = NOW() WHERE id = $1`,
		segmentID,
	)
	return err
}

// GetCustomerSegments returns all segments a customer belongs to.
func (r *CustomerSegmentRepository) GetCustomerSegments(ctx context.Context, tx pgx.Tx, customerID uuid.UUID) ([]model.CustomerSegment, error) {
	rows, err := tx.Query(ctx,
		fmt.Sprintf(`SELECT s.%s FROM customer_segments s
		 JOIN customer_segment_members m ON m.segment_id = s.id
		 WHERE m.customer_id = $1
		 ORDER BY s.name`, segmentColumns),
		customerID,
	)
	if err != nil {
		return nil, fmt.Errorf("get customer segments: %w", err)
	}
	defer rows.Close()

	var segments []model.CustomerSegment
	for rows.Next() {
		s, err := scanSegment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan customer segment: %w", err)
		}
		segments = append(segments, *s)
	}
	return segments, rows.Err()
}

// ClearSegmentMembers removes all members from a segment (used for rfm_auto refresh).
func (r *CustomerSegmentRepository) ClearSegmentMembers(ctx context.Context, tx pgx.Tx, segmentID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		"DELETE FROM customer_segment_members WHERE segment_id = $1", segmentID,
	)
	return err
}

// GetRFMData returns raw RFM data for all customers with orders.
func (r *CustomerSegmentRepository) GetRFMData(ctx context.Context, tx pgx.Tx) ([]model.CustomerRFM, error) {
	rows, err := tx.Query(ctx,
		`SELECT c.id, c.name, c.email,
		        COALESCE(EXTRACT(DAY FROM NOW() - MAX(o.created_at))::int, 9999) AS days_since,
		        COUNT(o.id)::int AS order_count,
		        COALESCE(SUM(o.total_amount), 0)::float8 AS total_spent
		 FROM customers c
		 LEFT JOIN orders o ON o.customer_id = c.id
		 GROUP BY c.id, c.name, c.email
		 HAVING COUNT(o.id) > 0
		 ORDER BY c.name`,
	)
	if err != nil {
		return nil, fmt.Errorf("get rfm data: %w", err)
	}
	defer rows.Close()

	var results []model.CustomerRFM
	for rows.Next() {
		var r model.CustomerRFM
		if err := rows.Scan(&r.CustomerID, &r.CustomerName, &r.CustomerEmail,
			&r.DaysSince, &r.OrderCount, &r.TotalSpent); err != nil {
			return nil, fmt.Errorf("scan rfm data: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// FindByTypeAndName finds a segment by type and name (for RFM auto-segments).
func (r *CustomerSegmentRepository) FindByTypeAndName(ctx context.Context, tx pgx.Tx, segmentType, name string) (*model.CustomerSegment, error) {
	s, err := scanSegment(tx.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM customer_segments WHERE segment_type = $1 AND name = $2", segmentColumns),
		segmentType, name,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find segment by type and name: %w", err)
	}
	return s, nil
}
