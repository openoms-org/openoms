package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrExchangeRateNotFound = errors.New("exchange rate not found")
	ErrRateNotAvailable     = errors.New("exchange rate not available for this currency pair")
)

type ExchangeRateService struct {
	exchangeRateRepo repository.ExchangeRateRepo
	auditRepo        repository.AuditRepo
	pool             *pgxpool.Pool
}

func NewExchangeRateService(
	exchangeRateRepo repository.ExchangeRateRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
) *ExchangeRateService {
	return &ExchangeRateService{
		exchangeRateRepo: exchangeRateRepo,
		auditRepo:        auditRepo,
		pool:             pool,
	}
}

func (s *ExchangeRateService) List(ctx context.Context, tenantID uuid.UUID, filter model.ExchangeRateListFilter) ([]model.ExchangeRate, int, error) {
	var rates []model.ExchangeRate
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rates, total, err = s.exchangeRateRepo.List(ctx, tx, filter)
		return err
	})
	return rates, total, err
}

func (s *ExchangeRateService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ExchangeRate, error) {
	var rate *model.ExchangeRate
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rate, err = s.exchangeRateRepo.FindByID(ctx, tx, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, ErrExchangeRateNotFound
	}
	return rate, nil
}

func (s *ExchangeRateService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateExchangeRateRequest, actorID uuid.UUID, ip string) (*model.ExchangeRate, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	source := "manual"
	if req.Source != "" {
		source = req.Source
	}

	rate := &model.ExchangeRate{
		ID:             uuid.New(),
		TenantID:       tenantID,
		BaseCurrency:   req.BaseCurrency,
		TargetCurrency: req.TargetCurrency,
		Rate:           req.Rate,
		Source:         source,
		FetchedAt:      time.Now(),
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.exchangeRateRepo.Create(ctx, tx, rate); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "exchange_rate.created",
			EntityType: "exchange_rate",
			EntityID:   rate.ID,
			Changes:    map[string]string{"base": req.BaseCurrency, "target": req.TargetCurrency, "rate": fmt.Sprintf("%.6f", req.Rate)},
			IPAddress:  ip,
		})
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, NewValidationError(errors.New("exchange rate for this currency pair already exists"))
		}
		return nil, err
	}
	return rate, nil
}

func (s *ExchangeRateService) Update(ctx context.Context, tenantID, id uuid.UUID, req model.UpdateExchangeRateRequest, actorID uuid.UUID, ip string) (*model.ExchangeRate, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var rate *model.ExchangeRate
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rate, err = s.exchangeRateRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if rate == nil {
			return ErrExchangeRateNotFound
		}

		if err := s.exchangeRateRepo.Update(ctx, tx, id, req); err != nil {
			return err
		}

		rate, err = s.exchangeRateRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "exchange_rate.updated",
			EntityType: "exchange_rate",
			EntityID:   id,
			IPAddress:  ip,
		})
	})
	if err != nil {
		return nil, err
	}
	return rate, nil
}

func (s *ExchangeRateService) Delete(ctx context.Context, tenantID, id uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rate, err := s.exchangeRateRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if rate == nil {
			return ErrExchangeRateNotFound
		}

		if err := s.exchangeRateRepo.Delete(ctx, tx, id); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "exchange_rate.deleted",
			EntityType: "exchange_rate",
			EntityID:   id,
			Changes:    map[string]string{"base": rate.BaseCurrency, "target": rate.TargetCurrency},
			IPAddress:  ip,
		})
	})
}

func (s *ExchangeRateService) ConvertAmount(ctx context.Context, tenantID uuid.UUID, amount float64, from, to string) (*model.ConvertAmountResponse, error) {
	if from == to {
		return &model.ConvertAmountResponse{
			OriginalAmount:  amount,
			ConvertedAmount: amount,
			From:            from,
			To:              to,
			Rate:            1.0,
		}, nil
	}

	var rate *model.ExchangeRate
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var err error
		rate, err = s.exchangeRateRepo.GetRate(ctx, tx, from, to)
		if err != nil {
			return err
		}
		if rate == nil {
			// Try reverse direction
			rate, err = s.exchangeRateRepo.GetRate(ctx, tx, to, from)
			if err != nil {
				return err
			}
			if rate != nil {
				// Invert the rate
				invertedRate := 1.0 / rate.Rate
				rate = &model.ExchangeRate{
					Rate:           invertedRate,
					BaseCurrency:   from,
					TargetCurrency: to,
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, ErrRateNotAvailable
	}

	converted := math.Round(amount*rate.Rate*100) / 100

	return &model.ConvertAmountResponse{
		OriginalAmount:  amount,
		ConvertedAmount: converted,
		From:            from,
		To:              to,
		Rate:            rate.Rate,
	}, nil
}

// NBP API response structures
type nbpTable struct {
	Table         string    `json:"table"`
	No            string    `json:"no"`
	EffectiveDate string    `json:"effectiveDate"`
	Rates         []nbpRate `json:"rates"`
}

type nbpRate struct {
	Currency string  `json:"currency"`
	Code     string  `json:"code"`
	Mid      float64 `json:"mid"`
}

// FetchNBPRates fetches exchange rates from NBP API (Polish National Bank) for PLN base.
func (s *ExchangeRateService) FetchNBPRates(ctx context.Context, tenantID uuid.UUID, actorID uuid.UUID, ip string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.nbp.pl/api/exchangerates/tables/A/?format=json", nil)
	if err != nil {
		return 0, fmt.Errorf("create NBP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch NBP rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("NBP API returned status %d", resp.StatusCode)
	}

	var tables []nbpTable
	if err := json.NewDecoder(resp.Body).Decode(&tables); err != nil {
		return 0, fmt.Errorf("decode NBP response: %w", err)
	}

	if len(tables) == 0 {
		return 0, fmt.Errorf("no NBP rate tables returned")
	}

	count := 0
	now := time.Now()

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		for _, nbpRate := range tables[0].Rates {
			rate := &model.ExchangeRate{
				ID:             uuid.New(),
				TenantID:       tenantID,
				BaseCurrency:   "PLN",
				TargetCurrency: nbpRate.Code,
				Rate:           1.0 / nbpRate.Mid, // NBP gives foreign->PLN, we want PLN->foreign
				Source:         "nbp",
				FetchedAt:      now,
			}
			if err := s.exchangeRateRepo.Upsert(ctx, tx, rate); err != nil {
				return fmt.Errorf("upsert rate %s: %w", nbpRate.Code, err)
			}
			count++
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "exchange_rate.nbp_fetched",
			EntityType: "exchange_rate",
			EntityID:   uuid.Nil,
			Changes:    map[string]string{"count": fmt.Sprintf("%d", count)},
			IPAddress:  ip,
		})
	})

	return count, err
}
