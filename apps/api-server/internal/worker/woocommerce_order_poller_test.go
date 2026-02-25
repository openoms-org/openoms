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
// NewWooCommerceOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewWooCommerceOrderPoller_Name(t *testing.T) {
	p := NewWooCommerceOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "woocommerce_order_poller", p.Name())
}

func TestNewWooCommerceOrderPoller_Interval(t *testing.T) {
	p := NewWooCommerceOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 60*time.Second, p.Interval())
}

func TestNewWooCommerceOrderPoller_HasCustomMapper(t *testing.T) {
	p := NewWooCommerceOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.NotNil(t, p.mapOrder, "WooCommerce poller should have a custom order mapper")
}

// ---------------------------------------------------------------------------
// woocommerceOrderMapper — basic mapping
// ---------------------------------------------------------------------------

func TestWooCommerceOrderMapper_BasicMapping(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "woocommerce",
	}

	orderedAt := time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC)
	mo := integration.MarketplaceOrder{
		ExternalID:    "WC-10042",
		CustomerName:  "Katarzyna Lewandowska",
		CustomerEmail: "kasia@example.com",
		CustomerPhone: "+48501502503",
		TotalAmount:   59.90,
		Currency:      "PLN",
		PaymentStatus: "paid",
		PaymentMethod: "blik",
		OrderedAt:     orderedAt,
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "wc-1", Name: "Herbata zielona 100g", Quantity: 2, UnitPrice: 29.95},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Katarzyna Lewandowska",
			Street:     "ul. Herbaciana 7",
			City:       "Lodz",
			PostalCode: "90-001",
			Country:    "PL",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", integrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Katarzyna Lewandowska", order.CustomerName)
	assert.Equal(t, "kasia@example.com", *order.CustomerEmail)
	assert.Equal(t, "+48501502503", *order.CustomerPhone)
	assert.Equal(t, 59.90, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "woocommerce", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.Equal(t, "blik", *order.PaymentMethod)
	assert.NotNil(t, order.Tags)
	assert.Empty(t, order.Tags)
}

func TestWooCommerceOrderMapper_DefaultPaymentStatusIsPending(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-NOPAY",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	assert.Equal(t, "pending", order.PaymentStatus)
}

// ---------------------------------------------------------------------------
// woocommerceOrderMapper — WooCommerce-specific RawData
// ---------------------------------------------------------------------------

func TestWooCommerceOrderMapper_CustomerNoteFromRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-NOTE",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"customer_note":        "Dzwoneczek przy drzwiach nie dziala",
			"woocommerce_order_id": 42,
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	require.NotNil(t, order.Notes)
	assert.Equal(t, "Dzwoneczek przy drzwiach nie dziala", *order.Notes)
}

func TestWooCommerceOrderMapper_EmptyCustomerNoteIgnored(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-EMPTYNOTE",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"customer_note": "",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	assert.Nil(t, order.Notes, "empty customer_note should not set Notes")
}

func TestWooCommerceOrderMapper_WooCommerceOrderIDInMetadata(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-META",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"woocommerce_order_id": float64(9876),
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "WC-META", metadata["external_id"])
	assert.Equal(t, float64(9876), metadata["woocommerce_order_id"])
}

func TestWooCommerceOrderMapper_MetadataWithoutWooCommerceOrderID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-NOWCID",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"other_field": "value",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "WC-NOWCID", metadata["external_id"])
	_, hasWCID := metadata["woocommerce_order_id"]
	assert.False(t, hasWCID, "woocommerce_order_id should not be in metadata when absent from RawData")
}

func TestWooCommerceOrderMapper_NilRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-NILRAW",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData:      nil,
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	assert.Nil(t, order.Notes)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "WC-NILRAW", metadata["external_id"])
}

// ---------------------------------------------------------------------------
// woocommerceOrderMapper — address and items JSON marshaling
// ---------------------------------------------------------------------------

func TestWooCommerceOrderMapper_ShippingAddressJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-ADDR",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Test User",
			Street:     "ul. Mokra 22",
			City:       "Szczecin",
			PostalCode: "70-001",
			Country:    "PL",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Szczecin", addr.City)
	assert.Equal(t, "PL", addr.Country)
}

func TestWooCommerceOrderMapper_ItemsJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "WC-ITEMS",
		CustomerName: "Test",
		TotalAmount:  100.0,
		Currency:     "PLN",
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "w1", Name: "Herbata Earl Grey", Quantity: 3, UnitPrice: 25.0},
			{ExternalID: "w2", Name: "Herbata Jasminowa", Quantity: 1, UnitPrice: 25.0},
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	order := woocommerceOrderMapper(mo, ti, req)

	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 2)
	assert.Equal(t, "Herbata Earl Grey", items[0].Name)
	assert.Equal(t, 3, items[0].Quantity)
}
