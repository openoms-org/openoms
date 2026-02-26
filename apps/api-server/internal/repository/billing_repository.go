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
// for pre-registration operations and direct SQL for tenant-scoped operations.
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
func (r *BillingRepository) ClaimCheckoutSession(ctx context.Context, pool *pgxpool.Pool, stripeSessionID string, tenantID uuid.UUID) (bool, error) {
	var claimed bool
	err := pool.QueryRow(ctx,
		`SELECT billing_claim_checkout_session($1, $2)`,
		stripeSessionID, tenantID).Scan(&claimed)
	if err != nil {
		return false, fmt.Errorf("claim checkout session: %w", err)
	}
	return claimed, nil
}

// CreateBillingCustomer stores a tenant ↔ Stripe customer mapping (RLS, needs tx).
func (r *BillingRepository) CreateBillingCustomer(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, stripeCustomerID string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO billing_customers (tenant_id, stripe_customer_id) VALUES ($1, $2)`,
		tenantID, stripeCustomerID)
	if err != nil {
		return fmt.Errorf("create billing customer: %w", err)
	}
	return nil
}

// CreateSubscription stores a new billing subscription (RLS, needs tx).
func (r *BillingRepository) CreateSubscription(ctx context.Context, tx pgx.Tx, sub *model.BillingSubscription) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO billing_subscriptions
			(tenant_id, stripe_subscription_id, stripe_customer_id, plan, billing_interval, status, trial_end, current_period_start, current_period_end)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		sub.TenantID, sub.StripeSubscriptionID, sub.StripeCustomerID,
		sub.Plan, sub.BillingInterval, sub.Status,
		sub.TrialEnd, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}
	return nil
}

// UpdateSubscriptionByStripeID updates subscription status and period fields (SECURITY DEFINER via pool — webhook context).
func (r *BillingRepository) UpdateSubscriptionByStripeID(ctx context.Context, pool *pgxpool.Pool, stripeSubID, status string, periodStart, periodEnd, canceledAt *time.Time) error {
	_, err := pool.Exec(ctx,
		`UPDATE billing_subscriptions
		 SET status = $1, current_period_start = COALESCE($2, current_period_start),
			 current_period_end = COALESCE($3, current_period_end),
			 canceled_at = COALESCE($4, canceled_at), updated_at = now()
		 WHERE stripe_subscription_id = $5`,
		status, periodStart, periodEnd, canceledAt, stripeSubID)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
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

// GetCustomerByStripeID finds a billing customer by Stripe customer ID (pool — webhook context).
func (r *BillingRepository) GetCustomerByStripeID(ctx context.Context, pool *pgxpool.Pool, stripeCustomerID string) (*model.BillingCustomer, error) {
	var c model.BillingCustomer
	err := pool.QueryRow(ctx,
		`SELECT id, tenant_id, stripe_customer_id, created_at
		 FROM billing_customers WHERE stripe_customer_id = $1`, stripeCustomerID).
		Scan(&c.ID, &c.TenantID, &c.StripeCustomerID, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get customer by stripe id: %w", err)
	}
	return &c, nil
}
