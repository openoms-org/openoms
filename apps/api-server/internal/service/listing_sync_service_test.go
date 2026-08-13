package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// --- applyPriceRule (pure unit) ---

func TestApplyPriceRule(t *testing.T) {
	svc := &ListingSyncService{}
	cases := []struct {
		name     string
		rule     string
		modifier float64
		base     float64
		want     float64
	}{
		{"same/default keeps price", "same", 0, 100, 100},
		{"unknown rule keeps price", "whatever", 50, 100, 100},
		{"markup_pct 10%", "markup_pct", 10, 100, 110},
		{"markup_pct rounds to 2dp", "markup_pct", 15, 19.99, 22.99},
		{"markup_fixed adds", "markup_fixed", 5, 100, 105},
		{"markup_fixed rounds to 2dp", "markup_fixed", 0.005, 10, 10.01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &model.ListingSyncConfig{PriceRule: tc.rule, PriceModifier: tc.modifier}
			assert.InDelta(t, tc.want, svc.applyPriceRule(cfg, tc.base), 0.0001)
		})
	}
}

// --- isPushEligible (the gate the wired workers use) ---

func TestIsPushEligible(t *testing.T) {
	ext := "OFFER-1"
	empty := ""
	cases := []struct {
		name    string
		listing *model.ProductListing
		want    bool
	}{
		{"active + auto + external_id", &model.ProductListing{ExternalID: &ext, Status: "active", StockSyncMode: "auto"}, true},
		{"nil external_id", &model.ProductListing{ExternalID: nil, Status: "active", StockSyncMode: "auto"}, false},
		{"empty external_id", &model.ProductListing{ExternalID: &empty, Status: "active", StockSyncMode: "auto"}, false},
		{"manual mode", &model.ProductListing{ExternalID: &ext, Status: "active", StockSyncMode: "manual"}, false},
		{"inactive status", &model.ProductListing{ExternalID: &ext, Status: "inactive", StockSyncMode: "auto"}, false},
		{"pending status", &model.ProductListing{ExternalID: &ext, Status: "pending", StockSyncMode: "auto"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isPushEligible(tc.listing))
		})
	}
}

// --- feedMetaForProvider ---

func TestFeedMetaForProvider_NoFeed(t *testing.T) {
	// A provider that does not implement AsyncFeedResult yields nil metadata.
	assert.Nil(t, feedMetaForProvider(struct{}{}))
}

// --- dispatchProductOffers ---

type listingSyncTestProvider struct {
	fail bool
}

func (p *listingSyncTestProvider) ProviderName() string { return "test" }
func (p *listingSyncTestProvider) PollOrders(context.Context, string) ([]integration.MarketplaceOrder, string, error) {
	return nil, "", nil
}
func (p *listingSyncTestProvider) GetOrder(context.Context, string) (*integration.MarketplaceOrder, error) {
	return nil, nil
}
func (p *listingSyncTestProvider) PushOffer(_ context.Context, product *model.Product, _ map[string]any) (string, error) {
	if p.fail {
		return "", errors.New("provider rejected offer")
	}
	return "offer-" + product.ID.String(), nil
}
func (p *listingSyncTestProvider) UpdateStock(context.Context, string, int) error { return nil }
func (p *listingSyncTestProvider) UpdatePrice(context.Context, string, float64) error {
	return nil
}

func TestDispatchProductOffers_RecordsExternalIDAfterSuccessfulPush(t *testing.T) {
	productID := uuid.New()
	listing := &model.ProductListing{ID: uuid.New(), ProductID: productID}
	result := &SyncResult{}
	var savedExternalID string
	var failed bool

	(&ListingSyncService{logger: slog.Default()}).dispatchProductOffers(
		context.Background(),
		&model.IntegrationWithCreds{Integration: model.Integration{Provider: "test"}},
		&listingSyncTestProvider{},
		&model.ListingSyncConfig{},
		[]listingPushJob{{listing: listing, product: model.Product{ID: productID}}},
		result,
		func(saved *model.ProductListing, externalID string) {
			assert.Same(t, listing, saved)
			savedExternalID = externalID
		},
		func(*model.ProductListing, string) { failed = true },
	)

	assert.Equal(t, "offer-"+productID.String(), savedExternalID)
	assert.False(t, failed)
	assert.Equal(t, 1, result.ItemsProcessed)
	assert.Zero(t, result.ItemsFailed)
}

func TestDispatchProductOffers_FailedPushIsNotMarkedSynced(t *testing.T) {
	listing := &model.ProductListing{ID: uuid.New(), ProductID: uuid.New()}
	result := &SyncResult{}
	var saved bool
	var failureMessage string

	(&ListingSyncService{logger: slog.Default()}).dispatchProductOffers(
		context.Background(),
		&model.IntegrationWithCreds{Integration: model.Integration{Provider: "test"}},
		&listingSyncTestProvider{fail: true},
		&model.ListingSyncConfig{},
		[]listingPushJob{{listing: listing, product: model.Product{ID: listing.ProductID}}},
		result,
		func(*model.ProductListing, string) { saved = true },
		func(failed *model.ProductListing, errMsg string) {
			assert.Same(t, listing, failed)
			failureMessage = errMsg
		},
	)

	assert.False(t, saved, "a failed push must not take the synced persistence path")
	assert.Equal(t, "provider rejected offer", failureMessage)
	assert.Zero(t, result.ItemsProcessed)
	assert.Equal(t, 1, result.ItemsFailed)
}

// --- construction + setter wiring ---

func TestNewListingSyncService_WiresDependencies(t *testing.T) {
	syncRepo := repository.NewListingSyncRepository()
	productRepo := repository.NewProductRepository()
	listingRepo := repository.NewProductListingRepository()
	auditRepo := repository.NewAuditRepository()
	integrationRepo := repository.NewIntegrationRepository()
	key := []byte("0123456789abcdef0123456789abcdef")

	svc := NewListingSyncService(syncRepo, productRepo, listingRepo, auditRepo, integrationRepo, nil, key, slog.Default())
	require.NotNil(t, svc)
	assert.NotNil(t, svc.integrationRepo, "integrationRepo must be wired")
	assert.Equal(t, key, svc.encryptionKey, "encryptionKey must be wired")
	assert.Nil(t, svc.stockSyncSvc, "stockSyncSvc starts nil until SetStockSyncService")

	stockSvc := &StockSyncService{}
	svc.SetStockSyncService(stockSvc)
	assert.Same(t, stockSvc, svc.stockSyncSvc, "SetStockSyncService must wire the stock owner")
}
