package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// haversine
// ---------------------------------------------------------------------------

func TestHaversine_SamePoint(t *testing.T) {
	dist := haversine(52.23, 21.01, 52.23, 21.01)
	assert.Equal(t, 0.0, dist)
}

func TestHaversine_KnownCityPairs(t *testing.T) {
	tests := []struct {
		name    string
		lat1    float64
		lon1    float64
		lat2    float64
		lon2    float64
		wantMin float64
		wantMax float64
	}{
		{
			name: "Warsaw to Krakow (~250 km)",
			lat1: 52.23, lon1: 21.01,
			lat2: 50.06, lon2: 19.94,
			wantMin: 240, wantMax: 260,
		},
		{
			name: "Warsaw to Gdansk (~300 km)",
			lat1: 52.23, lon1: 21.01,
			lat2: 54.35, lon2: 18.65,
			wantMin: 280, wantMax: 310,
		},
		{
			name: "Krakow to Wroclaw (~240 km)",
			lat1: 50.06, lon1: 19.94,
			lat2: 51.11, lon2: 17.04,
			wantMin: 220, wantMax: 260,
		},
		{
			name: "Poznan to Szczecin (~250 km)",
			lat1: 52.41, lon1: 16.93,
			lat2: 53.43, lon2: 14.55,
			wantMin: 180, wantMax: 220,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := haversine(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.GreaterOrEqual(t, dist, tt.wantMin, "distance should be >= %f km", tt.wantMin)
			assert.LessOrEqual(t, dist, tt.wantMax, "distance should be <= %f km", tt.wantMax)
		})
	}
}

func TestHaversine_Symmetry(t *testing.T) {
	// Distance A->B should equal B->A
	d1 := haversine(52.23, 21.01, 50.06, 19.94)
	d2 := haversine(50.06, 19.94, 52.23, 21.01)
	assert.InDelta(t, d1, d2, 0.001)
}

func TestHaversine_Antipodal(t *testing.T) {
	// Opposite sides of the earth: ~20,000 km (half circumference)
	dist := haversine(0, 0, 0, 180)
	assert.InDelta(t, math.Pi*6371.0, dist, 1.0)
}

func TestHaversine_AlongEquator(t *testing.T) {
	// 1 degree of longitude at the equator is ~111.2 km
	dist := haversine(0, 0, 0, 1)
	assert.InDelta(t, 111.2, dist, 1.0)
}

// ---------------------------------------------------------------------------
// extractPostalPrefix
// ---------------------------------------------------------------------------

