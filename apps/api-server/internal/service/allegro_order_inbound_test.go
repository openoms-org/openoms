package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	allegroint "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func sampleMarketplaceCheckoutForm() integration.MarketplaceOrder {
	return integration.MarketplaceOrder{
		ExternalID:     "cf-7781994292",
		ExternalStatus: "READY_FOR_PROCESSING",
		CustomerName:   "Jan Kowalski",
		CustomerEmail:  "jan@test.pl",
		CustomerPhone:  "+48123456789",
		TotalAmount:    9.99,
		Currency:       "PLN",
		PaymentStatus:  "paid",
		PaymentMethod:  "ONLINE",
		OrderedAt:      time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC),
		ShippingAddress: model.ShippingAddress{
			Name:       "Jan Kowalski",
			Street:     "Marszalkowska 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
			Phone:      "+48999888777",
			Email:      "jan@test.pl",
		},
		Items: []integration.MarketplaceOrderItem{
			{
				ExternalID: "7781994292",
				Name:       "Sandbox widget",
				SKU:        "SKU-SANDBOX",
				Quantity:   1,
				UnitPrice:  9.99,
				TotalPrice: 9.99,
			},
		},
		RawData: map[string]any{
			"delivery_method_id":   "c3066682-97a3-42fe-9eb5-3beeccab840c",
			"delivery_method_name": "Allegro miniKurier24 InPost",
			"pickup_point_id":      "WAW01A",
		},
	}
}

func TestToOMSOrder_MapsCheckoutFormFields(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	mo := sampleMarketplaceCheckoutForm()

	order := allegroint.ToOMSOrder(mo, tenantID, integrationID)

	assert.Equal(t, tenantID, order.TenantID)
	assert.Equal(t, "allegro", order.Source)
	require.NotNil(t, order.ExternalID)
	assert.Equal(t, "cf-7781994292", *order.ExternalID)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Jan Kowalski", order.CustomerName)
	assert.Equal(t, "jan@test.pl", *order.CustomerEmail)
	assert.Equal(t, "+48123456789", *order.CustomerPhone)
	assert.Equal(t, 9.99, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.Equal(t, "ONLINE", *order.PaymentMethod)
	require.NotNil(t, order.DeliveryMethod)
	assert.Equal(t, "Allegro miniKurier24 InPost", *order.DeliveryMethod)
	require.NotNil(t, order.PickupPointID)
	assert.Equal(t, "WAW01A", *order.PickupPointID)

	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 1)
	assert.Equal(t, "Sandbox widget", items[0].Name)

	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Warszawa", addr.City)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "cf-7781994292", metadata["external_id"])
	assert.Equal(t, "c3066682-97a3-42fe-9eb5-3beeccab840c", metadata["delivery_method_id"])
	assert.Equal(t, "Allegro miniKurier24 InPost", metadata["delivery_method_name"])
}

func TestUpsertAllegroCheckoutForm_CreatesThenIdempotent(t *testing.T) {
	repo := newMockOrderRepo()
	tenantID := uuid.New()
	integrationID := uuid.New()
	mo := sampleMarketplaceCheckoutForm()

	createdID, err := upsertAllegroCheckoutForm(context.Background(), nil, repo, mo, tenantID, integrationID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, createdID)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "allegro", repo.created[0].Source)
	require.NotNil(t, repo.created[0].ExternalID)
	assert.Equal(t, "cf-7781994292", *repo.created[0].ExternalID)
	assert.Equal(t, 1, repo.createIfExternalIDCalls)

	repo.externalIDLookup["allegro:cf-7781994292"] = repo.created[0]
	createdAgain, err := upsertAllegroCheckoutForm(context.Background(), nil, repo, mo, tenantID, integrationID)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, createdAgain)
	assert.Len(t, repo.created, 1)
	assert.Equal(t, 1, repo.createIfExternalIDCalls, "duplicate must not insert again")
}
