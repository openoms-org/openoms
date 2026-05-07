package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfigForValidation(registrationMode string) *Config {
	return &Config{
		Env:              "production",
		EncryptionKey:    strings.Repeat("a", 64),
		JWTSecret:        strings.Repeat("b", 32),
		RegistrationMode: registrationMode,
	}
}

func TestConfig_Validate_RegistrationModes(t *testing.T) {
	for _, mode := range []string{"open", "invite", "closed", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			err := validConfigForValidation(mode).Validate()
			require.NoError(t, err)
		})
	}
}

func TestConfig_Validate_RegistrationModeRejectsUnknown(t *testing.T) {
	err := validConfigForValidation("typo").Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "REGISTRATION_MODE must be one of")
}

func TestConfig_InMemoryStatePolicy(t *testing.T) {
	tests := []struct {
		name         string
		cfg          Config
		allowState   bool
		requireRedis bool
	}{
		{
			name:         "development allows in-memory state by default",
			cfg:          Config{Env: "development"},
			allowState:   true,
			requireRedis: false,
		},
		{
			name:         "staging requires Redis by default",
			cfg:          Config{Env: "staging"},
			allowState:   false,
			requireRedis: true,
		},
		{
			name:         "production requires Redis by default",
			cfg:          Config{Env: "production"},
			allowState:   false,
			requireRedis: true,
		},
		{
			name:         "explicit opt-in allows in-memory state outside development",
			cfg:          Config{Env: "production", AllowInMemoryState: true},
			allowState:   true,
			requireRedis: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.allowState, tt.cfg.AllowsInMemoryState())
			assert.Equal(t, tt.requireRedis, tt.cfg.RequiresRedis())
		})
	}
}

func TestConfig_ParseLicensePublicKey_Valid(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	encoded := base64.StdEncoding.EncodeToString(pub)

	cfg := &Config{LicensePublicKey: encoded}
	key, err := cfg.ParseLicensePublicKey()
	require.NoError(t, err)
	assert.Equal(t, pub, key)
}

func TestConfig_ParseLicensePublicKey_Empty(t *testing.T) {
	cfg := &Config{LicensePublicKey: ""}
	key, err := cfg.ParseLicensePublicKey()
	assert.NoError(t, err)
	assert.Nil(t, key)
}

func TestConfig_ParseLicensePublicKey_Invalid(t *testing.T) {
	cfg := &Config{LicensePublicKey: "not-valid-base64!!!"}
	_, err := cfg.ParseLicensePublicKey()
	assert.Error(t, err)
}

func TestConfig_ParseLicensePublicKey_WrongLength(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("too-short"))
	cfg := &Config{LicensePublicKey: encoded}
	_, err := cfg.ParseLicensePublicKey()
	assert.Error(t, err)
}

// --- Billing Plan Parsing Tests ---

