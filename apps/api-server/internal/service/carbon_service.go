package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// Emission factors: kg CO2 per tonne-km (DEFRA/GLEC Framework averages).
var carrierEmissionFactors = map[string]float64{
	"inpost":        0.045, // Locker network, optimized last-mile
	"dhl":           0.080, // DHL Express
	"dhl_express":   0.080,
	"gls":           0.055, // Parcel carrier
	"dpd":           0.055, // Parcel carrier
	"ups":           0.065,
	"fedex":         0.070,
	"poczta_polska": 0.050,
	"orlen_paczka":  0.052,
}

const (
	defaultEmissionFactor = 0.062 // Generic road freight
	airFreightFactor      = 0.602 // International air freight

	defaultDomesticDistanceKm      = 300.0
	defaultInternationalDistanceKm = 1500.0
)

// Polish postal code prefix (first 2 digits) to approximate region centroid (lat, lon).
// Based on voivodeship/region centroids.
var postalPrefixCoords = map[string][2]float64{
	// Warszawa & Mazowieckie
	"00": {52.23, 21.01}, "01": {52.23, 21.01}, "02": {52.23, 21.01},
	"03": {52.23, 21.01}, "04": {52.23, 21.01}, "05": {52.10, 20.90},
	"06": {52.76, 20.50}, "07": {52.55, 21.50}, "08": {51.90, 21.60},
	"09": {52.40, 19.80},
	// Łódzkie
	"90": {51.77, 19.46}, "91": {51.77, 19.46}, "92": {51.77, 19.46},
	"93": {51.77, 19.46}, "94": {51.77, 19.46}, "95": {51.77, 19.46},
	"96": {51.90, 19.20}, "97": {51.50, 19.50}, "98": {51.30, 18.90},
	"99": {51.90, 18.70},
	// Kraków & Małopolskie
	"30": {50.06, 19.94}, "31": {50.06, 19.94}, "32": {49.90, 19.80},
	"33": {49.70, 20.40}, "34": {49.80, 19.50},
	// Wrocław & Dolnośląskie
	"50": {51.11, 17.04}, "51": {51.11, 17.04}, "52": {51.11, 17.04},
	"53": {51.11, 17.04}, "54": {51.11, 17.04}, "55": {51.10, 16.90},
	"56": {51.20, 16.50}, "57": {50.80, 16.50}, "58": {50.80, 16.30},
	"59": {51.40, 15.90},
	// Poznań & Wielkopolskie
	"60": {52.41, 16.93}, "61": {52.41, 16.93}, "62": {52.30, 17.30},
	"63": {52.00, 17.50}, "64": {52.10, 16.50},
	// Gdańsk & Pomorskie
	"80": {54.35, 18.65}, "81": {54.50, 18.55}, "82": {53.80, 18.90},
	"83": {54.20, 17.90}, "84": {54.50, 17.60},
	// Katowice & Śląskie
	"40": {50.26, 19.02}, "41": {50.30, 18.95}, "42": {50.30, 19.10},
	"43": {49.80, 19.00}, "44": {50.20, 18.60},
	// Lublin & Lubelskie
	"20": {51.25, 22.57}, "21": {51.00, 22.00}, "22": {50.70, 23.50},
	"23": {50.90, 23.00}, "24": {51.60, 22.30},
	// Białystok & Podlaskie
	"15": {53.13, 23.16}, "16": {53.50, 22.70}, "17": {52.70, 22.00},
	"18": {53.80, 23.10}, "19": {53.50, 22.30},
	// Szczecin & Zachodniopomorskie
	"70": {53.43, 14.55}, "71": {53.43, 14.55}, "72": {53.90, 15.10},
	"73": {53.50, 15.50}, "74": {53.60, 14.50}, "75": {54.20, 16.20},
	"76": {54.00, 15.50}, "77": {53.80, 15.80}, "78": {54.10, 15.80},
	// Kielce & Świętokrzyskie
	"25": {50.87, 20.63}, "26": {51.10, 20.20}, "27": {50.70, 21.30},
	"28": {50.60, 20.50}, "29": {51.00, 20.80},
	// Olsztyn & Warmińsko-Mazurskie
	"10": {53.78, 20.48}, "11": {53.90, 20.50}, "12": {54.00, 19.80},
	"13": {53.60, 20.00}, "14": {53.50, 21.00},
	// Rzeszów & Podkarpackie
	"35": {50.04, 22.00}, "36": {50.10, 21.70}, "37": {50.30, 22.20},
	"38": {49.50, 22.50}, "39": {50.00, 21.50},
	// Opole & Opolskie
	"45": {50.67, 17.93}, "46": {50.60, 17.50}, "47": {50.50, 18.00},
	"48": {50.50, 17.30}, "49": {50.80, 17.80},
	// Bydgoszcz/Toruń & Kujawsko-Pomorskie
	"85": {53.12, 18.00}, "86": {53.00, 18.60}, "87": {53.01, 18.61},
	"88": {53.40, 17.90}, "89": {53.80, 18.50},
	// Zielona Góra & Lubuskie
	"65": {51.94, 15.51}, "66": {52.30, 15.30}, "67": {51.60, 15.20},
	"68": {51.70, 15.80}, "69": {52.10, 14.70},
}

