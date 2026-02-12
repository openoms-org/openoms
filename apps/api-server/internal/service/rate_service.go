package service

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// RateService aggregates shipping rates from all active carrier integrations.
type RateService struct {
	integrationRepo repository.IntegrationRepo
	pool            *pgxpool.Pool
	encryptionKey   []byte
}

// NewRateService creates a new RateService.
func NewRateService(
	integrationRepo repository.IntegrationRepo,
	pool *pgxpool.Pool,
	encryptionKey []byte,
) *RateService {
	return &RateService{
		integrationRepo: integrationRepo,
		pool:            pool,
		encryptionKey:   encryptionKey,
	}
}

// GetRates queries all active carrier integrations in parallel and returns
// their rates sorted by price ascending.
func (s *RateService) GetRates(ctx context.Context, tenantID uuid.UUID, req integration.RateRequest) ([]integration.Rate, error) {
	// Load all integrations within the tenant transaction
	var integrations []struct {
		provider string
		creds    []byte
		settings []byte
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		all, err := s.integrationRepo.List(ctx, tx)
		if err != nil {
			return err
		}

		// Carrier provider names that we know support rates
		carrierProviders := map[string]bool{
			"inpost":        true,
			"dhl":           true,
			"dpd":           true,
			"gls":           true,
			"ups":           true,
			"poczta_polska": true,
			"orlen_paczka":  true,
			"fedex":         true,
		}

		for _, intg := range all {
			if intg.Status != "active" {
				continue
			}
			if !carrierProviders[intg.Provider] {
				continue
			}
			if intg.EncryptedCredentials == "" {
				continue
			}

			credJSON, err := crypto.Decrypt(intg.EncryptedCredentials, s.encryptionKey)
			if err != nil {
				slog.Warn("rate_service: failed to decrypt credentials",
					"provider", intg.Provider, "error", err)
				continue
			}

			integrations = append(integrations, struct {
				provider string
				creds    []byte
				settings []byte
			}{
				provider: intg.Provider,
				creds:    credJSON,
				settings: intg.Settings,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(integrations) == 0 {
		return []integration.Rate{}, nil
	}

	// Query all carriers in parallel
	var mu sync.Mutex
	var allRates []integration.Rate
	var wg sync.WaitGroup

	for _, intg := range integrations {
		wg.Add(1)
		go func(provider string, creds []byte, settings []byte) {
			defer wg.Done()

			carrier, err := integration.NewCarrierProvider(provider, creds, settings)
			if err != nil {
				slog.Warn("rate_service: failed to create carrier provider",
					"provider", provider, "error", err)
				return
			}

			rates, err := carrier.GetRates(ctx, req)
			if err != nil {
				slog.Warn("rate_service: carrier GetRates failed",
					"provider", provider, "error", err)
				return
			}

			if len(rates) > 0 {
				mu.Lock()
				allRates = append(allRates, rates...)
				mu.Unlock()
			}
		}(intg.provider, intg.creds, intg.settings)
	}

	wg.Wait()

	// Sort by price ascending
	sort.Slice(allRates, func(i, j int) bool {
		return allRates[i].Price < allRates[j].Price
	})

	return allRates, nil
}
