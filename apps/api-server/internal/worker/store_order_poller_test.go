package worker

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ---------------------------------------------------------------------------
// NewShoperOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewShoperOrderPoller_Name(t *testing.T) {
	p := NewShoperOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "shoper_order_poller", p.Name())
}

func TestNewShoperOrderPoller_Interval(t *testing.T) {
	p := NewShoperOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 60*time.Second, p.Interval())
}

// ---------------------------------------------------------------------------
// NewPrestaShopOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewPrestaShopOrderPoller_Name(t *testing.T) {
	p := NewPrestaShopOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "prestashop_order_poller", p.Name())
}

func TestNewPrestaShopOrderPoller_Interval(t *testing.T) {
	p := NewPrestaShopOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 90*time.Second, p.Interval())
}

// ---------------------------------------------------------------------------
// NewShopifyOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewShopifyOrderPoller_Name(t *testing.T) {
	p := NewShopifyOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "shopify_order_poller", p.Name())
}

func TestNewShopifyOrderPoller_Interval(t *testing.T) {
	p := NewShopifyOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 60*time.Second, p.Interval())
}

// ---------------------------------------------------------------------------
// All store pollers have custom mappers
// ---------------------------------------------------------------------------

func TestStorePollers_HaveCustomMappers(t *testing.T) {
	shoper := NewShoperOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	prestashop := NewPrestaShopOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	shopify := NewShopifyOrderPoller(nil, nil, nil, nil, nil, slog.Default())

	assert.NotNil(t, shoper.mapOrder, "Shoper poller should have a custom order mapper")
	assert.NotNil(t, prestashop.mapOrder, "PrestaShop poller should have a custom order mapper")
	assert.NotNil(t, shopify.mapOrder, "Shopify poller should have a custom order mapper")
}

// ---------------------------------------------------------------------------
// storeOrderMapper — basic mapping
// ---------------------------------------------------------------------------

func TestStoreOrderMapper_BasicMapping(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "shoper",
	}

	orderedAt := time.Date(2025, 5, 10, 8, 0, 0, 0, time.UTC)
	mo := integration.MarketplaceOrder{
		ExternalID:    "SHOPER-555",
		CustomerName:  "Tomasz Mazur",
		CustomerEmail: "tomasz@example.com",
		CustomerPhone: "+48600700800",
		TotalAmount:   79.99,
		Currency:      "PLN",
		PaymentStatus: "paid",
		PaymentMethod: "przelewy24",
		OrderedAt:     orderedAt,
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "s-item-1", Name: "T-shirt", Quantity: 1, UnitPrice: 79.99},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Tomasz Mazur",
			Street:     "ul. Kwiatowa 3",
			City:       "Poznan",
			PostalCode: "60-001",
			Country:    "PL",
		},
		RawData: map[string]any{
			"shoper_order_id": "12345",
		},
	}

	mapper := storeOrderMapper("shoper")
	req := integration.MarketplaceOrderToCreateRequest(mo, "shoper", integrationID)
	order := mapper(mo, ti, req)

	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Tomasz Mazur", order.CustomerName)
	assert.Equal(t, "tomasz@example.com", *order.CustomerEmail)
	assert.Equal(t, 79.99, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "shoper", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.NotNil(t, order.Tags)
	assert.Empty(t, order.Tags)
}

func TestStoreOrderMapper_DefaultPaymentStatusIsPending(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "STORE-NOPAY",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	mapper := storeOrderMapper("prestashop")
	req := integration.MarketplaceOrderToCreateRequest(mo, "prestashop", ti.IntegrationID)
	order := mapper(mo, ti, req)

	assert.Equal(t, "pending", order.PaymentStatus)
}

// ---------------------------------------------------------------------------
// storeOrderMapper — store-specific metadata
// ---------------------------------------------------------------------------

