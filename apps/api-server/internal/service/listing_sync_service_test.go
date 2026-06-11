package service

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
