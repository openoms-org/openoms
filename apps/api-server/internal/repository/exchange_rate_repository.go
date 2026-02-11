package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type ExchangeRateRepository struct{}

func NewExchangeRateRepository() *ExchangeRateRepository {
	return &ExchangeRateRepository{}
}

func (r *ExchangeRateRepository) List(ctx context.Context, tx pgx.Tx, filter model.ExchangeRateListFilter) ([]model.ExchangeRate, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.BaseCurrency != nil {
		where += fmt.Sprintf(" AND base_currency = $%d", argIdx)
		args = append(args, *filter.BaseCurrency)
		argIdx++
	}
	if filter.TargetCurrency != nil {
		where += fmt.Sprintf(" AND target_currency = $%d", argIdx)
		args = append(args, *filter.TargetCurrency)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM exchange_rates " + where
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count exchange_rates: %w", err)
	}

	allowedSortColumns := map[string]string{
		"created_at":      "created_at",
		"base_currency":   "base_currency",
		"target_currency": "target_currency",
		"rate":            "rate",
	}
	orderByClause := model.BuildOrderByClause(filter.SortBy, filter.SortOrder, allowedSortColumns)

	query := fmt.Sprintf(
		`SELECT id, tenant_id, base_currency, target_currency, rate, source, fetched_at, created_at
		 FROM exchange_rates %s
		 %s
		 LIMIT $%d OFFSET $%d`,
		where, orderByClause, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list exchange_rates: %w", err)
	}
	defer rows.Close()

	var rates []model.ExchangeRate
	for rows.Next() {
		var rate model.ExchangeRate
		if err := rows.Scan(
			&rate.ID, &rate.TenantID, &rate.BaseCurrency, &rate.TargetCurrency,
			&rate.Rate, &rate.Source, &rate.FetchedAt, &rate.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan exchange_rate: %w", err)
		}
		rates = append(rates, rate)
	}
	return rates, total, rows.Err()
}

func (r *ExchangeRateRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ExchangeRate, error) {
	var rate model.ExchangeRate
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, base_currency, target_currency, rate, source, fetched_at, created_at
		 FROM exchange_rates WHERE id = $1`, id,
	).Scan(
		&rate.ID, &rate.TenantID, &rate.BaseCurrency, &rate.TargetCurrency,
		&rate.Rate, &rate.Source, &rate.FetchedAt, &rate.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find exchange_rate by id: %w", err)
	}
	return &rate, nil
}

func (r *ExchangeRateRepository) GetRate(ctx context.Context, tx pgx.Tx, baseCurrency, targetCurrency string) (*model.ExchangeRate, error) {
	var rate model.ExchangeRate
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, base_currency, target_currency, rate, source, fetched_at, created_at
		 FROM exchange_rates WHERE base_currency = $1 AND target_currency = $2`, baseCurrency, targetCurrency,
	).Scan(
		&rate.ID, &rate.TenantID, &rate.BaseCurrency, &rate.TargetCurrency,
		&rate.Rate, &rate.Source, &rate.FetchedAt, &rate.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get exchange_rate: %w", err)
	}
	return &rate, nil
}

func (r *ExchangeRateRepository) Create(ctx context.Context, tx pgx.Tx, rate *model.ExchangeRate) error {
	return tx.QueryRow(ctx,
		`INSERT INTO exchange_rates (id, tenant_id, base_currency, target_currency, rate, source, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		rate.ID, rate.TenantID, rate.BaseCurrency, rate.TargetCurrency,
		rate.Rate, rate.Source, rate.FetchedAt,
	).Scan(&rate.CreatedAt)
}

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

func (r *ExchangeRateRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateExchangeRateRequest) error {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if req.Rate != nil {
		setClauses = append(setClauses, fmt.Sprintf("rate = $%d", argIdx))
		args = append(args, *req.Rate)
		argIdx++
	}
	if req.Source != nil {
		setClauses = append(setClauses, fmt.Sprintf("source = $%d", argIdx))
		args = append(args, *req.Source)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "fetched_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE exchange_rates SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update exchange_rate: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("exchange rate not found")
	}
	return nil
}

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