// CarbonService provides carbon footprint estimation for shipments.
type CarbonService struct {
	pool *pgxpool.Pool
}

// NewCarbonService creates a new CarbonService.
func NewCarbonService(pool *pgxpool.Pool) *CarbonService {
	return &CarbonService{pool: pool}
}

// EstimateCarbon calculates estimated CO2 emissions in kg.
// Formula: carbonKg = weightKg * distanceKm * emissionFactor / 1000
func EstimateCarbon(carrier string, weightKg, distanceKm float64) float64 {
	factor := getEmissionFactor(carrier)
	if weightKg <= 0 {
		weightKg = 1.0 // Default 1 kg if weight unknown
	}
	return weightKg * distanceKm * factor / 1000.0
}

// getEmissionFactor returns the emission factor for a carrier (kg CO2 / tonne-km).
func getEmissionFactor(carrier string) float64 {
	normalized := strings.ToLower(strings.TrimSpace(carrier))
	if f, ok := carrierEmissionFactors[normalized]; ok {
		return f
	}
	return defaultEmissionFactor
}

// EstimateDistance estimates distance in km between two Polish postal codes.
// Uses Haversine formula on region centroids mapped from the first 2 digits.
func EstimateDistance(originPostalCode, destPostalCode string) float64 {
	originPrefix := extractPostalPrefix(originPostalCode)
	destPrefix := extractPostalPrefix(destPostalCode)

	// If either is not a valid Polish postal code, return defaults
	originCoords, originOK := postalPrefixCoords[originPrefix]
	destCoords, destOK := postalPrefixCoords[destPrefix]

	if !originOK || !destOK {
		// If one is non-Polish (international), use international default
		if isInternational(originPostalCode) || isInternational(destPostalCode) {
			return defaultInternationalDistanceKm
		}
		return defaultDomesticDistanceKm
	}

	dist := haversine(originCoords[0], originCoords[1], destCoords[0], destCoords[1])

	// Minimum 20 km (same-city deliveries still have some distance)
	if dist < 20 {
		dist = 20
	}

	// Road distance is ~1.3x straight-line distance
	return math.Round(dist*1.3*10) / 10
}

