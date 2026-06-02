package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v82"
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

	// PlanLimitsJSON must produce nested structure: {"limits": {"max_users": ...}}
	// to match PlanGuard middleware's PlanSettings{Limits: ...} unmarshaling.
	limitsRaw, ok := m["limits"]
	require.True(t, ok, "expected nested 'limits' key")
	limits, ok := limitsRaw.(map[string]any)
	require.True(t, ok, "expected 'limits' to be a map")
	assert.Equal(t, float64(5), limits["max_users"])
	assert.Equal(t, float64(1000), limits["max_orders_monthly"])
	assert.Equal(t, float64(2), limits["max_integrations"])
}

func TestCheckoutServiceGetSessionStatusStoresStripeRefsWhenSessionCompletes(t *testing.T) {
	trialEnd := time.Unix(1_800_000_000, 0)
	currentPeriodStart := time.Unix(1_700_000_000, 0)
	currentPeriodEnd := time.Unix(1_702_592_000, 0)

	repo := &checkoutFinalizeRepo{
		session: &model.BillingCheckoutSession{
			StripeSessionID: "cs_test",
			Plan:            "plus",
			BillingInterval: "month",
			Status:          "pending",
		},
	}
	svc := NewCheckoutService(repo, nil, []config.PlanConfig{{ID: "plus"}})
	var expands []string
	svc.getCheckoutSession = func(_ string, params *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		for _, expand := range params.Expand {
			expands = append(expands, *expand)
		}
		return &stripe.CheckoutSession{
			Status:          stripe.CheckoutSessionStatusComplete,
			CustomerDetails: &stripe.CheckoutSessionCustomerDetails{Email: "billing@example.com"},
			Customer:        &stripe.Customer{ID: "cus_live"},
			Subscription: &stripe.Subscription{
				ID:       "sub_live",
				Status:   stripe.SubscriptionStatusTrialing,
				TrialEnd: trialEnd.Unix(),
				Items: &stripe.SubscriptionItemList{
					Data: []*stripe.SubscriptionItem{
						{
							CurrentPeriodStart: currentPeriodStart.Unix(),
							CurrentPeriodEnd:   currentPeriodEnd.Unix(),
						},
					},
				},
			},
		}, nil
	}

	status, err := svc.GetSessionStatus(context.Background(), "cs_test")

	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Equal(t, "completed", status.Status)
	assert.Equal(t, "billing@example.com", status.Email)
	assert.ElementsMatch(t, []string{"customer", "subscription", "subscription.items.data"}, expands)
	assert.Equal(t, "cs_test", repo.completedSessionID)
	assert.Equal(t, "billing@example.com", repo.completedEmail)
	assert.Equal(t, "cus_live", repo.completedRefs.StripeCustomerID)
	assert.Equal(t, "sub_live", repo.completedRefs.StripeSubscriptionID)
	assert.Equal(t, "trialing", repo.completedRefs.SubscriptionStatus)
	assert.Equal(t, trialEnd, *repo.completedRefs.TrialEnd)
	assert.Equal(t, currentPeriodStart, *repo.completedRefs.CurrentPeriodStart)
	assert.Equal(t, currentPeriodEnd, *repo.completedRefs.CurrentPeriodEnd)
}

