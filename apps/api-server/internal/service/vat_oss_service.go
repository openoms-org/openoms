package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// EU VAT rates as of 2026. These change rarely and are maintained in code.
var euVATRates = map[string]model.VATRateSet{
	"AT": {Standard: 20, Reduced1: 10, Reduced2: 13},
	"BE": {Standard: 21, Reduced1: 6, Reduced2: 12},
	"BG": {Standard: 20, Reduced1: 9},
	"HR": {Standard: 25, Reduced1: 5, Reduced2: 13},
	"CY": {Standard: 19, Reduced1: 5, Reduced2: 9},
	"CZ": {Standard: 21, Reduced1: 12},
	"DK": {Standard: 25},
	"EE": {Standard: 22, Reduced1: 9},
	"FI": {Standard: 25.5, Reduced1: 10, Reduced2: 14},
	"FR": {Standard: 20, Reduced1: 5.5, Reduced2: 10, SuperReduced: 2.1},
	"DE": {Standard: 19, Reduced1: 7},
	"GR": {Standard: 24, Reduced1: 6, Reduced2: 13},
	"HU": {Standard: 27, Reduced1: 5, Reduced2: 18},
	"IE": {Standard: 23, Reduced1: 9, Reduced2: 13.5, SuperReduced: 4.8},
	"IT": {Standard: 22, Reduced1: 5, Reduced2: 10, SuperReduced: 4},
	"LV": {Standard: 21, Reduced1: 12, Reduced2: 5},
	"LT": {Standard: 21, Reduced1: 9, Reduced2: 5},
	"LU": {Standard: 17, Reduced1: 8, Reduced2: 14, SuperReduced: 3},
	"MT": {Standard: 18, Reduced1: 5, Reduced2: 7},
	"NL": {Standard: 21, Reduced1: 9},
	"PL": {Standard: 23, Reduced1: 5, Reduced2: 8},
	"PT": {Standard: 23, Reduced1: 6, Reduced2: 13},
	"RO": {Standard: 19, Reduced1: 5, Reduced2: 9},
	"SK": {Standard: 23, Reduced1: 10},
	"SI": {Standard: 22, Reduced1: 5, Reduced2: 9.5},
	"ES": {Standard: 21, Reduced1: 4, Reduced2: 10, SuperReduced: 0},
	"SE": {Standard: 25, Reduced1: 6, Reduced2: 12},
}

// euCountryNames maps ISO codes to human-readable names (Polish).
var euCountryNames = map[string]string{
	"AT": "Austria",
	"BE": "Belgia",
	"BG": "Bulgaria",
	"HR": "Chorwacja",
	"CY": "Cypr",
	"CZ": "Czechy",
	"DK": "Dania",
	"EE": "Estonia",
	"FI": "Finlandia",
	"FR": "Francja",
	"DE": "Niemcy",
	"GR": "Grecja",
	"HU": "Wegry",
	"IE": "Irlandia",
	"IT": "Wlochy",
	"LV": "Lotwa",
	"LT": "Litwa",
	"LU": "Luksemburg",
	"MT": "Malta",
	"NL": "Holandia",
	"PL": "Polska",
	"PT": "Portugalia",
	"RO": "Rumunia",
	"SK": "Slowacja",
	"SI": "Slowenia",
	"ES": "Hiszpania",
	"SE": "Szwecja",
}

// ossThresholdEUR is the annual cross-border distance selling threshold.
const ossThresholdEUR = 10000.0

// VATOSSService handles VAT OSS rate lookups, calculations, and reporting.
type VATOSSService struct {
	tenantRepo repository.TenantRepo
	pool       *pgxpool.Pool
}

// NewVATOSSService creates a new VATOSSService.
func NewVATOSSService(tenantRepo repository.TenantRepo, pool *pgxpool.Pool) *VATOSSService {
	return &VATOSSService{tenantRepo: tenantRepo, pool: pool}
}

// GetAllRates returns VAT rates for all EU countries.
func (s *VATOSSService) GetAllRates() map[string]model.VATRateSet {
	return euVATRates
}

