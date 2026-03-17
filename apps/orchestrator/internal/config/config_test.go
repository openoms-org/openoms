package config_test

import (
	"testing"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "lin_api_test")
	t.Setenv("LINEAR_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "lin_api_test", cfg.LinearAPIKey)
	assert.Equal(t, "whsec_test", cfg.LinearWebhookSecret)
	assert.Equal(t, "OPE", cfg.LinearTeamID)
	assert.Equal(t, 5, cfg.MaxCIRetries)
	assert.Equal(t, 300, cfg.PollIntervalSeconds)
	assert.Equal(t, float64(280), cfg.BudgetMonthlyLimit)
}

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "")
	t.Setenv("LINEAR_WEBHOOK_SECRET", "")

	_, err := config.Load()
	assert.Error(t, err)
}

func TestConfig_Validate(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "lin_api_test")
	t.Setenv("LINEAR_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MAX_CI_RETRIES", "0")

	cfg, err := config.Load()
	require.NoError(t, err)

	err = cfg.Validate()
	assert.Error(t, err, "MaxCIRetries=0 should fail validation")
}
