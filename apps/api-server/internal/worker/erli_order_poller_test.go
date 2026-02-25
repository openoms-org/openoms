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
// NewErliOrderPoller — Name / Interval / Mapper
// ---------------------------------------------------------------------------

func TestNewErliOrderPoller_Name(t *testing.T) {
	p := NewErliOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "erli_order_poller", p.Name())
}

func TestNewErliOrderPoller_Interval(t *testing.T) {
	p := NewErliOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 60*time.Second, p.Interval())
}

func TestNewErliOrderPoller_HasCustomMapper(t *testing.T) {
	p := NewErliOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.NotNil(t, p.mapOrder, "Erli poller should have a custom order mapper")
}

// ---------------------------------------------------------------------------
// erliOrderMapper — basic mapping
// ---------------------------------------------------------------------------

func TestErliOrderMapper_BasicMapping(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "erli",
	}

	orderedAt := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	mo := integration.MarketplaceOrder{
		ExternalID:    "ERLI-12345",
		CustomerName:  "Jan Kowalski",
		CustomerEmail: "jan@example.com",
		CustomerPhone: "+48123456789",
		TotalAmount:   199.99,
		Currency:      "PLN",
		PaymentStatus: "paid",
		OrderedAt:     orderedAt,
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "item-1", Name: "Produkt A", SKU: "SKU-001", Quantity: 2, UnitPrice: 49.99},
			{ExternalID: "item-2", Name: "Produkt B", SKU: "SKU-002", Quantity: 1, UnitPrice: 100.01},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Jan Kowalski",
			Street:     "ul. Kwiatowa 5",
			City:       "Krakow",
			PostalCode: "30-001",
			Country:    "PL",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", integrationID)
	order := erliOrderMapper(mo, ti, req)

	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Jan Kowalski", order.CustomerName)
	assert.Equal(t, "jan@example.com", *order.CustomerEmail)
	assert.Equal(t, "+48123456789", *order.CustomerPhone)
	assert.Equal(t, 199.99, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "erli", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	require.NotNil(t, order.OrderedAt)
	assert.Equal(t, orderedAt, *order.OrderedAt)

	require.NotNil(t, order.ExternalID)
	assert.Equal(t, "ERLI-12345", *order.ExternalID)

	assert.NotNil(t, order.Tags)
	assert.Empty(t, order.Tags)
}

// ---------------------------------------------------------------------------
// erliOrderMapper — default status and payment_status
// ---------------------------------------------------------------------------

func TestErliOrderMapper_DefaultStatusIsNew(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-1",
		CustomerName: "Test User",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	assert.Equal(t, "new", order.Status)
}

func TestErliOrderMapper_DefaultPaymentStatusIsPending(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-2",
		CustomerName: "Anna Nowak",
		TotalAmount:  20.0,
		Currency:     "PLN",
		// PaymentStatus intentionally empty
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	assert.Equal(t, "pending", order.PaymentStatus)
}

func TestErliOrderMapper_PaymentStatusFromRequest(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:    "EXT-3",
		CustomerName:  "Test",
		TotalAmount:   10.0,
		Currency:      "PLN",
		PaymentStatus: "paid",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	assert.Equal(t, "paid", order.PaymentStatus)
}

// ---------------------------------------------------------------------------
// erliOrderMapper — address and items JSON marshaling
// ---------------------------------------------------------------------------

func TestErliOrderMapper_ShippingAddressJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-ADDR",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Anna Nowak",
			Street:     "ul. Dluga 10/2",
			City:       "Gdansk",
			PostalCode: "80-001",
			Country:    "PL",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	require.NotNil(t, order.ShippingAddress)
	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Anna Nowak", addr.Name)
	assert.Equal(t, "ul. Dluga 10/2", addr.Street)
	assert.Equal(t, "Gdansk", addr.City)
	assert.Equal(t, "80-001", addr.PostalCode)
	assert.Equal(t, "PL", addr.Country)
}

func TestErliOrderMapper_ItemsJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-ITEMS",
		CustomerName: "Test",
		TotalAmount:  300.0,
		Currency:     "PLN",
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "E1", Name: "Klocki hamulcowe", SKU: "KB-001", Quantity: 2, UnitPrice: 125.0},
			{ExternalID: "E2", Name: "Filtr oleju", SKU: "FO-001", Quantity: 1, UnitPrice: 50.0},
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	require.NotNil(t, order.Items)
	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 2)
	assert.Equal(t, "Klocki hamulcowe", items[0].Name)
	assert.Equal(t, 2, items[0].Quantity)
	assert.Equal(t, "Filtr oleju", items[1].Name)
}

// ---------------------------------------------------------------------------
// erliOrderMapper — Erli-specific metadata (erli_status / oms_status)
// ---------------------------------------------------------------------------

func TestErliOrderMapper_MetadataContainsExternalID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "ERLI-META-99",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "ERLI-META-99", metadata["external_id"])
}

func TestErliOrderMapper_MetadataFromRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:     "ERLI-RAW-1",
		ExternalStatus: "paid",
		CustomerName:   "Test",
		TotalAmount:    10.0,
		Currency:       "PLN",
		RawData: map[string]any{
			"erli_status": "paid",
			"oms_status":  "confirmed",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "paid", metadata["erli_status"])
	assert.Equal(t, "confirmed", metadata["oms_status"])
}

func TestErliOrderMapper_MetadataFallbackFromExternalStatus(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:     "ERLI-FALLBACK-1",
		ExternalStatus: "shipped",
		CustomerName:   "Test",
		TotalAmount:    10.0,
		Currency:       "PLN",
		// No RawData — mapper should derive erli_status from ExternalStatus.
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "shipped", metadata["erli_status"])
	assert.Equal(t, "shipped", metadata["oms_status"])
}

func TestErliOrderMapper_MetadataNoStatusForUnknownExternalStatus(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:     "ERLI-UNK-1",
		ExternalStatus: "unknown_status",
		CustomerName:   "Test",
		TotalAmount:    10.0,
		Currency:       "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "erli", ti.IntegrationID)
	order := erliOrderMapper(mo, ti, req)

	require.NotNil(t, order.Metadata)
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	// Unknown status should not appear in metadata.
	assert.NotContains(t, metadata, "erli_status")
	assert.NotContains(t, metadata, "oms_status")
	// external_id still present.
	assert.Equal(t, "ERLI-UNK-1", metadata["external_id"])
}
