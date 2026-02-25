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
// NewAllegroOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewAllegroOrderPoller_Name(t *testing.T) {
	p := NewAllegroOrderPoller(nil, nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "allegro_order_poller", p.Name())
}

func TestNewAllegroOrderPoller_Interval(t *testing.T) {
	p := NewAllegroOrderPoller(nil, nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 45*time.Second, p.Interval())
}

func TestNewAllegroOrderPoller_HasCustomMapper(t *testing.T) {
	p := NewAllegroOrderPoller(nil, nil, nil, nil, nil, nil, slog.Default())
	assert.NotNil(t, p.mapOrder, "Allegro poller should have a custom order mapper")
}

// ---------------------------------------------------------------------------
// allegroOrderMapper — basic mapping
// ---------------------------------------------------------------------------

func TestAllegroOrderMapper_BasicMapping(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "allegro",
	}

	orderedAt := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	mo := integration.MarketplaceOrder{
		ExternalID:    "ALLEGRO-12345",
		CustomerName:  "Jan Kowalski",
		CustomerEmail: "jan@example.com",
		CustomerPhone: "+48123456789",
		TotalAmount:   299.99,
		Currency:      "PLN",
		PaymentStatus: "paid",
		PaymentMethod: "PAYU",
		OrderedAt:     orderedAt,
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "item-1", Name: "Filtr oleju", Quantity: 1, UnitPrice: 49.99},
			{ExternalID: "item-2", Name: "Klocki hamulcowe", Quantity: 2, UnitPrice: 125.00},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Jan Kowalski",
			Street:     "ul. Testowa 5",
			City:       "Krakow",
			PostalCode: "30-001",
			Country:    "PL",
			Phone:      "+48111222333",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", integrationID)
	order := allegroOrderMapper(mo, ti, req)

	// Core fields
	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Jan Kowalski", order.CustomerName)
	assert.Equal(t, "jan@example.com", *order.CustomerEmail)
	assert.Equal(t, "+48123456789", *order.CustomerPhone)
	assert.Equal(t, 299.99, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "allegro", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.Equal(t, "PAYU", *order.PaymentMethod)
	assert.NotNil(t, order.OrderedAt)
	assert.Equal(t, orderedAt, *order.OrderedAt)

	// Tags initialized as empty slice
	assert.NotNil(t, order.Tags)
	assert.Empty(t, order.Tags)

	// ExternalID
	require.NotNil(t, order.ExternalID)
	assert.Equal(t, "ALLEGRO-12345", *order.ExternalID)
}

// ---------------------------------------------------------------------------
// allegroOrderMapper — default status and payment_status
// ---------------------------------------------------------------------------

func TestAllegroOrderMapper_DefaultStatusIsNew(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-1",
		CustomerName: "Test User",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	assert.Equal(t, "new", order.Status)
}

func TestAllegroOrderMapper_DefaultPaymentStatusIsPending(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-2",
		CustomerName: "Anna Nowak",
		TotalAmount:  20.0,
		Currency:     "PLN",
		// PaymentStatus intentionally empty
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	assert.Equal(t, "pending", order.PaymentStatus)
}

func TestAllegroOrderMapper_PaymentStatusFromRequest(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:    "EXT-3",
		CustomerName:  "Test",
		TotalAmount:   10.0,
		Currency:      "PLN",
		PaymentStatus: "paid",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	assert.Equal(t, "paid", order.PaymentStatus)
}

// ---------------------------------------------------------------------------
// allegroOrderMapper — address and items JSON marshaling
// ---------------------------------------------------------------------------

func TestAllegroOrderMapper_ShippingAddressJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-ADDR",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Jan Kowalski",
			Street:     "ul. Dluga 10/2",
			City:       "Gdansk",
			PostalCode: "80-001",
			Country:    "PL",
			Phone:      "+48999888777",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	require.NotNil(t, order.ShippingAddress)
	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Jan Kowalski", addr.Name)
	assert.Equal(t, "ul. Dluga 10/2", addr.Street)
	assert.Equal(t, "Gdansk", addr.City)
	assert.Equal(t, "80-001", addr.PostalCode)
	assert.Equal(t, "PL", addr.Country)
	assert.Equal(t, "+48999888777", addr.Phone)
}

func TestAllegroOrderMapper_ItemsJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-ITEMS",
		CustomerName: "Test",
		TotalAmount:  200.0,
		Currency:     "PLN",
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "A1", Name: "Filtr", Quantity: 2, UnitPrice: 50.0},
			{ExternalID: "A2", Name: "Olej", Quantity: 1, UnitPrice: 100.0},
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	require.NotNil(t, order.Items)
	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 2)
	assert.Equal(t, "Filtr", items[0].Name)
	assert.Equal(t, 2, items[0].Quantity)
	assert.Equal(t, "Olej", items[1].Name)
}

// ---------------------------------------------------------------------------
// allegroOrderMapper — Allegro-specific RawData (delivery_method, pickup_point)
// ---------------------------------------------------------------------------

func TestAllegroOrderMapper_RawDataDeliveryMethodAndPickupPoint(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-RAW",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"delivery_method_name": "InPost Paczkomaty 24/7",
			"pickup_point_id":      "KRA04A",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	require.NotNil(t, order.DeliveryMethod)
	assert.Equal(t, "InPost Paczkomaty 24/7", *order.DeliveryMethod)
	require.NotNil(t, order.PickupPointID)
	assert.Equal(t, "KRA04A", *order.PickupPointID)
}

func TestAllegroOrderMapper_RawDataWithoutDeliveryFields(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-NODELIV",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"some_other_field": "value",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	assert.Nil(t, order.DeliveryMethod)
	assert.Nil(t, order.PickupPointID)
}

func TestAllegroOrderMapper_NilRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-NILRAW",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData:      nil,
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	assert.Nil(t, order.DeliveryMethod)
	assert.Nil(t, order.PickupPointID)
}

// ---------------------------------------------------------------------------
// allegroOrderMapper — metadata includes external_id
// ---------------------------------------------------------------------------

func TestAllegroOrderMapper_MetadataContainsExternalID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "ALLEGRO-META-99",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "ALLEGRO-META-99", metadata["external_id"])
}

// ---------------------------------------------------------------------------
// allegroOrderMapper — phone fallback from shipping address
// ---------------------------------------------------------------------------

func TestAllegroOrderMapper_PhoneFallbackFromShippingAddress(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-PHONE",
		CustomerName: "Test User",
		// CustomerPhone is empty
		TotalAmount: 10.0,
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
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	require.NotNil(t, order.CustomerPhone)
	assert.Equal(t, "+48555666777", *order.CustomerPhone)
}

func TestAllegroOrderMapper_BuyerPhoneTakesPrecedence(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:    "EXT-PHONE2",
		CustomerName:  "Test User",
		CustomerPhone: "+48111222333",
		TotalAmount:   10.0,
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
	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(mo, ti, req)

	require.NotNil(t, order.CustomerPhone)
	assert.Equal(t, "+48111222333", *order.CustomerPhone, "buyer phone should take precedence over shipping address phone")
}
