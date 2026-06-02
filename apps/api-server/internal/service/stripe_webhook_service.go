package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// ErrWebhookSignature indicates a Stripe signature verification failure (client error, not retryable).
var ErrWebhookSignature = errors.New("webhook signature verification failed")

// ErrWebhookUnprocessable indicates a permanent failure to process the event
// (e.g. a malformed payload). Retrying will never succeed, so the handler
// returns 4xx to stop Stripe from retrying it for days. Transient failures
// (e.g. a database blip) return other errors -> 5xx, which Stripe retries.
var ErrWebhookUnprocessable = errors.New("webhook payload unprocessable")

// StripeWebhookService processes Stripe webhook events.
type StripeWebhookService struct {
	webhookSecret string
	billingRepo   repository.BillingRepo
	pool          *pgxpool.Pool
	planCache     *PlanCache
}

// NewStripeWebhookService creates a new StripeWebhookService.
func NewStripeWebhookService(webhookSecret string, billingRepo repository.BillingRepo, pool *pgxpool.Pool, planCache *PlanCache) *StripeWebhookService {
	return &StripeWebhookService{
		webhookSecret: webhookSecret,
		billingRepo:   billingRepo,
		pool:          pool,
		planCache:     planCache,
	}
}

// HandleEvent verifies and dispatches a Stripe webhook event.
func (s *StripeWebhookService) HandleEvent(ctx context.Context, payload []byte, sigHeader string) error {
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWebhookSignature, err)
	}

	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutCompleted(ctx, event)
	case "customer.subscription.updated":
		return s.handleSubscriptionUpdated(ctx, event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, event)
	case "invoice.payment_failed":
		return s.handlePaymentFailed(ctx, event)
	default:
		slog.Debug("unhandled stripe event type", "type", event.Type)
		return nil
	}
}

func (s *StripeWebhookService) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
		return fmt.Errorf("%w: unmarshal checkout session: %v", ErrWebhookUnprocessable, err)
	}

	email := ""
	if sess.CustomerDetails != nil {
		email = sess.CustomerDetails.Email
	}

	completed, err := s.billingRepo.CompleteCheckoutSession(ctx, s.pool, sess.ID, email, checkoutSessionStripeRefs(&sess))
	if err != nil {
		return fmt.Errorf("complete checkout session: %w", err)
	}
	if !completed {
		slog.Warn("checkout session already completed or not found", "stripe_session_id", sess.ID)
	} else {
		slog.Info("checkout session completed via webhook", "stripe_session_id", sess.ID)
	}
	return nil
}

func (s *StripeWebhookService) handleSubscriptionUpdated(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("%w: unmarshal subscription: %v", ErrWebhookUnprocessable, err)
	}

	status := string(sub.Status)
	var periodStart, periodEnd, canceledAt *time.Time

	// In stripe-go v82, current period is on subscription items, not the subscription itself.
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.CurrentPeriodStart > 0 {
			t := time.Unix(item.CurrentPeriodStart, 0)
			periodStart = &t
		}
		if item.CurrentPeriodEnd > 0 {
			t := time.Unix(item.CurrentPeriodEnd, 0)
			periodEnd = &t
		}
	}
	if sub.CanceledAt > 0 {
		t := time.Unix(sub.CanceledAt, 0)
		canceledAt = &t
	}

	// Use SECURITY DEFINER function to bypass RLS
	if err := s.billingRepo.UpdateSubscriptionByStripeID(ctx, s.pool, sub.ID, status, periodStart, periodEnd, canceledAt); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	// Update tenant plan/settings if customer is linked
	if sub.Customer != nil {
		s.syncTenantPlan(ctx, sub.Customer.ID, string(sub.Status))
	}

	slog.Info("subscription updated via webhook", "stripe_sub_id", sub.ID, "status", status)
	return nil
}

func (s *StripeWebhookService) handleSubscriptionDeleted(ctx context.Context, event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("%w: unmarshal subscription: %v", ErrWebhookUnprocessable, err)
	}

	now := time.Now()
	if err := s.billingRepo.UpdateSubscriptionByStripeID(ctx, s.pool, sub.ID, "canceled", nil, nil, &now); err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}

	// Mark tenant as suspended
	if sub.Customer != nil {
		s.syncTenantPlan(ctx, sub.Customer.ID, "canceled")
	}

	slog.Info("subscription deleted via webhook", "stripe_sub_id", sub.ID)
	return nil
}

func (s *StripeWebhookService) handlePaymentFailed(ctx context.Context, event stripe.Event) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
		return fmt.Errorf("%w: unmarshal invoice: %v", ErrWebhookUnprocessable, err)
	}

	// In stripe-go v82, subscription is under invoice.Parent.SubscriptionDetails.Subscription
	var sub *stripe.Subscription
	if invoice.Parent != nil && invoice.Parent.SubscriptionDetails != nil {
		sub = invoice.Parent.SubscriptionDetails.Subscription
	}
	if sub != nil && sub.ID != "" {
		if err := s.billingRepo.UpdateSubscriptionByStripeID(ctx, s.pool, sub.ID, "past_due", nil, nil, nil); err != nil {
			return fmt.Errorf("mark subscription past_due: %w", err)
		}

		if invoice.Customer != nil && invoice.Customer.ID != "" {
			s.syncTenantPlan(ctx, invoice.Customer.ID, "past_due")
		}

		slog.Warn("payment failed for subscription", "stripe_sub_id", sub.ID)
	}
	return nil
}

// syncTenantPlan updates the tenant's plan status based on subscription state.
// Uses SECURITY DEFINER function to bypass RLS.
func (s *StripeWebhookService) syncTenantPlan(ctx context.Context, stripeCustomerID, subStatus string) {
	customer, err := s.billingRepo.GetCustomerByStripeID(ctx, s.pool, stripeCustomerID)
	if err != nil || customer == nil {
		slog.Debug("no billing customer found for stripe customer", "stripe_customer_id", stripeCustomerID)
		return
	}

	settings := map[string]any{"subscription_status": subStatus}
	settingsJSON, _ := json.Marshal(settings)
	if err := s.billingRepo.SyncTenantPlan(ctx, s.pool, customer.TenantID, settingsJSON); err != nil {
		slog.Error("failed to sync tenant plan status", "tenant_id", customer.TenantID, "error", err)
		return
	}
	if s.planCache != nil {
		s.planCache.Invalidate(customer.TenantID)
	}
}