func TestCheckoutServiceFinalizeCheckoutClaimUsesStoredStripeRefsWhenStripeFetchFails(t *testing.T) {
	tenantID := uuid.New()
	customerID := "cus_stored"
	subscriptionID := "sub_stored"
	subscriptionStatus := "active"
	trialEnd := time.Unix(1_800_000_000, 0)

	repo := &checkoutFinalizeRepo{
		session: &model.BillingCheckoutSession{
			StripeSessionID:      "cs_test",
			Plan:                 "plus",
			BillingInterval:      "month",
			Status:               "registered",
			TenantID:             &tenantID,
			StripeCustomerID:     &customerID,
			StripeSubscriptionID: &subscriptionID,
			SubscriptionStatus:   &subscriptionStatus,
			TrialEnd:             &trialEnd,
			CurrentPeriodStart:   nil,
			CurrentPeriodEnd:     nil,
		},
	}
	svc := NewCheckoutService(repo, nil, []config.PlanConfig{{ID: "plus"}})
	svc.getCheckoutSession = func(string, *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		return nil, errors.New("stripe unavailable")
	}

	err := svc.FinalizeCheckoutClaim(context.Background(), "cs_test", tenantID, "plus", "month")

	require.NoError(t, err)
	assert.Equal(t, tenantID, repo.updatedTenantID)
	assert.Equal(t, customerID, repo.createdCustomerID)
	require.NotNil(t, repo.upsertedSubscription)
	assert.Equal(t, tenantID, repo.upsertedSubscription.TenantID)
	assert.Equal(t, subscriptionID, repo.upsertedSubscription.StripeSubscriptionID)
	assert.Equal(t, customerID, repo.upsertedSubscription.StripeCustomerID)
	assert.Equal(t, "plus", repo.upsertedSubscription.Plan)
	assert.Equal(t, "month", repo.upsertedSubscription.BillingInterval)
	assert.Equal(t, subscriptionStatus, repo.upsertedSubscription.Status)
	assert.Equal(t, trialEnd, *repo.upsertedSubscription.TrialEnd)
}

func TestCheckoutServiceFinalizeSkipsStripeWhenStoredRefsComplete(t *testing.T) {
	tenantID := uuid.New()
	customerID := "cus_stored"
	subscriptionID := "sub_stored"
	subscriptionStatus := "active"

	repo := &checkoutFinalizeRepo{
		session: &model.BillingCheckoutSession{
			StripeSessionID:      "cs_test",
			Plan:                 "plus",
			BillingInterval:      "month",
			Status:               "registered",
			TenantID:             &tenantID,
			StripeCustomerID:     &customerID,
			StripeSubscriptionID: &subscriptionID,
			SubscriptionStatus:   &subscriptionStatus,
		},
	}
	svc := NewCheckoutService(repo, nil, []config.PlanConfig{{ID: "plus"}})
	// OPE-506: with complete stored refs, finalization must NOT hit the Stripe API.
	stripeCalled := false
	svc.getCheckoutSession = func(string, *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		stripeCalled = true
		return nil, errors.New("Stripe must not be called when stored refs are complete")
	}

	err := svc.FinalizeCheckoutClaim(context.Background(), "cs_test", tenantID, "plus", "month")

	require.NoError(t, err)
	assert.False(t, stripeCalled, "Stripe API must not be called when stored refs are complete")
	assert.Equal(t, customerID, repo.createdCustomerID)
	require.NotNil(t, repo.upsertedSubscription)
	assert.Equal(t, subscriptionID, repo.upsertedSubscription.StripeSubscriptionID)
}

func TestCheckoutServiceReconcileCheckoutClaimsHealsGap(t *testing.T) {
	tenantID := uuid.New()
	customerID := "cus_stored"
	subscriptionID := "sub_stored"
	subscriptionStatus := "active"

	repo := &checkoutFinalizeRepo{
		// GetCheckoutSession (used by finalization fallback) returns stored Stripe refs.
		session: &model.BillingCheckoutSession{
			StripeSessionID:      "cs_gap",
			Plan:                 "plus",
			BillingInterval:      "month",
			Status:               "registered",
			TenantID:             &tenantID,
			StripeCustomerID:     &customerID,
			StripeSubscriptionID: &subscriptionID,
			SubscriptionStatus:   &subscriptionStatus,
		},
		unreconciled: []model.BillingCheckoutSession{
			{StripeSessionID: "cs_gap", Plan: "plus", BillingInterval: "month", TenantID: &tenantID},
		},
	}
	svc := NewCheckoutService(repo, nil, []config.PlanConfig{{ID: "plus"}})
	// Force the stored-refs fallback so the test does not hit the Stripe API.
	svc.getCheckoutSession = func(string, *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		return nil, errors.New("stripe unavailable")
	}

	reconciled, failed, err := svc.ReconcileCheckoutClaims(context.Background(), nil, 50)

	require.NoError(t, err)
	assert.Equal(t, 1, reconciled)
	assert.Equal(t, 0, failed)
	assert.Equal(t, customerID, repo.createdCustomerID)
	require.NotNil(t, repo.upsertedSubscription)
	assert.Equal(t, tenantID, repo.upsertedSubscription.TenantID)
	assert.Equal(t, subscriptionID, repo.upsertedSubscription.StripeSubscriptionID)
}

