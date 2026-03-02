package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// Checkout session error sentinels.
var (
	ErrCheckoutSessionNotFound = errors.New("checkout session not found")
	ErrCheckoutSessionClaimed  = errors.New("checkout session already claimed")
	ErrCheckoutSessionPending  = errors.New("checkout session not yet completed")
	ErrPlanNotFound            = errors.New("plan not found")
)

// CheckoutService manages Stripe Checkout sessions for billing.
type CheckoutService struct {
	billingRepo repository.BillingRepo
	pool        *pgxpool.Pool
	plans       []config.PlanConfig
}

// NewCheckoutService creates a new CheckoutService.
func NewCheckoutService(billingRepo repository.BillingRepo, pool *pgxpool.Pool, plans []config.PlanConfig) *CheckoutService {
	return &CheckoutService{
		billingRepo: billingRepo,
		pool:        pool,
		plans:       plans,
	}
}

// ListPlans returns plans safe for frontend consumption (no Stripe Price IDs).
func (s *CheckoutService) ListPlans() []model.PublicPlanInfo {
	result := make([]model.PublicPlanInfo, len(s.plans))
	for i, p := range s.plans {
		result[i] = model.PublicPlanInfo{
			ID:            p.ID,
			Name:          p.Name,
			MonthlyAmount: p.MonthlyAmount,
			YearlyAmount:  p.YearlyAmount,
			Currency:      p.Currency,
			TrialDays:     p.TrialDays,
			Limits: model.LicenseLimits{
				MaxUsers:         p.Limits.MaxUsers,
				MaxOrdersMonthly: p.Limits.MaxOrdersMonthly,
				MaxIntegrations:  p.Limits.MaxIntegrations,
			},
			Features: p.Features,
		}
	}
	return result
}

// FindPlan returns a plan config by ID.
func (s *CheckoutService) FindPlan(planID string) *config.PlanConfig {
	for i := range s.plans {
		if s.plans[i].ID == planID {
			return &s.plans[i]
		}
	}
	return nil
}

// CreateCheckoutSession creates a Stripe Checkout Session and records it in the DB.
func (s *CheckoutService) CreateCheckoutSession(ctx context.Context, planID, interval, successURL, cancelURL string) (*model.CheckoutSessionResponse, error) {
	plan := s.FindPlan(planID)
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	var priceID string
	if interval == "year" {
		priceID = plan.YearlyPriceID
	} else {
		priceID = plan.MonthlyPriceID
	}
	if priceID == "" {
		return nil, fmt.Errorf("no price ID configured for plan %s interval %s", planID, interval)
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}
	if plan.TrialDays > 0 {
		params.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(plan.TrialDays),
		}
	}

	// Store plan info in metadata for webhook processing
	params.AddMetadata("plan_id", planID)
	params.AddMetadata("billing_interval", interval)

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe create checkout session: %w", err)
	}

	// Record in our DB — fail if DB write fails to avoid orphaned Stripe sessions
	if err := s.billingRepo.CreateCheckoutSession(ctx, s.pool, sess.ID, planID, interval); err != nil {
		return nil, fmt.Errorf("record checkout session in DB: %w", err)
	}

	return &model.CheckoutSessionResponse{
		CheckoutURL: sess.URL,
		SessionID:   sess.ID,
	}, nil
}

// GetSessionStatus returns the status of a checkout session.
// Checks DB first, falls back to Stripe API if session is still pending.
func (s *CheckoutService) GetSessionStatus(ctx context.Context, stripeSessionID string) (*model.CheckoutSessionStatus, error) {
	dbSession, err := s.billingRepo.GetCheckoutSession(ctx, s.pool, stripeSessionID)
	if err != nil {
		return nil, err
	}

	if dbSession != nil && dbSession.Status != "pending" {
		plan := s.FindPlan(dbSession.Plan)
		var limits *model.LicenseLimits
		if plan != nil {
			limits = &model.LicenseLimits{
				MaxUsers:         plan.Limits.MaxUsers,
				MaxOrdersMonthly: plan.Limits.MaxOrdersMonthly,
				MaxIntegrations:  plan.Limits.MaxIntegrations,
			}
		}
		return &model.CheckoutSessionStatus{
			Plan:     dbSession.Plan,
			Interval: dbSession.BillingInterval,
			Email:    dbSession.Email,
			Status:   dbSession.Status,
			Limits:   limits,
		}, nil
	}

	// Session is pending or not in DB — check Stripe API
	sess, err := session.Get(stripeSessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get session: %w", err)
	}

	status := "pending"
	var email string
	if sess.Status == stripe.CheckoutSessionStatusComplete {
		status = "completed"
		email = sess.CustomerDetails.Email
		// Update our DB with the completed status
		if _, err := s.billingRepo.CompleteCheckoutSession(ctx, s.pool, stripeSessionID, email); err != nil {
			slog.Error("failed to complete checkout session in DB", "stripe_session_id", stripeSessionID, "error", err)
		}
	}

	var planID, interval string
	if dbSession != nil {
		planID = dbSession.Plan
		interval = dbSession.BillingInterval
	} else if sess.Metadata != nil {
		planID = sess.Metadata["plan_id"]
		interval = sess.Metadata["billing_interval"]
	}

	plan := s.FindPlan(planID)
	var limits *model.LicenseLimits
	if plan != nil {
		limits = &model.LicenseLimits{
			MaxUsers:         plan.Limits.MaxUsers,
			MaxOrdersMonthly: plan.Limits.MaxOrdersMonthly,
			MaxIntegrations:  plan.Limits.MaxIntegrations,
		}
	}

	return &model.CheckoutSessionStatus{
		Plan:     planID,
		Interval: interval,
		Email:    email,
		Status:   status,
		Limits:   limits,
	}, nil
}

