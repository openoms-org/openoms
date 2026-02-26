package service

import (
	"encoding/json"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTenantSettings_WithLimits(t *testing.T) {
	limits := &model.LicenseLimits{
		MaxUsers:         10,
		MaxOrdersMonthly: 5000,
		MaxIntegrations:  5,
	}

	settings := buildTenantSettings(nil, limits)
	require.NotNil(t, settings)

	var parsed map[string]any
	err := json.Unmarshal(settings, &parsed)
	require.NoError(t, err)

	limitsMap := parsed["limits"].(map[string]any)
	assert.Equal(t, float64(10), limitsMap["max_users"])
	assert.Equal(t, float64(5000), limitsMap["max_orders_monthly"])
	assert.Equal(t, float64(5), limitsMap["max_integrations"])
}

func TestBuildTenantSettings_WithoutLimits(t *testing.T) {
	settings := buildTenantSettings(nil, nil)

	// Should return nil when no limits
	if settings != nil {
		var parsed map[string]any
		err := json.Unmarshal(settings, &parsed)
		require.NoError(t, err)
		_, hasLimits := parsed["limits"]
		assert.False(t, hasLimits)
	}
}

func TestBuildTenantSettings_MergesExisting(t *testing.T) {
	existing := json.RawMessage(`{"custom_field":"value"}`)
	limits := &model.LicenseLimits{MaxUsers: 3}

	settings := buildTenantSettings(existing, limits)
	require.NotNil(t, settings)

	var parsed map[string]any
	err := json.Unmarshal(settings, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "value", parsed["custom_field"])
	limitsMap := parsed["limits"].(map[string]any)
	assert.Equal(t, float64(3), limitsMap["max_users"])
}
