package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ExchangeRateRepository handles persistence for currency exchange rates.
type ExchangeRateRepository struct{}

// NewExchangeRateRepository creates a new ExchangeRateRepository.
func NewExchangeRateRepository() *ExchangeRateRepository {
	return &ExchangeRateRepository{}
}

// exchangeRateColumns is the canonical column list for SELECTing an exchange rate row.
const exchangeRateColumns = "id, tenant_id, base_currency, target_currency, rate, source, fetched_at, created_at"

// scanExchangeRate scans a single exchange rate row in exchangeRateColumns order.
func scanExchangeRate(row pgx.Row) (model.ExchangeRate, error) {
	var rate model.ExchangeRate
	err := row.Scan(
		&rate.ID, &rate.TenantID, &rate.BaseCurrency, &rate.TargetCurrency,
		&rate.Rate, &rate.Source, &rate.FetchedAt, &rate.CreatedAt,
	)
	return rate, err
}

// List returns a paginated list of exchange rates matching the filter.
func (r *ExchangeRateRepository) List(ctx context.Context, tx pgx.Tx, filter model.ExchangeRateListFilter) ([]model.ExchangeRate, int, error) {
	qb := NewQueryBuilder()
	if filter.BaseCurrency != nil {
		qb.Add("base_currency = $%d", *filter.BaseCurrency)
	}
	if filter.TargetCurrency != nil {
		qb.Add("target_currency = $%d", *filter.TargetCurrency)
	}
	where := qb.WhereClause()

	var total int
	countQuery := "SELECT COUNT(*) FROM exchange_rates " + where
	if err := tx.QueryRow(ctx, countQuery, qb.Args()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count exchange_rates: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":      "created_at",
		"base_currency":   "base_currency",
		"target_currency": "target_currency",
		"rate":            "rate",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	limitIdx := qb.AddArgs(filter.Limit, filter.Offset)
	query := fmt.Sprintf(
		"SELECT "+exchangeRateColumns+" FROM exchange_rates %s %s LIMIT $%d OFFSET $%d",
		where, orderByClause, limitIdx, limitIdx+1,
	)

	rows, err := tx.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list exchange_rates: %w", err)
	}
	defer rows.Close()

	var rates []model.ExchangeRate
	for rows.Next() {
		rate, err := scanExchangeRate(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan exchange_rate: %w", err)
		}
		rates = append(rates, rate)
	}
	return rates, total, rows.Err()
}

// FindByID returns an exchange rate by its ID.
func (r *ExchangeRateRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ExchangeRate, error) {
	rate, err := scanExchangeRate(tx.QueryRow(ctx,
		"SELECT "+exchangeRateColumns+" FROM exchange_rates WHERE id = $1", id,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find exchange_rate by id: %w", err)
	}
	return &rate, nil
}

// GetRate returns the exchange rate for a given currency pair.
func (r *ExchangeRateRepository) GetRate(ctx context.Context, tx pgx.Tx, baseCurrency, targetCurrency string) (*model.ExchangeRate, error) {
	rate, err := scanExchangeRate(tx.QueryRow(ctx,
		"SELECT "+exchangeRateColumns+" FROM exchange_rates WHERE base_currency = $1 AND target_currency = $2", baseCurrency, targetCurrency,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get exchange_rate: %w", err)
	}
	return &rate, nil
}

// Create inserts a new exchange rate record.
func (r *ExchangeRateRepository) Create(ctx context.Context, tx pgx.Tx, rate *model.ExchangeRate) error {
	return tx.QueryRow(ctx,
		`INSERT INTO exchange_rates (id, tenant_id, base_currency, target_currency, rate, source, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		rate.ID, rate.TenantID, rate.BaseCurrency, rate.TargetCurrency,
		rate.Rate, rate.Source, rate.FetchedAt,
	).Scan(&rate.CreatedAt)
}

// Upsert inserts or updates the exchange rate for a currency pair.
func (r *ExchangeRateRepository) Upsert(ctx context.Context, tx pgx.Tx, rate *model.ExchangeRate) error {
	return tx.QueryRow(ctx,
		`INSERT INTO exchange_rates (id, tenant_id, base_currency, target_currency, rate, source, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (tenant_id, base_currency, target_currency)
		 DO UPDATE SET rate = EXCLUDED.rate, source = EXCLUDED.source, fetched_at = EXCLUDED.fetched_at
		 RETURNING created_at`,
		rate.ID, rate.TenantID, rate.BaseCurrency, rate.TargetCurrency,
		rate.Rate, rate.Source, rate.FetchedAt,
	).Scan(&rate.CreatedAt)
}

// Update applies partial updates to an exchange rate record.
func (r *ExchangeRateRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateExchangeRateRequest) error {
	ub := NewUpdateBuilder()
	SetPtr(ub, "rate", req.Rate)
	SetPtr(ub, "source", req.Source)

	if ub.IsEmpty() {
		return nil
	}

	ub.SetRaw("fetched_at = NOW()")

	query := fmt.Sprintf("UPDATE exchange_rates SET %s WHERE id = $%d",
		ub.SetClause(), ub.NextArgIdx())
	args := append(ub.Args(), id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update exchange_rate: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("exchange rate not found")
	}
	return nil
}

// Delete removes an exchange rate record by its ID.
func (r *ExchangeRateRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM exchange_rates WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete exchange_rate: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("exchange rate not found")
	}
	return nil
}
