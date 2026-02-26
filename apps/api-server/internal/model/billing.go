package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// CheckoutSessionRequest is the body of POST /v1/billing/checkout.
type CheckoutSessionRequest struct {
	PlanID   string `json:"plan_id"`
	Interval string `json:"interval"` // "month" or "year"
}

func (r *CheckoutSessionRequest) Validate() error {
	if r.PlanID == "" {
		return errors.New("plan_id is required")
	}
	if r.Interval != "month" && r.Interval != "year" {
		return errors.New("interval must be 'month' or 'year'")
	}
	return nil
}

// CheckoutSessionResponse is returned by POST /v1/billing/checkout.
type CheckoutSessionResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
}

// CheckoutSessionStatus is returned by GET /v1/billing/checkout/{session_id}.
type CheckoutSessionStatus struct {
	Plan     string         `json:"plan"`
	Interval string         `json:"interval"`
	Email    string         `json:"email,omitempty"`
	Status   string         `json:"status"`
	Limits   *LicenseLimits `json:"limits,omitempty"`
}

// BillingCheckoutSession represents a row in billing_checkout_sessions.
type BillingCheckoutSession struct {
	ID              uuid.UUID  `json:"id"`
	StripeSessionID string     `json:"stripe_session_id"`
	Plan            string     `json:"plan"`
	BillingInterval string     `json:"billing_interval"`
	Email           string     `json:"email,omitempty"`
	Status          string     `json:"status"`
	TenantID        *uuid.UUID `json:"tenant_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// BillingCustomer links a tenant to a Stripe customer.
type BillingCustomer struct {
	ID               uuid.UUID `json:"id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	StripeCustomerID string    `json:"stripe_customer_id"`
	CreatedAt        time.Time `json:"created_at"`
}

// BillingSubscription represents a row in billing_subscriptions.
type BillingSubscription struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	StripeCustomerID     string     `json:"stripe_customer_id"`
	Plan                 string     `json:"plan"`
	BillingInterval      string     `json:"billing_interval"`
	Status               string     `json:"status"`
	TrialEnd             *time.Time `json:"trial_end,omitempty"`
	CurrentPeriodStart   *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CanceledAt           *time.Time `json:"canceled_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// PublicPlanInfo is the frontend-safe view of a plan (no Stripe Price IDs).
type PublicPlanInfo struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	MonthlyAmount int64         `json:"monthly_amount"`
	YearlyAmount  int64         `json:"yearly_amount"`
	Currency      string        `json:"currency"`
	TrialDays     int64         `json:"trial_days"`
	Limits        LicenseLimits `json:"limits"`
	Features      []string      `json:"features"`
}
