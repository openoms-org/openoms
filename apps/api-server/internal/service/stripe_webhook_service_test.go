package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripeWebhookServiceSyncTenantPlanInvalidatesPlanCache(t *testing.T) {
	tenantID := uuid.New()
	cache := NewPlanCache(5 * time.Minute)
	cache.Set(tenantID, "plus", json.RawMessage(`{"subscription_status":"active"}`))

	repo := &fakeStripeBillingRepo{
		customer: &model.BillingCustomer{
			TenantID:         tenantID,
			StripeCustomerID: "cus_test",
		},
	}
	svc := NewStripeWebhookService("whsec_test", repo, nil, cache)

	svc.syncTenantPlan(context.Background(), "cus_test", "past_due")

	require.True(t, repo.synced)
	assert.Equal(t, tenantID, repo.syncedTenantID)
	assert.JSONEq(t, `{"subscription_status":"past_due"}`, string(repo.syncedSettings))

	_, _, ok := cache.Get(tenantID)
	assert.False(t, ok, "plan cache should be invalidated after billing status changes")
}

type fakeStripeBillingRepo struct {
	customer       *model.BillingCustomer
	synced         bool
	syncedTenantID uuid.UUID
	syncedSettings []byte
}

func (f *fakeStripeBillingRepo) CreateCheckoutSession(context.Context, *pgxpool.Pool, string, string, string) error {
	return nil
}

func (f *fakeStripeBillingRepo) CompleteCheckoutSession(context.Context, *pgxpool.Pool, string, string, model.CheckoutSessionStripeRefs) (bool, error) {
	return false, nil
}

func (f *fakeStripeBillingRepo) GetCheckoutSession(context.Context, *pgxpool.Pool, string) (*model.BillingCheckoutSession, error) {
	return nil, nil
}

func (f *fakeStripeBillingRepo) ClaimCheckoutSession(context.Context, *pgxpool.Pool, string) (bool, error) {
	return false, nil
}

func (f *fakeStripeBillingRepo) UpdateClaimedCheckoutTenant(context.Context, *pgxpool.Pool, string, uuid.UUID) error {
	return nil
}

func (f *fakeStripeBillingRepo) CreateBillingCustomer(context.Context, *pgxpool.Pool, uuid.UUID, string) error {
	return nil
}

func (f *fakeStripeBillingRepo) GetCustomerByStripeID(context.Context, *pgxpool.Pool, string) (*model.BillingCustomer, error) {
	return f.customer, nil
}

func (f *fakeStripeBillingRepo) UpsertSubscription(context.Context, *pgxpool.Pool, *model.BillingSubscription) error {
	return nil
}

func (f *fakeStripeBillingRepo) UpdateSubscriptionByStripeID(context.Context, *pgxpool.Pool, string, string, *time.Time, *time.Time, *time.Time) error {
	return nil
}

func (f *fakeStripeBillingRepo) GetSubscriptionByTenant(context.Context, pgx.Tx) (*model.BillingSubscription, error) {
	return nil, nil
}

func (f *fakeStripeBillingRepo) SyncTenantPlan(_ context.Context, _ *pgxpool.Pool, tenantID uuid.UUID, settingsJSON []byte) error {
	f.synced = true
	f.syncedTenantID = tenantID
	f.syncedSettings = append([]byte(nil), settingsJSON...)
	return nil
}