// GetCountryRates returns VAT rates for a specific country.
func (s *VATOSSService) GetCountryRates(countryCode string) (*model.VATRateSet, bool) {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	rates, ok := euVATRates[code]
	if !ok {
		return nil, false
	}
	return &rates, true
}

// GetVATRate returns the VAT rate for a country and rate type.
func (s *VATOSSService) GetVATRate(countryCode, rateType string) float64 {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	rates, ok := euVATRates[code]
	if !ok {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(rateType)) {
	case "standard":
		return rates.Standard
	case "reduced_1":
		return rates.Reduced1
	case "reduced_2":
		return rates.Reduced2
	case "super_reduced":
		return rates.SuperReduced
	default:
		return rates.Standard
	}
}

// CalculateVAT calculates net, VAT amount, and gross for a given gross amount, country, and rate type.
func (s *VATOSSService) CalculateVAT(grossAmount float64, countryCode, rateType string) model.VATCalculation {
	rate := s.GetVATRate(countryCode, rateType)
	code := strings.ToUpper(strings.TrimSpace(countryCode))

	if rate == 0 {
		return model.VATCalculation{
			NetAmount:   roundCurrency(grossAmount),
			VATAmount:   0,
			GrossAmount: roundCurrency(grossAmount),
			VATRate:     0,
			Country:     code,
			RateType:    rateType,
		}
	}

	netAmount := grossAmount / (1 + rate/100)
	vatAmount := grossAmount - netAmount

	return model.VATCalculation{
		NetAmount:   roundCurrency(netAmount),
		VATAmount:   roundCurrency(vatAmount),
		GrossAmount: roundCurrency(grossAmount),
		VATRate:     rate,
		Country:     code,
		RateType:    rateType,
	}
}

// IsEUCountry checks whether the given country code is an EU member state.
func (s *VATOSSService) IsEUCountry(countryCode string) bool {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	_, ok := euVATRates[code]
	return ok
}

// GetOSSConfig reads the OSS configuration from tenant settings.
func (s *VATOSSService) GetOSSConfig(ctx context.Context, tenantID uuid.UUID) (*model.OSSConfig, error) {
	cfg := &model.OSSConfig{
		HomeCountry:    "PL",
		DefaultVATRate: "standard",
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		settings, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if settings == nil {
			return nil
		}
		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(settings, &allSettings); err != nil {
			return nil
		}
		raw, ok := allSettings["vat_oss"]
		if !ok {
			return nil
		}
		_ = json.Unmarshal(raw, cfg)
		return nil
	})

	return cfg, err
}

// UpdateOSSConfig updates the OSS configuration in tenant settings.
func (s *VATOSSService) UpdateOSSConfig(ctx context.Context, tenantID uuid.UUID, cfg model.OSSConfig) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		existing, err := s.tenantRepo.GetSettings(ctx, tx, tenantID)
		if err != nil {
			return err
		}

		var allSettings map[string]json.RawMessage
		if err := json.Unmarshal(existing, &allSettings); err != nil {
			allSettings = make(map[string]json.RawMessage)
		}

		valueJSON, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		allSettings["vat_oss"] = valueJSON

		newSettings, err := json.Marshal(allSettings)
		if err != nil {
			return err
		}

		return s.tenantRepo.UpdateSettings(ctx, tx, tenantID, newSettings)
	})
}