func TestStoreOrderMapper_MetadataIncludesStoreOrderID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}

	tests := []struct {
		provider string
		rawKey   string
		rawValue any
	}{
		{"shoper", "shoper_order_id", "SH-001"},
		{"prestashop", "prestashop_order_id", "PS-002"},
		{"shopify", "shopify_order_id", "SPF-003"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			mo := integration.MarketplaceOrder{
				ExternalID:   "EXT-" + tt.provider,
				CustomerName: "Test",
				TotalAmount:  10.0,
				Currency:     "PLN",
				RawData: map[string]any{
					tt.rawKey: tt.rawValue,
				},
			}
			mapper := storeOrderMapper(tt.provider)
			req := integration.MarketplaceOrderToCreateRequest(mo, tt.provider, ti.IntegrationID)
			order := mapper(mo, ti, req)

			var metadata map[string]any
			require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
			assert.Equal(t, "EXT-"+tt.provider, metadata["external_id"])
			assert.Equal(t, tt.rawValue, metadata[tt.provider+"_order_id"])
		})
	}
}

func TestStoreOrderMapper_MetadataWithoutStoreOrderID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-NOMETA",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData:      nil,
	}
	mapper := storeOrderMapper("shoper")
	req := integration.MarketplaceOrderToCreateRequest(mo, "shoper", ti.IntegrationID)
	order := mapper(mo, ti, req)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "EXT-NOMETA", metadata["external_id"])
	// shoper_order_id should be nil when RawData is nil
	assert.Nil(t, metadata["shoper_order_id"])
}

// ---------------------------------------------------------------------------
// storeOrderMapper — customer note from RawData
// ---------------------------------------------------------------------------

func TestStoreOrderMapper_CustomerNoteFromRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-NOTE",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"customer_note":   "Prosze o szybka wysylke",
			"shoper_order_id": "123",
		},
	}
	mapper := storeOrderMapper("shoper")
	req := integration.MarketplaceOrderToCreateRequest(mo, "shoper", ti.IntegrationID)
	order := mapper(mo, ti, req)

	require.NotNil(t, order.Notes)
	assert.Equal(t, "Prosze o szybka wysylke", *order.Notes)
}

func TestStoreOrderMapper_EmptyCustomerNoteIgnored(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-EMPTYNOTE",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"customer_note": "",
		},
	}
	mapper := storeOrderMapper("shopify")
	req := integration.MarketplaceOrderToCreateRequest(mo, "shopify", ti.IntegrationID)
	order := mapper(mo, ti, req)

	assert.Nil(t, order.Notes, "empty customer_note should not set Notes")
}

func TestStoreOrderMapper_NoCustomerNote(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-NONOTE",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"other_field": "value",
		},
	}
	mapper := storeOrderMapper("prestashop")
	req := integration.MarketplaceOrderToCreateRequest(mo, "prestashop", ti.IntegrationID)
	order := mapper(mo, ti, req)

	assert.Nil(t, order.Notes)
}

// ---------------------------------------------------------------------------
// storeOrderMapper — address and items JSON marshaling
// ---------------------------------------------------------------------------

func TestStoreOrderMapper_ShippingAddressJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-ADDR",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Test User",
			Street:     "ul. Polna 15",
			City:       "Wroclaw",
			PostalCode: "50-001",
			Country:    "PL",
		},
	}
	mapper := storeOrderMapper("shoper")
	req := integration.MarketplaceOrderToCreateRequest(mo, "shoper", ti.IntegrationID)
	order := mapper(mo, ti, req)

	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Wroclaw", addr.City)
	assert.Equal(t, "50-001", addr.PostalCode)
}

func TestStoreOrderMapper_ItemsJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-ITEMS",
		CustomerName: "Test",
		TotalAmount:  50.0,
		Currency:     "PLN",
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "i1", Name: "Product A", Quantity: 5, UnitPrice: 10.0},
		},
	}
	mapper := storeOrderMapper("shopify")
	req := integration.MarketplaceOrderToCreateRequest(mo, "shopify", ti.IntegrationID)
	order := mapper(mo, ti, req)

	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 1)
	assert.Equal(t, "Product A", items[0].Name)
	assert.Equal(t, 5, items[0].Quantity)
}
