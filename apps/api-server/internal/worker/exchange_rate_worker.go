package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ExchangeRateWorker fetches exchange rates daily from NBP for all tenants.
type ExchangeRateWorker struct {
	pool                *pgxpool.Pool
	exchangeRateService *service.ExchangeRateService
	logger              *slog.Logger
}

func NewExchangeRateWorker(pool *pgxpool.Pool, exchangeRateService *service.ExchangeRateService, logger *slog.Logger) *ExchangeRateWorker {
	return &ExchangeRateWorker{
		pool:                pool,
		exchangeRateService: exchangeRateService,
		logger:              logger,
	}
}

func (w *ExchangeRateWorker) Name() string {
	return "exchange_rate_fetcher"
}

func (w *ExchangeRateWorker) Interval() time.Duration {
	return 24 * time.Hour
}

func (w *ExchangeRateWorker) Run(ctx context.Context) error {
	// Get all tenant IDs
	rows, err := w.pool.Query(ctx, "SELECT id FROM tenants")
	if err != nil {
		return err
	}
	defer rows.Close()

	var tenantIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			w.logger.Error("exchange rate worker: scan tenant", "error", err)
			continue
		}
		tenantIDs = append(tenantIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	totalFetched := 0
	for _, tenantID := range tenantIDs {
		count, err := w.exchangeRateService.FetchNBPRates(ctx, tenantID, uuid.Nil, "worker")
		if err != nil {
			w.logger.Error("exchange rate worker: fetch NBP rates", "tenant_id", tenantID, "error", err)
			continue
		}
		totalFetched += count
	}

	w.logger.Info("exchange rate worker completed", "tenants", len(tenantIDs), "rates_fetched", totalFetched)
	return nil
}
