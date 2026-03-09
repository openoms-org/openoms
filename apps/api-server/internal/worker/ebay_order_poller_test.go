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
// NewEbayOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewEbayOrderPoller(t *testing.T) {
	p := NewEbayOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "ebay_order_poller", p.Name())
	assert.Equal(t, 90*time.Second, p.Interval())
	assert.NotNil(t, p.mapOrder, "eBay poller should have a custom order mapper")
}

// ---------------------------------------------------------------------------
// ebayOrderMapper — basic mapping
// ---------------------------------------------------------------------------

func TestEbayOrderMapper(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "ebay",
	}

	orderedAt := time.Date(2025, 8, 20, 14, 0, 0, 0, time.UTC)
	mo := integration.MarketplaceOrder{
		ExternalID:    "EBAY-ORDER-12345",
		CustomerName:  "Anna Nowak",
		CustomerEmail: "anna@example.com",
		CustomerPhone: "+48600700800",
		TotalAmount:   159.99,
		Currency:      "PLN",
		PaymentStatus: "paid",
		PaymentMethod: "PayPal",
		OrderedAt:     orderedAt,
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "item-1", Name: "Kabel USB-C", Quantity: 2, UnitPrice: 29.99},
			{ExternalID: "item-2", Name: "Etui na telefon", Quantity: 1, UnitPrice: 100.01},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Anna Nowak",
			Street:     "ul. Kwiatowa 15",
			City:       "Poznan",
			PostalCode: "60-001",
			Country:    "PL",
			Phone:      "+48111222333",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", integrationID)
	order := ebayOrderMapper(mo, ti, req)

	// Core fields
	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Anna Nowak", order.CustomerName)
	assert.Equal(t, "anna@example.com", *order.CustomerEmail)
	assert.Equal(t, "+48600700800", *order.CustomerPhone)
	assert.Equal(t, 159.99, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "ebay", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.Equal(t, "PayPal", *order.PaymentMethod)
	assert.NotNil(t, order.OrderedAt)
	assert.Equal(t, orderedAt, *order.OrderedAt)

	// ExternalID
	require.NotNil(t, order.ExternalID)
	assert.Equal(t, "EBAY-ORDER-12345", *order.ExternalID)

	// Tags initialized as empty slice
	assert.NotNil(t, order.Tags)
	assert.Empty(t, order.Tags)

	// Items
	require.NotNil(t, order.Items)
	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 2)
	assert.Equal(t, "Kabel USB-C", items[0].Name)
	assert.Equal(t, 2, items[0].Quantity)

	// Shipping address
	require.NotNil(t, order.ShippingAddress)
	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Anna Nowak", addr.Name)
	assert.Equal(t, "ul. Kwiatowa 15", addr.Street)
	assert.Equal(t, "Poznan", addr.City)
	assert.Equal(t, "60-001", addr.PostalCode)
	assert.Equal(t, "PL", addr.Country)

	// Metadata contains external_id
	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "EBAY-ORDER-12345", metadata["external_id"])
}

// ---------------------------------------------------------------------------
// ebayOrderMapper — PaymentStatus fallback
// ---------------------------------------------------------------------------

func TestEbayOrderMapper_PaymentFallback(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EBAY-NOPAY",
		CustomerName: "Test User",
		TotalAmount:  50.0,
		Currency:     "EUR",
		// PaymentStatus intentionally empty
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	order := ebayOrderMapper(mo, ti, req)

	assert.Equal(t, "pending", order.PaymentStatus)
}

// ---------------------------------------------------------------------------
// ebayOrderMapper — phone fallback from shipping address
// ---------------------------------------------------------------------------

func TestEbayOrderMapper_PhoneFallback(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EBAY-PHONE",
		CustomerName: "Test User",
		// CustomerPhone is empty
		TotalAmount: 30.0,
		Currency:    "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Test",
			Street:     "ul. Test 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
			Phone:      "+48555666777",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	order := ebayOrderMapper(mo, ti, req)

	require.NotNil(t, order.CustomerPhone)
	assert.Equal(t, "+48555666777", *order.CustomerPhone)
}

func TestEbayOrderMapper_BuyerPhoneTakesPrecedence(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:    "EBAY-PHONE2",
		CustomerName:  "Test User",
		CustomerPhone: "+48111222333",
		TotalAmount:   30.0,
		Currency:      "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Test",
			Street:     "ul. Test 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
			Phone:      "+48555666777",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	order := ebayOrderMapper(mo, ti, req)

	require.NotNil(t, order.CustomerPhone)
	assert.Equal(t, "+48111222333", *order.CustomerPhone, "buyer phone should take precedence over shipping address phone")
}

// ---------------------------------------------------------------------------
// ebayOrderMapper — legacy order ID in metadata
// ---------------------------------------------------------------------------

func TestEbayOrderMapper_LegacyOrderID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EBAY-LEGACY",
		CustomerName: "Test User",
		TotalAmount:  25.0,
		Currency:     "USD",
		RawData: map[string]any{
			"ebay_legacy_order_id": "110-12345-67890",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	order := ebayOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "EBAY-LEGACY", metadata["external_id"])
	assert.Equal(t, "110-12345-67890", metadata["ebay_legacy_order_id"])
}

func TestEbayOrderMapper_NoLegacyOrderID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EBAY-NOLEGACY",
		CustomerName: "Test User",
		TotalAmount:  25.0,
		Currency:     "USD",
		RawData: map[string]any{
			"some_other_field": "value",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	order := ebayOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "EBAY-NOLEGACY", metadata["external_id"])
	_, hasLegacy := metadata["ebay_legacy_order_id"]
	assert.False(t, hasLegacy, "should not have ebay_legacy_order_id when not in RawData")
}

func TestEbayOrderMapper_NilRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EBAY-NILRAW",
		CustomerName: "Test User",
		TotalAmount:  25.0,
		Currency:     "USD",
		RawData:      nil,
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	order := ebayOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "EBAY-NILRAW", metadata["external_id"])
	_, hasLegacy := metadata["ebay_legacy_order_id"]
	assert.False(t, hasLegacy, "should not have ebay_legacy_order_id when RawData is nil")
}
