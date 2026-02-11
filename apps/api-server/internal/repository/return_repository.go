package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type ReturnRepository struct{}

func NewReturnRepository() *ReturnRepository {
	return &ReturnRepository{}
}

func (r *ReturnRepository) List(ctx context.Context, tx pgx.Tx, filter model.ReturnListFilter) ([]model.Return, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.OrderID != nil {
		where += fmt.Sprintf(" AND order_id = $%d", argIdx)
		args = append(args, *filter.OrderID)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM returns " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count returns: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":    "created_at",
		"status":        "status",
		"refund_amount": "refund_amount",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, order_id, status, reason, items, refund_amount, notes,
		        return_token, customer_email, customer_notes,
		        created_at, updated_at
		 FROM returns %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list returns: %w", err)
	}
	defer rows.Close()

	var returns []model.Return
	for rows.Next() {
		var ret model.Return
		if err := rows.Scan(
			&ret.ID, &ret.TenantID, &ret.OrderID, &ret.Status, &ret.Reason,
			&ret.Items, &ret.RefundAmount, &ret.Notes,
			&ret.ReturnToken, &ret.CustomerEmail, &ret.CustomerNotes,
			&ret.CreatedAt, &ret.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan return: %w", err)
		}
		returns = append(returns, ret)
	}
	return returns, total, rows.Err()
}

func (r *ReturnRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Return, error) {
	var ret model.Return
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, status, reason, items, refund_amount, notes,
		        return_token, customer_email, customer_notes,
		        created_at, updated_at
		 FROM returns WHERE id = $1`, id,
	).Scan(
		&ret.ID, &ret.TenantID, &ret.OrderID, &ret.Status, &ret.Reason,
		&ret.Items, &ret.RefundAmount, &ret.Notes,
		&ret.ReturnToken, &ret.CustomerEmail, &ret.CustomerNotes,
		&ret.CreatedAt, &ret.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find return by id: %w", err)
	}
	return &ret, nil
}

func (r *ReturnRepository) FindByToken(ctx context.Context, tx pgx.Tx, token string) (*model.Return, error) {
	var ret model.Return
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, status, reason, items, refund_amount, notes,
		        return_token, customer_email, customer_notes,
		        created_at, updated_at
		 FROM returns WHERE return_token = $1`, token,
	).Scan(
		&ret.ID, &ret.TenantID, &ret.OrderID, &ret.Status, &ret.Reason,
		&ret.Items, &ret.RefundAmount, &ret.Notes,
		&ret.ReturnToken, &ret.CustomerEmail, &ret.CustomerNotes,
		&ret.CreatedAt, &ret.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find return by token: %w", err)
	}
	return &ret, nil
}

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
