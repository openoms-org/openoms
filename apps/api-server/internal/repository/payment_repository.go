package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type PaymentRepository struct{}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{}
}

// --- Settlements ---

func scanSettlement(row pgx.Row) (model.PaymentSettlement, error) {
	var s model.PaymentSettlement
	var settlementDate time.Time
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Provider, &s.SettlementID,
		&settlementDate, &s.TotalAmount, &s.FeeAmount, &s.NetAmount,
		&s.Currency, &s.Status, &s.Notes,
		&s.ImportedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == nil {
		s.SettlementDate = settlementDate.Format("2006-01-02")
	}
	return s, err
}

const settlementSelectColumns = `id, tenant_id, provider, settlement_id,
	settlement_date, total_amount, fee_amount, net_amount,
	currency, status, notes, imported_at, created_at, updated_at`

func (r *PaymentRepository) ListSettlements(ctx context.Context, tx pgx.Tx, filter model.SettlementListFilter) ([]model.PaymentSettlement, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.Provider != nil {
		where += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, *filter.Provider)
		argIdx++
	}
	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.DateFrom != nil {
		where += fmt.Sprintf(" AND settlement_date >= $%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		where += fmt.Sprintf(" AND settlement_date <= $%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM payment_settlements " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count payment settlements: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":      "created_at",
		"settlement_date": "settlement_date",
		"total_amount":    "total_amount",
		"provider":        "provider",
		"status":          "status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		"SELECT %s FROM payment_settlements %s %s LIMIT $%d OFFSET $%d",
		settlementSelectColumns, where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list payment settlements: %w", err)
	}
	defer rows.Close()

	var settlements []model.PaymentSettlement
	for rows.Next() {
		s, err := scanSettlement(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan payment settlement: %w", err)
		}
		settlements = append(settlements, s)
	}

	return settlements, total, rows.Err()
}

func (r *PaymentRepository) FindSettlementByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PaymentSettlement, error) {
	query := fmt.Sprintf("SELECT %s FROM payment_settlements WHERE id = $1", settlementSelectColumns)
	s, err := scanSettlement(tx.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find settlement by id: %w", err)
	}
	return &s, nil
}

