package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// BillingRepository implements BillingRepo using SECURITY DEFINER functions
// for all billing operations (no RLS context during checkout/webhook flows).
type BillingRepository struct{}

func NewBillingRepository() *BillingRepository {
	return &BillingRepository{}
}

// CreateCheckoutSession stores a new checkout session (SECURITY DEFINER).
func (r *BillingRepository) CreateCheckoutSession(ctx context.Context, pool *pgxpool.Pool, stripeSessionID, plan, interval string) error {
	_, err := pool.Exec(ctx,
		`SELECT billing_create_checkout_session($1, $2, $3)`,
		stripeSessionID, plan, interval)
	if err != nil {
		return fmt.Errorf("create checkout session: %w", err)
	}
	return nil
}

// CompleteCheckoutSession marks a session as completed with the customer email (SECURITY DEFINER).
func (r *BillingRepository) CompleteCheckoutSession(ctx context.Context, pool *pgxpool.Pool, stripeSessionID, email string) (bool, error) {
	var completed bool
	err := pool.QueryRow(ctx,
		`SELECT billing_complete_checkout_session($1, $2)`,
		stripeSessionID, email).Scan(&completed)
	if err != nil {
		return false, fmt.Errorf("complete checkout session: %w", err)
	}
	return completed, nil
}

// GetCheckoutSession retrieves a checkout session by Stripe session ID (SECURITY DEFINER).
func (r *BillingRepository) GetCheckoutSession(ctx context.Context, pool *pgxpool.Pool, stripeSessionID string) (*model.BillingCheckoutSession, error) {
	var s model.BillingCheckoutSession
	err := pool.QueryRow(ctx,
		`SELECT id, stripe_session_id, plan, billing_interval, email, status, tenant_id, created_at
		 FROM billing_get_checkout_session($1)`, stripeSessionID).
		Scan(&s.ID, &s.StripeSessionID, &s.Plan, &s.BillingInterval, &s.Email, &s.Status, &s.TenantID, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get checkout session: %w", err)
	}
	return &s, nil
}

// ClaimCheckoutSession atomically marks a completed session as registered (SECURITY DEFINER).
// Pre-registration: does NOT set tenant_id (use UpdateClaimedCheckoutTenant after registration).
func (r *BillingRepository) ClaimCheckoutSession(ctx context.Context, pool *pgxpool.Pool, stripeSessionID string) (bool, error) {
	var claimed bool
	err := pool.QueryRow(ctx,
		`SELECT billing_claim_checkout_session($1)`,
		stripeSessionID).Scan(&claimed)
	if err != nil {
		return false, fmt.Errorf("claim checkout session: %w", err)
	}
	return claimed, nil
}

// UpdateClaimedCheckoutTenant sets tenant_id on a claimed (registered) checkout session (SECURITY DEFINER).
func (r *BillingRepository) UpdateClaimedCheckoutTenant(ctx context.Context, pool *pgxpool.Pool, stripeSessionID string, tenantID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`SELECT billing_update_checkout_tenant($1, $2)`,
		stripeSessionID, tenantID)
	if err != nil {
		return fmt.Errorf("update checkout tenant: %w", err)
	}
	return nil
}

// CreateBillingCustomer stores a tenant ↔ Stripe customer mapping (SECURITY DEFINER, upserts on conflict).
func (r *BillingRepository) CreateBillingCustomer(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, stripeCustomerID string) error {
	_, err := pool.Exec(ctx,
		`SELECT billing_create_customer($1, $2)`,
		tenantID, stripeCustomerID)
	if err != nil {
		return fmt.Errorf("create billing customer: %w", err)
	}
	return nil
}

// GetCustomerByStripeID finds a billing customer by Stripe customer ID (SECURITY DEFINER).
func (r *BillingRepository) GetCustomerByStripeID(ctx context.Context, pool *pgxpool.Pool, stripeCustomerID string) (*model.BillingCustomer, error) {
	var c model.BillingCustomer
	err := pool.QueryRow(ctx,
		`SELECT id, tenant_id, stripe_customer_id, created_at
		 FROM billing_get_customer_by_stripe_id($1)`, stripeCustomerID).
		Scan(&c.ID, &c.TenantID, &c.StripeCustomerID, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer by stripe id: %w", err)
	}
	return &c, nil
}

// UpsertSubscription creates or updates a subscription by stripe_subscription_id (SECURITY DEFINER).
func (r *BillingRepository) UpsertSubscription(ctx context.Context, pool *pgxpool.Pool, sub *model.BillingSubscription) error {
	_, err := pool.Exec(ctx,
		`SELECT billing_upsert_subscription($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		sub.TenantID, sub.StripeSubscriptionID, sub.StripeCustomerID,
		sub.Plan, sub.BillingInterval, sub.Status,
		sub.TrialEnd, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}
	return nil
}

// UpdateSubscriptionByStripeID updates subscription status and period fields (SECURITY DEFINER).
func (r *BillingRepository) UpdateSubscriptionByStripeID(ctx context.Context, pool *pgxpool.Pool, stripeSubID, status string, periodStart, periodEnd, canceledAt *time.Time) error {
	_, err := pool.Exec(ctx,
		`SELECT billing_update_subscription_by_stripe_id($1, $2, $3, $4, $5)`,
		stripeSubID, status, periodStart, periodEnd, canceledAt)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

// SyncTenantPlan updates tenant settings with subscription status (SECURITY DEFINER).
func (r *BillingRepository) SyncTenantPlan(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, settingsJSON []byte) error {
	_, err := pool.Exec(ctx,
		`SELECT billing_sync_tenant_plan($1, $2::jsonb)`,
		tenantID, settingsJSON)
	if err != nil {
		return fmt.Errorf("sync tenant plan: %w", err)
	}
	return nil
}

// GetSubscriptionByTenant returns the active subscription for the current RLS tenant.
func (r *BillingRepository) GetSubscriptionByTenant(ctx context.Context, tx pgx.Tx) (*model.BillingSubscription, error) {
	var s model.BillingSubscription
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, stripe_subscription_id, stripe_customer_id, plan, billing_interval,
				status, trial_end, current_period_start, current_period_end, canceled_at, created_at, updated_at
		 FROM billing_subscriptions
		 ORDER BY created_at DESC LIMIT 1`).
		Scan(&s.ID, &s.TenantID, &s.StripeSubscriptionID, &s.StripeCustomerID,
			&s.Plan, &s.BillingInterval, &s.Status,
			&s.TrialEnd, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CanceledAt,
			&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return &s, nil
}