func TestConfig_ParseBillingPlans_Valid(t *testing.T) {
	plans := []PlanConfig{
		{
			ID:             "standard",
			Name:           "Standard",
			MonthlyPriceID: "price_month_std",
			YearlyPriceID:  "price_year_std",
			MonthlyAmount:  9900,
			YearlyAmount:   99000,
			Currency:       "pln",
			TrialDays:      14,
			Limits:         PlanLimits{MaxUsers: 5, MaxOrdersMonthly: 1000, MaxIntegrations: 2},
			Features:       []string{"5 users", "1000 orders/mo"},
		},
		{
			ID:             "pro",
			Name:           "Pro",
			MonthlyPriceID: "price_month_pro",
			YearlyPriceID:  "price_year_pro",
			MonthlyAmount:  49900,
			YearlyAmount:   499000,
			Currency:       "eur",
			TrialDays:      30,
			Limits:         PlanLimits{MaxUsers: 0, MaxOrdersMonthly: 0, MaxIntegrations: 0},
			Features:       []string{"Unlimited users", "Unlimited orders"},
		},
	}
	b, err := json.Marshal(plans)
	require.NoError(t, err)

	cfg := &Config{BillingPlansJSON: string(b)}
	parsed, err := cfg.ParseBillingPlans()
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	assert.Equal(t, "standard", parsed[0].ID)
	assert.Equal(t, "Standard", parsed[0].Name)
	assert.Equal(t, "price_month_std", parsed[0].MonthlyPriceID)
	assert.Equal(t, "price_year_std", parsed[0].YearlyPriceID)
	assert.Equal(t, int64(9900), parsed[0].MonthlyAmount)
	assert.Equal(t, int64(99000), parsed[0].YearlyAmount)
	assert.Equal(t, "pln", parsed[0].Currency)
	assert.Equal(t, int64(14), parsed[0].TrialDays)
	assert.Equal(t, 5, parsed[0].Limits.MaxUsers)
	assert.Equal(t, 1000, parsed[0].Limits.MaxOrdersMonthly)
	assert.Equal(t, 2, parsed[0].Limits.MaxIntegrations)
	assert.Equal(t, []string{"5 users", "1000 orders/mo"}, parsed[0].Features)

	assert.Equal(t, "pro", parsed[1].ID)
	assert.Equal(t, "Pro", parsed[1].Name)
	assert.Equal(t, "eur", parsed[1].Currency)
	assert.Equal(t, int64(30), parsed[1].TrialDays)
}

func TestConfig_ParseBillingPlans_Empty(t *testing.T) {
	cfg := &Config{BillingPlansJSON: ""}
	plans, err := cfg.ParseBillingPlans()
	assert.NoError(t, err)
	assert.Nil(t, plans)
}

func TestConfig_ParseBillingPlans_InvalidJSON(t *testing.T) {
	cfg := &Config{BillingPlansJSON: "not valid json ["}
	_, err := cfg.ParseBillingPlans()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestConfig_ParseBillingPlans_MissingID(t *testing.T) {
	// Plan with empty id should fail validation
	cfg := &Config{BillingPlansJSON: `[{"name":"Standard","monthly_amount":9900}]`}
	_, err := cfg.ParseBillingPlans()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id and name are required")
}

func TestConfig_ParseBillingPlans_MissingName(t *testing.T) {
	// Plan with id but no name should also fail
	cfg := &Config{BillingPlansJSON: `[{"id":"standard","monthly_amount":9900}]`}
	_, err := cfg.ParseBillingPlans()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id and name are required")
}

func TestConfig_ParseBillingPlans_DefaultCurrency(t *testing.T) {
	// When currency is empty, it should default to "pln"
	cfg := &Config{BillingPlansJSON: `[{"id":"basic","name":"Basic","monthly_amount":4900}]`}
	plans, err := cfg.ParseBillingPlans()
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "pln", plans[0].Currency)
}

func TestConfig_ParseBillingPlans_CurrencyPreserved(t *testing.T) {
	// When currency is set explicitly, it should not be overwritten
	cfg := &Config{BillingPlansJSON: `[{"id":"basic","name":"Basic","currency":"eur"}]`}
	plans, err := cfg.ParseBillingPlans()
	require.NoError(t, err)
	require.Len(t, plans, 1)
	assert.Equal(t, "eur", plans[0].Currency)
}

func TestConfig_BillingEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected bool
	}{
		{
			name:     "both set",
			cfg:      Config{StripeSecretKey: "sk_test_123", BillingPlansJSON: `[{"id":"x","name":"X"}]`},
			expected: true,
		},
		{
			name:     "missing stripe key",
			cfg:      Config{BillingPlansJSON: `[{"id":"x","name":"X"}]`},
			expected: false,
		},
		{
			name:     "missing plans",
			cfg:      Config{StripeSecretKey: "sk_test_123"},
			expected: false,
		},
		{
			name:     "both empty",
			cfg:      Config{},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.BillingEnabled())
		})
	}
}