func TestExtractPostalPrefix(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "standard format XX-XXX", input: "00-001", expect: "00"},
		{name: "no dash", input: "00001", expect: "00"},
		{name: "with spaces", input: "30 001", expect: "30"},
		{name: "Krakow code", input: "31-456", expect: "31"},
		{name: "Gdansk code", input: "80-100", expect: "80"},
		{name: "single digit", input: "5", expect: ""},
		{name: "empty string", input: "", expect: ""},
		{name: "dash only", input: "-", expect: ""},
		{name: "spaces only", input: "   ", expect: ""},
		{name: "two digits only", input: "30", expect: "30"},
		{name: "code with leading spaces", input: " 50-100", expect: "50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPostalPrefix(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// ---------------------------------------------------------------------------
// isInternational
// ---------------------------------------------------------------------------

func TestIsInternational(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{name: "Polish code with dash", input: "00-001", expect: false},
		{name: "Polish code without dash", input: "00001", expect: false},
		{name: "Polish code Krakow", input: "31-456", expect: false},
		{name: "German postal code (5 letters)", input: "10115", expect: false}, // 5 digits, looks Polish structurally
		{name: "UK postal code", input: "SW1A1AA", expect: true},
		{name: "US ZIP code (short)", input: "9021", expect: true},  // only 4 digits
		{name: "US ZIP+4", input: "90210-1234", expect: true},       // 9 digits after cleanup
		{name: "French postal code", input: "750008", expect: true}, // 6 digits
		{name: "empty string", input: "", expect: false},
		{name: "letters in code", input: "AB-CDE", expect: true},
		{name: "mixed alphanumeric", input: "1A2B3", expect: true},
		{name: "three digit code", input: "123", expect: true},
		{name: "six digit code", input: "123456", expect: true},
		{name: "code with spaces only digits", input: "00 001", expect: false}, // spaces stripped -> 00001
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInternational(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getEmissionFactor
// ---------------------------------------------------------------------------

func TestGetEmissionFactor(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect float64
	}{
		{name: "inpost", input: "inpost", expect: 0.045},
		{name: "dhl", input: "dhl", expect: 0.080},
		{name: "dhl_express", input: "dhl_express", expect: 0.080},
		{name: "gls", input: "gls", expect: 0.055},
		{name: "dpd", input: "dpd", expect: 0.055},
		{name: "ups", input: "ups", expect: 0.065},
		{name: "fedex", input: "fedex", expect: 0.070},
		{name: "poczta_polska", input: "poczta_polska", expect: 0.050},
		{name: "orlen_paczka", input: "orlen_paczka", expect: 0.052},
		{name: "uppercase InPost", input: "InPost", expect: 0.045},
		{name: "uppercase DHL", input: "DHL", expect: 0.080},
		{name: "with leading space", input: " inpost", expect: 0.045},
		{name: "with trailing space", input: "inpost ", expect: 0.045},
		{name: "with surrounding spaces", input: "  dhl  ", expect: 0.080},
		{name: "unknown carrier", input: "unknown_carrier", expect: defaultEmissionFactor},
		{name: "empty string", input: "", expect: defaultEmissionFactor},
		{name: "random text", input: "some_new_carrier", expect: defaultEmissionFactor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEmissionFactor(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// ---------------------------------------------------------------------------
// EstimateCarbon
// ---------------------------------------------------------------------------

func TestEstimateCarbon_KnownCarriers(t *testing.T) {
	tests := []struct {
		name       string
		carrier    string
		weightKg   float64
		distanceKm float64
		expect     float64
	}{
		{
			name:       "InPost 2kg 300km",
			carrier:    "inpost",
			weightKg:   2.0,
			distanceKm: 300.0,
			// 2.0 * 300.0 * 0.045 / 1000 = 0.027
			expect: 0.027,
		},
		{
			name:       "DHL 5kg 500km",
			carrier:    "dhl",
			weightKg:   5.0,
			distanceKm: 500.0,
			// 5.0 * 500.0 * 0.080 / 1000 = 0.2
			expect: 0.2,
		},
		{
			name:       "UPS 10kg 1000km",
			carrier:    "ups",
			weightKg:   10.0,
			distanceKm: 1000.0,
			// 10.0 * 1000.0 * 0.065 / 1000 = 0.65
			expect: 0.65,
		},
		{
			name:       "FedEx 1kg 100km",
			carrier:    "fedex",
			weightKg:   1.0,
			distanceKm: 100.0,
			// 1.0 * 100.0 * 0.070 / 1000 = 0.007
			expect: 0.007,
		},
		{
			name:       "Poczta Polska 3kg 200km",
			carrier:    "poczta_polska",
			weightKg:   3.0,
			distanceKm: 200.0,
			// 3.0 * 200.0 * 0.050 / 1000 = 0.03
			expect: 0.03,
		},
		{
			name:       "Orlen Paczka 4kg 150km",
			carrier:    "orlen_paczka",
			weightKg:   4.0,
			distanceKm: 150.0,
			// 4.0 * 150.0 * 0.052 / 1000 = 0.0312
			expect: 0.0312,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateCarbon(tt.carrier, tt.weightKg, tt.distanceKm)
			assert.InDelta(t, tt.expect, result, 0.0001)
		})
	}
}

func TestEstimateCarbon_UnknownCarrier(t *testing.T) {
	// Unknown carrier should use defaultEmissionFactor = 0.062
	result := EstimateCarbon("unknown", 2.0, 300.0)
	// 2.0 * 300.0 * 0.062 / 1000 = 0.0372
	assert.InDelta(t, 0.0372, result, 0.0001)
}

func TestEstimateCarbon_ZeroWeight(t *testing.T) {
	// Zero weight should default to 1.0 kg
	result := EstimateCarbon("inpost", 0, 300.0)
	// 1.0 * 300.0 * 0.045 / 1000 = 0.0135
	assert.InDelta(t, 0.0135, result, 0.0001)
}

func TestEstimateCarbon_NegativeWeight(t *testing.T) {
	// Negative weight should default to 1.0 kg
	result := EstimateCarbon("inpost", -5.0, 300.0)
	// 1.0 * 300.0 * 0.045 / 1000 = 0.0135
	assert.InDelta(t, 0.0135, result, 0.0001)
}

func TestEstimateCarbon_ZeroDistance(t *testing.T) {
	result := EstimateCarbon("inpost", 2.0, 0)
	assert.Equal(t, 0.0, result)
}

func TestEstimateCarbon_EmptyCarrier(t *testing.T) {
	// Empty carrier should use default factor
	result := EstimateCarbon("", 1.0, 1000.0)
	// 1.0 * 1000.0 * 0.062 / 1000 = 0.062
	assert.InDelta(t, 0.062, result, 0.0001)
}

func TestEstimateCarbon_LargeValues(t *testing.T) {
	result := EstimateCarbon("dhl", 1000.0, 10000.0)
	// 1000.0 * 10000.0 * 0.080 / 1000 = 800.0
	assert.InDelta(t, 800.0, result, 0.01)
}

func TestEstimateCarbon_CaseInsensitive(t *testing.T) {
	lower := EstimateCarbon("inpost", 2.0, 300.0)
	upper := EstimateCarbon("INPOST", 2.0, 300.0)
	mixed := EstimateCarbon("InPost", 2.0, 300.0)
	assert.Equal(t, lower, upper)
	assert.Equal(t, lower, mixed)
}

// ---------------------------------------------------------------------------
// EstimateDistance
// ---------------------------------------------------------------------------

func TestEstimateDistance_SameCity(t *testing.T) {
	// Two codes in the same Warsaw prefix -> same centroid -> haversine=0 -> clamped to 20 km
	dist := EstimateDistance("00-001", "00-999")
	// Same prefix "00", haversine = 0, clamped to 20, then * 1.3 = 26.0
	assert.InDelta(t, 26.0, dist, 0.1)
}

func TestEstimateDistance_SamePrefixDifferentCodes(t *testing.T) {
	// Both in Krakow prefix "30"
	dist := EstimateDistance("30-001", "30-999")
	assert.InDelta(t, 26.0, dist, 0.1)
}

func TestEstimateDistance_WarsawToKrakow(t *testing.T) {
	// "00" (Warsaw) to "30" (Krakow) ~ 252 km straight-line * 1.3 ~ 327 km
	dist := EstimateDistance("00-001", "30-001")
	assert.Greater(t, dist, 280.0)
	assert.Less(t, dist, 380.0)
}

func TestEstimateDistance_WarsawToGdansk(t *testing.T) {
	// "00" (Warsaw) to "80" (Gdansk) ~ 290 km straight-line * 1.3 ~ 377 km
	dist := EstimateDistance("00-001", "80-001")
	assert.Greater(t, dist, 320.0)
	assert.Less(t, dist, 420.0)
}

func TestEstimateDistance_WarsawToWroclaw(t *testing.T) {
	// "00" (Warsaw) to "50" (Wroclaw) ~ 300 km straight-line * 1.3 ~ 390 km
	dist := EstimateDistance("00-001", "50-001")
	assert.Greater(t, dist, 300.0)
	assert.Less(t, dist, 450.0)
}

func TestEstimateDistance_DifferentCities(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		dest   string
		minKm  float64
		maxKm  float64
	}{
		{
			name:   "Krakow to Wroclaw",
			origin: "30-001",
			dest:   "50-001",
			minKm:  250,
			maxKm:  360,
		},
		{
			name:   "Poznan to Szczecin",
			origin: "60-001",
			dest:   "70-001",
			minKm:  200,
			maxKm:  320,
		},
		{
			name:   "Lublin to Bialystok",
			origin: "20-001",
			dest:   "15-001",
			minKm:  200,
			maxKm:  320,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := EstimateDistance(tt.origin, tt.dest)
			assert.Greater(t, dist, tt.minKm, "distance should be > %f km", tt.minKm)
			assert.Less(t, dist, tt.maxKm, "distance should be < %f km", tt.maxKm)
		})
	}
}

func TestEstimateDistance_Symmetry(t *testing.T) {
	d1 := EstimateDistance("00-001", "30-001")
	d2 := EstimateDistance("30-001", "00-001")
	assert.Equal(t, d1, d2)
}

func TestEstimateDistance_InternationalOrigin(t *testing.T) {
	// UK postal code -> international default
	dist := EstimateDistance("SW1A1AA", "00-001")
	assert.Equal(t, defaultInternationalDistanceKm, dist)
}

func TestEstimateDistance_InternationalDest(t *testing.T) {
	dist := EstimateDistance("00-001", "SW1A1AA")
	assert.Equal(t, defaultInternationalDistanceKm, dist)
}

func TestEstimateDistance_BothInternational(t *testing.T) {
	dist := EstimateDistance("SW1A1AA", "10115DE")
	assert.Equal(t, defaultInternationalDistanceKm, dist)
}

func TestEstimateDistance_UnknownPolishPrefix(t *testing.T) {
	// "ZZ" is not in the map but "ZZ001" is 5 chars, all not digits -> international
	// Let's use a 5-digit code with an unmapped prefix: e.g. "11-111" -- "11" IS mapped
	// We need a prefix that is 5 digits but not in the map.
	// Looking at postalPrefixCoords: "01"-"09", "10"-"19", "20"-"29", "30"-"34", "35"-"39",
	// "40"-"44", "45"-"49", "50"-"59", "60"-"64", "65"-"69", "70"-"78", "80"-"89", "90"-"99"
	// Not all prefixes are mapped. Let's check a couple that shouldn't be there.
	// Looking at the map: no "79" entry. Let's try it.
	// Actually wait -- let me re-read the map. No "79" in the map.
	// "79-001" -> prefix "79", not in map -> not international (5 digits, all digits) -> domestic default
	dist := EstimateDistance("79-001", "00-001")
	assert.Equal(t, defaultDomesticDistanceKm, dist)
}

func TestEstimateDistance_EmptyOrigin(t *testing.T) {
	// Empty string -> extractPostalPrefix returns "" -> not in map
	// isInternational("") returns false -> domestic default
	dist := EstimateDistance("", "00-001")
	assert.Equal(t, defaultDomesticDistanceKm, dist)
}

func TestEstimateDistance_EmptyBoth(t *testing.T) {
	dist := EstimateDistance("", "")
	assert.Equal(t, defaultDomesticDistanceKm, dist)
}

func TestEstimateDistance_CodesWithoutDash(t *testing.T) {
	// Should work the same with or without dash
	withDash := EstimateDistance("00-001", "30-001")
	withoutDash := EstimateDistance("00001", "30001")
	assert.Equal(t, withDash, withoutDash)
}

func TestEstimateDistance_MinimumClamp(t *testing.T) {
	// Same-prefix codes produce haversine=0, clamped to 20 km, then * 1.3 = 26.0
	dist := EstimateDistance("00-001", "00-002")
	assert.GreaterOrEqual(t, dist, 26.0)
}

func TestEstimateDistance_ResultIsRounded(t *testing.T) {
	// Result should be rounded to 1 decimal place
	dist := EstimateDistance("00-001", "30-001")
	rounded := math.Round(dist*10) / 10
	assert.Equal(t, rounded, dist)
}

// ---------------------------------------------------------------------------
// NewCarbonService
// ---------------------------------------------------------------------------

func TestCarbonService_New(t *testing.T) {
	svc := NewCarbonService(nil)
	assert.NotNil(t, svc)
}
