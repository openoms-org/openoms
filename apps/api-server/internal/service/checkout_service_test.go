package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
)

var testPlans = []config.PlanConfig{
	{
		ID:             "standard",
		Name:           "Standard",
		MonthlyPriceID: "price_month_std",
		YearlyPriceID:  "price_year_std",
		MonthlyAmount:  9900,
		YearlyAmount:   99000,
		Currency:       "pln",
		TrialDays:      14,
		Features:       []string{"5 users"},
		Limits:         config.PlanLimits{MaxUsers: 5, MaxOrdersMonthly: 1000, MaxIntegrations: 2},
	},
	{
		ID:             "pro",
		Name:           "Pro",
		MonthlyPriceID: "price_month_pro",
		YearlyPriceID:  "price_year_pro",
		MonthlyAmount:  49900,
		YearlyAmount:   499000,
		Currency:       "pln",
		TrialDays:      14,
		Features:       []string{"Unlimited users"},
		Limits:         config.PlanLimits{MaxUsers: 0, MaxOrdersMonthly: 0, MaxIntegrations: 0},
	},
}

func TestCheckoutService_ListPlans(t *testing.T) {
	svc := NewCheckoutService(nil, nil, testPlans)
	plans := svc.ListPlans()

	assert.Len(t, plans, 2)
	assert.Equal(t, "standard", plans[0].ID)
	assert.Equal(t, "Standard", plans[0].Name)
	assert.Equal(t, int64(9900), plans[0].MonthlyAmount)
	assert.Equal(t, int64(99000), plans[0].YearlyAmount)
	assert.Equal(t, "pln", plans[0].Currency)
	assert.Equal(t, int64(14), plans[0].TrialDays)
	assert.Equal(t, 5, plans[0].Limits.MaxUsers)

	// Ensure public plan info is JSON-serializable without sensitive fields
	b, err := json.Marshal(plans[0])
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	_, hasMonthlyPriceID := m["monthly_price_id"]
	_, hasYearlyPriceID := m["yearly_price_id"]
	assert.False(t, hasMonthlyPriceID, "monthly_price_id should not be in JSON")
	assert.False(t, hasYearlyPriceID, "yearly_price_id should not be in JSON")
}

func TestCheckoutService_ListPlans_Empty(t *testing.T) {
	svc := NewCheckoutService(nil, nil, nil)
	plans := svc.ListPlans()
	assert.Len(t, plans, 0)
}

func TestCheckoutService_FindPlan(t *testing.T) {
	svc := NewCheckoutService(nil, nil, testPlans)

	plan := svc.FindPlan("standard")
	assert.NotNil(t, plan)
	assert.Equal(t, "Standard", plan.Name)
	assert.Equal(t, "price_month_std", plan.MonthlyPriceID)

	plan = svc.FindPlan("pro")
	assert.NotNil(t, plan)
	assert.Equal(t, "Pro", plan.Name)

	plan = svc.FindPlan("enterprise")
	assert.Nil(t, plan)
}

func TestPlanLimitsJSON(t *testing.T) {
	plan := &config.PlanConfig{
		Limits: config.PlanLimits{
			MaxUsers:         5,
			MaxOrdersMonthly: 1000,
			MaxIntegrations:  2,
		},
	}

	raw := PlanLimitsJSON(plan)
	var m map[string]any
	err := json.Unmarshal(raw, &m)
	require.NoError(t, err)
	assert.Equal(t, float64(5), m["max_users"])
	assert.Equal(t, float64(1000), m["max_orders_monthly"])
	assert.Equal(t, float64(2), m["max_integrations"])
}