func TestCheckoutServiceReconcileCheckoutClaimsReportsFailure(t *testing.T) {
	tenantID := uuid.New()

	repo := &checkoutFinalizeRepo{
		// Stored session carries no Stripe refs, so finalization cannot complete.
		session: &model.BillingCheckoutSession{
			StripeSessionID: "cs_gap",
			Plan:            "plus",
			BillingInterval: "month",
			Status:          "registered",
			TenantID:        &tenantID,
		},
		unreconciled: []model.BillingCheckoutSession{
			{StripeSessionID: "cs_gap", Plan: "plus", BillingInterval: "month", TenantID: &tenantID},
		},
	}
	svc := NewCheckoutService(repo, nil, []config.PlanConfig{{ID: "plus"}})
	svc.getCheckoutSession = func(string, *stripe.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
		return nil, errors.New("stripe unavailable")
	}

	reconciled, failed, err := svc.ReconcileCheckoutClaims(context.Background(), nil, 50)

	require.Error(t, err)
	assert.Equal(t, 0, reconciled)
	assert.Equal(t, 1, failed)
	// The error must reference the tenant for alerting, never the raw session ID.
	assert.Contains(t, err.Error(), tenantID.String())
	assert.NotContains(t, err.Error(), "cs_gap")
}

type checkoutFinalizeRepo struct {
	session              *model.BillingCheckoutSession
	unreconciled         []model.BillingCheckoutSession
	completedSessionID   string
	completedEmail       string
	completedRefs        model.CheckoutSessionStripeRefs
	updatedTenantID      uuid.UUID
	createdCustomerID    string
	upsertedSubscription *model.BillingSubscription
}

func (f *checkoutFinalizeRepo) CreateCheckoutSession(context.Context, *pgxpool.Pool, string, string, string) error {
	return nil
}

func (f *checkoutFinalizeRepo) FindUnreconciledCheckoutSessions(context.Context, *pgxpool.Pool, int) ([]model.BillingCheckoutSession, error) {
	return f.unreconciled, nil
}

func (f *checkoutFinalizeRepo) CompleteCheckoutSession(_ context.Context, _ *pgxpool.Pool, stripeSessionID, email string, refs model.CheckoutSessionStripeRefs) (bool, error) {
	f.completedSessionID = stripeSessionID
	f.completedEmail = email
	f.completedRefs = refs
	return true, nil
}

func (f *checkoutFinalizeRepo) GetCheckoutSession(context.Context, *pgxpool.Pool, string) (*model.BillingCheckoutSession, error) {
	return f.session, nil
}

func (f *checkoutFinalizeRepo) ClaimCheckoutSession(context.Context, *pgxpool.Pool, string) (bool, error) {
	return true, nil
}

func (f *checkoutFinalizeRepo) UpdateClaimedCheckoutTenant(_ context.Context, _ *pgxpool.Pool, _ string, tenantID uuid.UUID) error {
	f.updatedTenantID = tenantID
	return nil
}

func (f *checkoutFinalizeRepo) CreateBillingCustomer(_ context.Context, _ *pgxpool.Pool, _ uuid.UUID, stripeCustomerID string) error {
	f.createdCustomerID = stripeCustomerID
	return nil
}

func (f *checkoutFinalizeRepo) GetCustomerByStripeID(context.Context, *pgxpool.Pool, string) (*model.BillingCustomer, error) {
	return nil, nil
}

func (f *checkoutFinalizeRepo) UpsertSubscription(_ context.Context, _ *pgxpool.Pool, sub *model.BillingSubscription) error {
	subCopy := *sub
	f.upsertedSubscription = &subCopy
	return nil
}

func (f *checkoutFinalizeRepo) UpdateSubscriptionByStripeID(context.Context, *pgxpool.Pool, string, string, *time.Time, *time.Time, *time.Time) error {
	return nil
}

func (f *checkoutFinalizeRepo) GetSubscriptionByTenant(context.Context, pgx.Tx) (*model.BillingSubscription, error) {
	return nil, nil
}

func (f *checkoutFinalizeRepo) SyncTenantPlan(context.Context, *pgxpool.Pool, uuid.UUID, []byte) error {
	return nil
}
