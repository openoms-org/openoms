package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ReturnRepository handles persistence for return requests.
type ReturnRepository struct{}

// NewReturnRepository creates a new ReturnRepository.
func NewReturnRepository() *ReturnRepository {
	return &ReturnRepository{}
}

// returnSelectColumns is the canonical list of columns selected from returns.
const returnSelectColumns = `id, tenant_id, order_id, status, reason, items, refund_amount, notes,
		        return_token, customer_email, customer_notes,
		        created_at, updated_at`

// scanReturn scans a row into a model.Return using the returnSelectColumns column order.
func scanReturn(row pgx.Row) (model.Return, error) {
	var ret model.Return
	err := row.Scan(
		&ret.ID, &ret.TenantID, &ret.OrderID, &ret.Status, &ret.Reason,
		&ret.Items, &ret.RefundAmount, &ret.Notes,
		&ret.ReturnToken, &ret.CustomerEmail, &ret.CustomerNotes,
		&ret.CreatedAt, &ret.UpdatedAt,
	)
	return ret, err
}

// scanReturns scans multiple rows into a slice of model.Return using the returnSelectColumns column order.
func scanReturns(rows pgx.Rows) ([]model.Return, error) {
	var returns []model.Return
	for rows.Next() {
		ret, err := scanReturn(rows)
		if err != nil {
			return nil, err
		}
		returns = append(returns, ret)
	}
	return returns, rows.Err()
}

// List returns a paginated list of returns matching the filter.
func (r *ReturnRepository) List(ctx context.Context, tx pgx.Tx, filter model.ReturnListFilter) ([]model.Return, int, error) {
	qb := NewQueryBuilder()

	if filter.Status != nil {
		qb.Add("status = $%d", *filter.Status)
	}
	if filter.OrderID != nil {
		qb.Add("order_id = $%d", *filter.OrderID)
	}

	where := qb.WhereClause()

	var total int
	countQuery := "SELECT COUNT(*) FROM returns " + where
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count returns: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":    "created_at",
		"status":        "status",
		"refund_amount": "refund_amount",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	argIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		`SELECT %s
		 FROM returns %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		returnSelectColumns, where, orderByClause, argIdx, argIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list returns: %w", err)
	}
	defer rows.Close()

	returns, err := scanReturns(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan return: %w", err)
	}
	return returns, total, nil
}

// FindByID returns a return by its ID.
func (r *ReturnRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Return, error) {
	ret, err := scanReturn(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM returns WHERE id = $1`, returnSelectColumns), id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find return by id: %w", err)
	}
	return &ret, nil
}

// FindByToken returns a return by its customer-facing token.
func (r *ReturnRepository) FindByToken(ctx context.Context, tx pgx.Tx, token string) (*model.Return, error) {
	ret, err := scanReturn(tx.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM returns WHERE return_token = $1`, returnSelectColumns), token,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find return by token: %w", err)
	}
	return &ret, nil
}

// Create inserts a new return.
func (r *ReturnRepository) Create(ctx context.Context, tx pgx.Tx, ret *model.Return) error {
	return tx.QueryRow(ctx,
		`INSERT INTO returns (
			id, tenant_id, order_id, status, reason, items, refund_amount, notes,
			return_token, customer_email, customer_notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`,
		ret.ID, ret.TenantID, ret.OrderID, ret.Status, ret.Reason,
		ret.Items, ret.RefundAmount, ret.Notes,
		ret.ReturnToken, ret.CustomerEmail, ret.CustomerNotes,
	).Scan(&ret.CreatedAt, &ret.UpdatedAt)
}

// Update applies partial updates to a return.
func (r *ReturnRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateReturnRequest) error {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.Reason != nil {
		setClauses = append(setClauses, fmt.Sprintf("reason = $%d", argIdx))
		args = append(args, *req.Reason)
		argIdx++
	}
	if req.Items != nil {
		setClauses = append(setClauses, fmt.Sprintf("items = $%d", argIdx))
		args = append(args, *req.Items)
		argIdx++
	}
	if req.RefundAmount != nil {
		setClauses = append(setClauses, fmt.Sprintf("refund_amount = $%d", argIdx))
		args = append(args, *req.RefundAmount)
		argIdx++
	}
	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE returns SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update return: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("return not found")
	}
	return nil
}

// UpdateStatus sets the status of a return.
func (r *ReturnRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	ct, err := tx.Exec(ctx,
		"UPDATE returns SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update return status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("return not found")
	}
	return nil
}

// Delete removes a return by its ID.
func (r *ReturnRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM returns WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete return: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("return not found")
	}
	return nil
}