func (r *PaymentRepository) CreateSettlement(ctx context.Context, tx pgx.Tx, s *model.PaymentSettlement) error {
	query := `INSERT INTO payment_settlements
		(id, tenant_id, provider, settlement_id, settlement_date, total_amount, fee_amount, net_amount, currency, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := tx.Exec(ctx, query,
		s.ID, s.TenantID, s.Provider, s.SettlementID,
		s.SettlementDate, s.TotalAmount, s.FeeAmount, s.NetAmount,
		s.Currency, s.Status, s.Notes,
	)
	if err != nil {
		return fmt.Errorf("create payment settlement: %w", err)
	}
	return nil
}

func (r *PaymentRepository) UpdateSettlementStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	query := `UPDATE payment_settlements SET status = $1, updated_at = now() WHERE id = $2`
	_, err := tx.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("update settlement status: %w", err)
	}
	return nil
}

// --- Transactions ---

const transactionSelectColumns = `id, tenant_id, settlement_id, order_id,
	provider, external_transaction_id, amount, fee, net_amount,
	currency, transaction_type, transaction_date,
	match_status, match_notes, created_at`

func scanTransaction(row pgx.Row) (model.PaymentTransaction, error) {
	var t model.PaymentTransaction
	err := row.Scan(
		&t.ID, &t.TenantID, &t.SettlementID, &t.OrderID,
		&t.Provider, &t.ExternalTransactionID,
		&t.Amount, &t.Fee, &t.NetAmount,
		&t.Currency, &t.TransactionType, &t.TransactionDate,
		&t.MatchStatus, &t.MatchNotes, &t.CreatedAt,
	)
	return t, err
}

func (r *PaymentRepository) ListTransactions(ctx context.Context, tx pgx.Tx, filter model.TransactionListFilter) ([]model.PaymentTransaction, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.SettlementID != nil {
		where += fmt.Sprintf(" AND settlement_id = $%d", argIdx)
		args = append(args, *filter.SettlementID)
		argIdx++
	}
	if filter.MatchStatus != nil {
		where += fmt.Sprintf(" AND match_status = $%d", argIdx)
		args = append(args, *filter.MatchStatus)
		argIdx++
	}
	if filter.Provider != nil {
		where += fmt.Sprintf(" AND provider = $%d", argIdx)
		args = append(args, *filter.Provider)
		argIdx++
	}
	if filter.TransactionType != nil {
		where += fmt.Sprintf(" AND transaction_type = $%d", argIdx)
		args = append(args, *filter.TransactionType)
		argIdx++
	}
	if filter.DateFrom != nil {
		where += fmt.Sprintf(" AND transaction_date >= $%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		where += fmt.Sprintf(" AND transaction_date <= $%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM payment_transactions " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count payment transactions: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":       "created_at",
		"transaction_date": "transaction_date",
		"amount":           "amount",
		"match_status":     "match_status",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		"SELECT %s FROM payment_transactions %s %s LIMIT $%d OFFSET $%d",
		transactionSelectColumns, where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list payment transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.PaymentTransaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan payment transaction: %w", err)
		}
		transactions = append(transactions, t)
	}

	return transactions, total, rows.Err()
}

func (r *PaymentRepository) FindTransactionByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PaymentTransaction, error) {
	query := fmt.Sprintf("SELECT %s FROM payment_transactions WHERE id = $1", transactionSelectColumns)
	t, err := scanTransaction(tx.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find transaction by id: %w", err)
	}
	return &t, nil
}

func (r *PaymentRepository) FindTransactionByExternalID(ctx context.Context, tx pgx.Tx, provider, externalID string) (*model.PaymentTransaction, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM payment_transactions WHERE provider = $1 AND external_transaction_id = $2",
		transactionSelectColumns,
	)
	t, err := scanTransaction(tx.QueryRow(ctx, query, provider, externalID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find transaction by external id: %w", err)
	}
	return &t, nil
}

func (r *PaymentRepository) FindTransactionsBySettlement(ctx context.Context, tx pgx.Tx, settlementID uuid.UUID) ([]model.PaymentTransaction, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM payment_transactions WHERE settlement_id = $1 ORDER BY transaction_date ASC",
		transactionSelectColumns,
	)
	rows, err := tx.Query(ctx, query, settlementID)
	if err != nil {
		return nil, fmt.Errorf("find transactions by settlement: %w", err)
	}
	defer rows.Close()

	var transactions []model.PaymentTransaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

func (r *PaymentRepository) FindUnmatchedTransactions(ctx context.Context, tx pgx.Tx) ([]model.PaymentTransaction, error) {
	query := fmt.Sprintf(
		"SELECT %s FROM payment_transactions WHERE match_status = 'unmatched' ORDER BY transaction_date ASC",
		transactionSelectColumns,
	)
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("find unmatched transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.PaymentTransaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

func (r *PaymentRepository) CreateTransaction(ctx context.Context, tx pgx.Tx, t *model.PaymentTransaction) error {
	query := `INSERT INTO payment_transactions
		(id, tenant_id, settlement_id, order_id, provider, external_transaction_id,
		 amount, fee, net_amount, currency, transaction_type, transaction_date,
		 match_status, match_notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	_, err := tx.Exec(ctx, query,
		t.ID, t.TenantID, t.SettlementID, t.OrderID,
		t.Provider, t.ExternalTransactionID,
		t.Amount, t.Fee, t.NetAmount,
		t.Currency, t.TransactionType, t.TransactionDate,
		t.MatchStatus, t.MatchNotes,
	)
	if err != nil {
		return fmt.Errorf("create payment transaction: %w", err)
	}
	return nil
}

func (r *PaymentRepository) UpdateTransactionMatch(ctx context.Context, tx pgx.Tx, id uuid.UUID, orderID *uuid.UUID, matchStatus string, matchNotes *string) error {
	query := `UPDATE payment_transactions SET order_id = $1, match_status = $2, match_notes = $3 WHERE id = $4`
	_, err := tx.Exec(ctx, query, orderID, matchStatus, matchNotes, id)
	if err != nil {
		return fmt.Errorf("update transaction match: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetReconciliationSummary(ctx context.Context, tx pgx.Tx, dateFrom, dateTo *string) (*model.ReconciliationSummary, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if dateFrom != nil {
		where += fmt.Sprintf(" AND transaction_date >= $%d", argIdx)
		args = append(args, *dateFrom)
		argIdx++
	}
	if dateTo != nil {
		where += fmt.Sprintf(" AND transaction_date <= $%d", argIdx)
		args = append(args, *dateTo)
		argIdx++ //nolint:ineffassign
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_transactions,
			COALESCE(SUM(CASE WHEN match_status = 'matched' THEN 1 ELSE 0 END), 0) AS matched_count,
			COALESCE(SUM(CASE WHEN match_status = 'unmatched' THEN 1 ELSE 0 END), 0) AS unmatched_count,
			COALESCE(SUM(CASE WHEN match_status = 'discrepancy' THEN 1 ELSE 0 END), 0) AS discrepancy_count,
			COALESCE(SUM(CASE WHEN match_status = 'manual_match' THEN 1 ELSE 0 END), 0) AS manual_match_count,
			COALESCE(SUM(amount), 0) AS total_amount,
			COALESCE(SUM(fee), 0) AS total_fees,
			COALESCE(SUM(net_amount), 0) AS total_net,
			COALESCE(SUM(CASE WHEN match_status IN ('matched', 'manual_match') THEN amount ELSE 0 END), 0) AS matched_amount,
			COALESCE(SUM(CASE WHEN match_status = 'unmatched' THEN amount ELSE 0 END), 0) AS unmatched_amount
		FROM payment_transactions %s
	`, where)

	var summary model.ReconciliationSummary
	err := tx.QueryRow(ctx, query, args...).Scan(
		&summary.TotalTransactions,
		&summary.MatchedCount,
		&summary.UnmatchedCount,
		&summary.DiscrepancyCount,
		&summary.ManualMatchCount,
		&summary.TotalAmount,
		&summary.TotalFees,
		&summary.TotalNet,
		&summary.MatchedAmount,
		&summary.UnmatchedAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("get reconciliation summary: %w", err)
	}
	return &summary, nil
}