// GenerateOSSReport generates a quarterly OSS report aggregating cross-border EU sales.
func (s *VATOSSService) GenerateOSSReport(ctx context.Context, tenantID uuid.UUID, quarter, year int) (*model.OSSReport, error) {
	if quarter < 1 || quarter > 4 {
		return nil, fmt.Errorf("invalid quarter: %d (must be 1-4)", quarter)
	}
	if year < 2021 || year > 2100 {
		return nil, fmt.Errorf("invalid year: %d", year)
	}

	// Get home country from config
	ossCfg, err := s.GetOSSConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get OSS config: %w", err)
	}
	homeCountry := strings.ToUpper(ossCfg.HomeCountry)
	if homeCountry == "" {
		homeCountry = "PL"
	}

	// Determine quarter date range
	startMonth := (quarter-1)*3 + 1
	startDate := time.Date(year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 3, 0) // first day of next quarter

	report := &model.OSSReport{
		Quarter:     quarter,
		Year:        year,
		HomeCountry: homeCountry,
		ByCountry:   []model.OSSCountryReport{},
		GeneratedAt: time.Now().UTC(),
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Query orders within the quarter, extracting the shipping country code
		rows, err := tx.Query(ctx, `
			SELECT
				UPPER(COALESCE(shipping_address->>'country', '')) AS country,
				COUNT(*) AS order_count,
				COALESCE(SUM(total_amount), 0) AS total_sales
			FROM orders
			WHERE created_at >= $1
			  AND created_at < $2
			  AND status NOT IN ('cancelled', 'refunded')
			  AND UPPER(COALESCE(shipping_address->>'country', '')) != ''
			  AND UPPER(COALESCE(shipping_address->>'country', '')) != $3
			GROUP BY UPPER(COALESCE(shipping_address->>'country', ''))
			ORDER BY total_sales DESC
		`, startDate, endDate, homeCountry)
		if err != nil {
			return fmt.Errorf("query orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var country string
			var orderCount int
			var totalSales float64
			if err := rows.Scan(&country, &orderCount, &totalSales); err != nil {
				return fmt.Errorf("scan row: %w", err)
			}

			// Only include EU countries (non-EU are not subject to OSS)
			if _, isEU := euVATRates[country]; !isEU {
				continue
			}

			vatRate := euVATRates[country].Standard
			vatAmount := totalSales - (totalSales / (1 + vatRate/100))

			countryName := euCountryNames[country]
			if countryName == "" {
				countryName = country
			}

			report.ByCountry = append(report.ByCountry, model.OSSCountryReport{
				Country:     country,
				CountryName: countryName,
				OrderCount:  orderCount,
				TotalSales:  roundCurrency(totalSales),
				VATRate:     vatRate,
				VATAmount:   roundCurrency(vatAmount),
			})
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("rows error: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Calculate totals
	for _, cr := range report.ByCountry {
		report.TotalSales += cr.TotalSales
		report.TotalVAT += cr.VATAmount
	}
	report.TotalSales = roundCurrency(report.TotalSales)
	report.TotalVAT = roundCurrency(report.TotalVAT)

	return report, nil
}

// GetOSSThresholdStatus checks the annual cross-border EU sales threshold (EUR 10,000).
func (s *VATOSSService) GetOSSThresholdStatus(ctx context.Context, tenantID uuid.UUID, year int) (*model.ThresholdStatus, error) {
	ossCfg, err := s.GetOSSConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get OSS config: %w", err)
	}
	homeCountry := strings.ToUpper(ossCfg.HomeCountry)
	if homeCountry == "" {
		homeCountry = "PL"
	}

	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)

	status := &model.ThresholdStatus{
		Year:      year,
		Threshold: ossThresholdEUR,
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var total float64
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(total_amount), 0)
			FROM orders
			WHERE created_at >= $1
			  AND created_at < $2
			  AND status NOT IN ('cancelled', 'refunded')
			  AND UPPER(COALESCE(shipping_address->>'country', '')) != ''
			  AND UPPER(COALESCE(shipping_address->>'country', '')) != $3
		`, startDate, endDate, homeCountry).Scan(&total)
		if err != nil {
			return fmt.Errorf("sum cross-border: %w", err)
		}
		status.TotalCrossBorder = roundCurrency(total)
		return nil
	})
	if err != nil {
		return nil, err
	}

	status.Exceeded = status.TotalCrossBorder >= ossThresholdEUR
	remaining := ossThresholdEUR - status.TotalCrossBorder
	if remaining < 0 {
		remaining = 0
	}
	status.RemainingToThreshold = roundCurrency(remaining)

	return status, nil
}

// roundCurrency rounds a float to 2 decimal places.
func roundCurrency(val float64) float64 {
	return math.Round(val*100) / 100
}