// extractPostalPrefix extracts the first 2 digits from a Polish postal code (XX-XXX format).
func extractPostalPrefix(postalCode string) string {
	cleaned := strings.ReplaceAll(postalCode, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if len(cleaned) >= 2 {
		return cleaned[:2]
	}
	return ""
}

// isInternational returns true if the postal code doesn't look like a Polish one.
func isInternational(postalCode string) bool {
	cleaned := strings.ReplaceAll(postalCode, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if len(cleaned) == 0 {
		return false
	}
	// Polish postal codes are 5 digits (XX-XXX → XXXXX after cleanup)
	if len(cleaned) != 5 {
		return true
	}
	for _, c := range cleaned {
		if c < '0' || c > '9' {
			return true
		}
	}
	return false
}

// haversine calculates the great-circle distance between two points (in km).
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// GetCarbonStats returns aggregate carbon statistics for a tenant within a date range.
func (s *CarbonService) GetCarbonStats(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo string) (*model.CarbonStats, error) {
	stats := &model.CarbonStats{}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Overall totals
		whereClause := "WHERE carbon_kg IS NOT NULL"
		args := []any{}
		argIdx := 1

		if dateFrom != "" {
			whereClause += fmt.Sprintf(" AND created_at >= $%d", argIdx)
			args = append(args, dateFrom)
			argIdx++
		}
		if dateTo != "" {
			whereClause += fmt.Sprintf(" AND created_at <= $%d", argIdx)
			args = append(args, dateTo+"T23:59:59Z")
			argIdx++
		}

		totalQuery := fmt.Sprintf(`
			SELECT
				COUNT(*),
				COALESCE(SUM(carbon_kg), 0),
				COALESCE(AVG(carbon_kg), 0),
				COALESCE(SUM(distance_km), 0)
			FROM shipments %s`, whereClause)

		if err := tx.QueryRow(ctx, totalQuery, args...).Scan(
			&stats.TotalShipments,
			&stats.TotalCarbonKg,
			&stats.AvgCarbonPerShip,
			&stats.TotalDistanceKm,
		); err != nil {
			return fmt.Errorf("carbon totals: %w", err)
		}

		// Round to 3 decimal places
		stats.TotalCarbonKg = math.Round(stats.TotalCarbonKg*1000) / 1000
		stats.AvgCarbonPerShip = math.Round(stats.AvgCarbonPerShip*1000) / 1000
		stats.TotalDistanceKm = math.Round(stats.TotalDistanceKm*10) / 10

		// By carrier
		byCarrierQuery := fmt.Sprintf(`
			SELECT
				provider,
				COUNT(*),
				COALESCE(SUM(carbon_kg), 0),
				COALESCE(AVG(carbon_kg), 0)
			FROM shipments %s
			GROUP BY provider
			ORDER BY SUM(carbon_kg) DESC`, whereClause)

		rows, err := tx.Query(ctx, byCarrierQuery, args...)
		if err != nil {
			return fmt.Errorf("carbon by carrier: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var cs model.CarrierCarbonStats
			if err := rows.Scan(&cs.Carrier, &cs.Shipments, &cs.TotalCarbon, &cs.AvgCarbon); err != nil {
				return fmt.Errorf("scan carrier carbon: %w", err)
			}
			cs.TotalCarbon = math.Round(cs.TotalCarbon*1000) / 1000
			cs.AvgCarbon = math.Round(cs.AvgCarbon*1000) / 1000
			stats.ByCarrier = append(stats.ByCarrier, cs)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("carrier carbon rows: %w", err)
		}

		// Monthly trend (last 12 months)
		monthlyQuery := fmt.Sprintf(`
			SELECT
				TO_CHAR(created_at, 'YYYY-MM') AS month,
				COUNT(*),
				COALESCE(SUM(carbon_kg), 0)
			FROM shipments %s
			GROUP BY TO_CHAR(created_at, 'YYYY-MM')
			ORDER BY month DESC
			LIMIT 12`, whereClause)

		monthRows, err := tx.Query(ctx, monthlyQuery, args...)
		if err != nil {
			return fmt.Errorf("carbon monthly: %w", err)
		}
		defer monthRows.Close()

		for monthRows.Next() {
			var ms model.MonthlyCarbonStat
			if err := monthRows.Scan(&ms.Month, &ms.Shipments, &ms.TotalCarbon); err != nil {
				return fmt.Errorf("scan monthly carbon: %w", err)
			}
			ms.TotalCarbon = math.Round(ms.TotalCarbon*1000) / 1000
			stats.MonthlyTrend = append(stats.MonthlyTrend, ms)
		}
		if err := monthRows.Err(); err != nil {
			return fmt.Errorf("monthly carbon rows: %w", err)
		}

		return nil
	})

	if stats.ByCarrier == nil {
		stats.ByCarrier = []model.CarrierCarbonStats{}
	}
	if stats.MonthlyTrend == nil {
		stats.MonthlyTrend = []model.MonthlyCarbonStat{}
	}

	return stats, err
}

// GetCarbonCSVData returns shipment-level carbon data for CSV export.
func (s *CarbonService) GetCarbonCSVData(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo string) ([]CarbonCSVRow, error) {
	var rows []CarbonCSVRow

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		whereClause := "WHERE carbon_kg IS NOT NULL"
		args := []any{}
		argIdx := 1

		if dateFrom != "" {
			whereClause += fmt.Sprintf(" AND created_at >= $%d", argIdx)
			args = append(args, dateFrom)
			argIdx++
		}
		if dateTo != "" {
			whereClause += fmt.Sprintf(" AND created_at <= $%d", argIdx)
			args = append(args, dateTo+"T23:59:59Z")
			argIdx++
		}

		query := fmt.Sprintf(`
			SELECT
				id, order_id, provider, tracking_number,
				weight, carbon_kg, distance_km, carbon_method,
				status, created_at
			FROM shipments %s
			ORDER BY created_at DESC
			LIMIT 10000`, whereClause)

		qRows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("carbon csv query: %w", err)
		}
		defer qRows.Close()

		for qRows.Next() {
			var r CarbonCSVRow
			if err := qRows.Scan(
				&r.ShipmentID, &r.OrderID, &r.Provider, &r.TrackingNumber,
				&r.Weight, &r.CarbonKg, &r.DistanceKm, &r.CarbonMethod,
				&r.Status, &r.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan carbon csv: %w", err)
			}
			rows = append(rows, r)
		}
		return qRows.Err()
	})

	return rows, err
}

// CarbonCSVRow holds a single row for the carbon CSV export.
type CarbonCSVRow struct {
	ShipmentID     string   `json:"shipment_id"`
	OrderID        string   `json:"order_id"`
	Provider       string   `json:"provider"`
	TrackingNumber *string  `json:"tracking_number"`
	Weight         *float64 `json:"weight"`
	CarbonKg       *float64 `json:"carbon_kg"`
	DistanceKm     *float64 `json:"distance_km"`
	CarbonMethod   *string  `json:"carbon_method"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"created_at"`
}