// ClaimSession atomically claims a checkout session BEFORE registration.
// This prevents TOCTOU race conditions — only one concurrent request succeeds.
// Tenant ID is set later via FinalizeCheckoutClaim after registration succeeds.
func (s *CheckoutService) ClaimSession(ctx context.Context, stripeSessionID string) error {
	// Get session to verify status and plan
	dbSession, err := s.billingRepo.GetCheckoutSession(ctx, s.pool, stripeSessionID)
	if err != nil {
		return err
	}
	if dbSession == nil {
		return ErrCheckoutSessionNotFound
	}
	if dbSession.Status == "registered" {
		return ErrCheckoutSessionClaimed
	}
	if dbSession.Status != "completed" {
		return ErrCheckoutSessionPending
	}

	// Atomically claim (pre-registration, no tenant_id)
	claimed, err := s.billingRepo.ClaimCheckoutSession(ctx, s.pool, stripeSessionID)
	if err != nil {
		return fmt.Errorf("claim checkout session: %w", err)
	}
	if !claimed {
		return ErrCheckoutSessionClaimed
	}

	return nil
}

// FinalizeCheckoutClaim sets tenant_id on a claimed session and creates billing records.
// Called after registration succeeds. Best-effort — errors are logged but don't fail registration.
func (s *CheckoutService) FinalizeCheckoutClaim(ctx context.Context, stripeSessionID string, tenantID uuid.UUID, plan string, interval string) {
	// Set tenant_id on the claimed checkout session
	if err := s.billingRepo.UpdateClaimedCheckoutTenant(ctx, s.pool, stripeSessionID, tenantID); err != nil {
		slog.Error("failed to update checkout session tenant", "session_id", stripeSessionID, "error", err)
	}

	// Get Stripe session to extract customer/subscription IDs
	sess, err := session.Get(stripeSessionID, nil)
	if err != nil {
		slog.Error("failed to get Stripe session for finalization", "session_id", stripeSessionID, "error", err)
		return
	}

	// Create billing customer record
	if sess.Customer != nil && sess.Customer.ID != "" {
		if err := s.billingRepo.CreateBillingCustomer(ctx, s.pool, tenantID, sess.Customer.ID); err != nil {
			slog.Error("failed to create billing customer", "tenant_id", tenantID, "error", err)
		}
	}

	// Create initial subscription record
	if sess.Subscription != nil && sess.Subscription.ID != "" {
		sub := &model.BillingSubscription{
			TenantID:             tenantID,
			StripeSubscriptionID: sess.Subscription.ID,
			StripeCustomerID:     sess.Customer.ID,
			Plan:                 plan,
			BillingInterval:      interval,
			Status:               string(sess.Subscription.Status),
		}
		if sess.Subscription.TrialEnd > 0 {
			t := time.Unix(sess.Subscription.TrialEnd, 0)
			sub.TrialEnd = &t
		}
		if err := s.billingRepo.UpsertSubscription(ctx, s.pool, sub); err != nil {
			slog.Error("failed to create initial subscription", "tenant_id", tenantID, "error", err)
		}
	}
}

// GetSubscription returns the current subscription status for a tenant.
// Falls back to tenant plan info for tenants without a Stripe subscription (license/free).
func (s *CheckoutService) GetSubscription(ctx context.Context, tenantID uuid.UUID, tenantPlan string, tenantSettings json.RawMessage) (*model.SubscriptionStatus, error) {
	var sub *model.BillingSubscription
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		sub, e = s.billingRepo.GetSubscriptionByTenant(ctx, tx)
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	// Parse limits from tenant settings
	var limits *model.LicenseLimits
	if tenantSettings != nil {
		var ps struct {
			Limits *model.LicenseLimits `json:"limits,omitempty"`
		}
		if json.Unmarshal(tenantSettings, &ps) == nil {
			limits = ps.Limits
		}
	}

	if sub != nil {
		return &model.SubscriptionStatus{
			Plan:             sub.Plan,
			Status:           sub.Status,
			BillingInterval:  sub.BillingInterval,
			TrialEnd:         sub.TrialEnd,
			CurrentPeriodEnd: sub.CurrentPeriodEnd,
			CanceledAt:       sub.CanceledAt,
			Limits:           limits,
		}, nil
	}

	// No Stripe subscription — return status from tenant plan
	return &model.SubscriptionStatus{
		Plan:   tenantPlan,
		Status: "active",
		Limits: limits,
	}, nil
}

// PlanLimitsJSON returns the plan limits as a JSON-encoded settings map,
// suitable for storing in tenant.settings.
func PlanLimitsJSON(plan *config.PlanConfig) json.RawMessage {
	settings := map[string]any{
		"max_users":          plan.Limits.MaxUsers,
		"max_orders_monthly": plan.Limits.MaxOrdersMonthly,
		"max_integrations":   plan.Limits.MaxIntegrations,
	}
	b, _ := json.Marshal(settings)
	return b
}
