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
// NewAmazonOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestNewAmazonOrderPoller_Name(t *testing.T) {
	p := NewAmazonOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, "amazon_order_poller", p.Name())
}

func TestNewAmazonOrderPoller_Interval(t *testing.T) {
	p := NewAmazonOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.Equal(t, 2*time.Minute, p.Interval())
}

func TestNewAmazonOrderPoller_HasCustomMapper(t *testing.T) {
	p := NewAmazonOrderPoller(nil, nil, nil, nil, nil, slog.Default())
	assert.NotNil(t, p.mapOrder, "Amazon poller should have a custom order mapper")
}

// ---------------------------------------------------------------------------
// amazonOrderMapper — basic mapping
// ---------------------------------------------------------------------------

func TestAmazonOrderMapper_BasicMapping(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "amazon",
	}

	orderedAt := time.Date(2025, 3, 20, 14, 0, 0, 0, time.UTC)
	mo := integration.MarketplaceOrder{
		ExternalID:    "AMZ-998877",
		CustomerName:  "Maria Wisniewska",
		CustomerEmail: "maria@example.com",
		CustomerPhone: "+48987654321",
		TotalAmount:   149.99,
		Currency:      "EUR",
		PaymentStatus: "paid",
		PaymentMethod: "amazon_pay",
		OrderedAt:     orderedAt,
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "amz-item-1", Name: "USB Cable", Quantity: 3, UnitPrice: 49.997},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Maria Wisniewska",
			Street:     "Am Marktplatz 7",
			City:       "Berlin",
			PostalCode: "10115",
			Country:    "DE",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", integrationID)
	order := amazonOrderMapper(mo, ti, req)

	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Maria Wisniewska", order.CustomerName)
	assert.Equal(t, "maria@example.com", *order.CustomerEmail)
	assert.Equal(t, 149.99, order.TotalAmount)
	assert.Equal(t, "EUR", order.Currency)
	assert.Equal(t, "amazon", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.NotNil(t, order.Tags)
	assert.Empty(t, order.Tags)
}

func TestAmazonOrderMapper_DefaultPaymentStatusIsPending(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-NO-PAY",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	assert.Equal(t, "pending", order.PaymentStatus)
}

// ---------------------------------------------------------------------------
// amazonOrderMapper — Amazon-specific RawData (fulfillment_channel)
// ---------------------------------------------------------------------------

func TestAmazonOrderMapper_FulfillmentChannelFromRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-FBA",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"fulfillment_channel": "AFN",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	require.NotNil(t, order.DeliveryMethod)
	assert.Equal(t, "AFN", *order.DeliveryMethod)
}

func TestAmazonOrderMapper_MFNFulfillmentChannel(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-MFN",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"fulfillment_channel": "MFN",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	require.NotNil(t, order.DeliveryMethod)
	assert.Equal(t, "MFN", *order.DeliveryMethod)
}

func TestAmazonOrderMapper_NoRawData(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-NORAW",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData:      nil,
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	assert.Nil(t, order.DeliveryMethod)
}

func TestAmazonOrderMapper_RawDataWithoutFulfillmentChannel(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-NOFC",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"other_field": "value",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	assert.Nil(t, order.DeliveryMethod)
}

// ---------------------------------------------------------------------------
// amazonOrderMapper — metadata and JSON marshaling
// ---------------------------------------------------------------------------

func TestAmazonOrderMapper_MetadataContainsExternalID(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-META-42",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "AMZ-META-42", metadata["external_id"])
}

func TestAmazonOrderMapper_ShippingAddressJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-ADDR",
		CustomerName: "Test",
		TotalAmount:  10.0,
		Currency:     "PLN",
		ShippingAddress: model.ShippingAddress{
			Name:       "Test User",
			Street:     "123 Main St",
			City:       "New York",
			PostalCode: "10001",
			Country:    "US",
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "New York", addr.City)
	assert.Equal(t, "US", addr.Country)
}

func TestAmazonOrderMapper_ItemsJSON(t *testing.T) {
	ti := TenantIntegration{TenantID: uuid.New(), IntegrationID: uuid.New()}
	mo := integration.MarketplaceOrder{
		ExternalID:   "AMZ-ITEMS",
		CustomerName: "Test",
		TotalAmount:  30.0,
		Currency:     "PLN",
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "i1", Name: "Widget A", Quantity: 1, UnitPrice: 10.0},
			{ExternalID: "i2", Name: "Widget B", Quantity: 2, UnitPrice: 10.0},
		},
	}
	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	order := amazonOrderMapper(mo, ti, req)

	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 2)
	assert.Equal(t, "Widget A", items[0].Name)
	assert.Equal(t, "Widget B", items[1].Name)
}
