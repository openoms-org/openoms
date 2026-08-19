package allegro

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

func newTestProvider(t *testing.T, srvURL string) *Provider {
	t.Helper()
	client := allegrosdk.NewClient("id", "secret",
		allegrosdk.WithBaseURL(srvURL),
		allegrosdk.WithHTTPClient(http.DefaultClient),
		allegrosdk.WithTokens("tok", "", time.Now().Add(time.Hour)),
	)
	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "allegro-test"),
	}
}

func sampleCheckoutForm() allegrosdk.Order {
	phone := &allegrosdk.BuyerPhone{Number: "+48123456789"}
	return allegrosdk.Order{
		ID: "cf-7781994292",
		Buyer: allegrosdk.Buyer{
			ID:    "buyer-1",
			Login: "jan_k",
			Email: "jan@test.pl",
			Phone: phone,
		},
		Payment: allegrosdk.Payment{
			ID:   "pay-1",
			Type: "ONLINE",
			PaidAmount: allegrosdk.Amount{
				Amount:   "9.99",
				Currency: "PLN",
			},
		},
		Status: "READY_FOR_PROCESSING",
		Delivery: allegrosdk.Delivery{
			Address: allegrosdk.Address{
				FirstName:   "Jan",
				LastName:    "Kowalski",
				Street:      "Marszalkowska 1",
				City:        "Warszawa",
				ZipCode:     "00-001",
				CountryCode: "PL",
				Phone:       "+48999888777",
			},
			Method: allegrosdk.DeliveryMethod{
				ID:   "dm-inpost",
				Name: "InPost Paczkomaty 24/7",
			},
			PickupPoint: &allegrosdk.PickupPoint{
				ID:   "WAW01A",
				Name: "Paczkomat WAW01A",
			},
		},
		LineItems: []allegrosdk.LineItem{
			{
				ID: "li-1",
				Offer: allegrosdk.LineItemOffer{
					ID:       "7781994292",
					Name:     "Sandbox widget",
					External: "SKU-SANDBOX",
				},
				Quantity: 1,
				Price: allegrosdk.Amount{
					Amount:   "9.99",
					Currency: "PLN",
				},
			},
		},
		UpdatedAt: time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC),
	}
}

func TestMapAllegroOrder_MapsBuyerItemsPaymentDeliveryPickup(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	mo := p.mapAllegroOrder(ptrOrder(sampleCheckoutForm()))

	assert.Equal(t, "cf-7781994292", mo.ExternalID)
	assert.Equal(t, "READY_FOR_PROCESSING", mo.ExternalStatus)
	assert.Equal(t, "Jan Kowalski", mo.CustomerName)
	assert.Equal(t, "jan@test.pl", mo.CustomerEmail)
	assert.Equal(t, "+48123456789", mo.CustomerPhone)
	assert.Equal(t, "paid", mo.PaymentStatus)
	assert.Equal(t, "ONLINE", mo.PaymentMethod)
	assert.Equal(t, 9.99, mo.TotalAmount)
	assert.Equal(t, "PLN", mo.Currency)

	assert.Equal(t, "Jan Kowalski", mo.ShippingAddress.Name)
	assert.Equal(t, "Marszalkowska 1", mo.ShippingAddress.Street)
	assert.Equal(t, "Warszawa", mo.ShippingAddress.City)
	assert.Equal(t, "00-001", mo.ShippingAddress.PostalCode)
	assert.Equal(t, "PL", mo.ShippingAddress.Country)
	assert.Equal(t, "jan@test.pl", mo.ShippingAddress.Email)

	require.Len(t, mo.Items, 1)
	assert.Equal(t, "7781994292", mo.Items[0].ExternalID)
	assert.Equal(t, "Sandbox widget", mo.Items[0].Name)
	assert.Equal(t, "SKU-SANDBOX", mo.Items[0].SKU)
	assert.Equal(t, 1, mo.Items[0].Quantity)
	assert.Equal(t, 9.99, mo.Items[0].UnitPrice)
	assert.Equal(t, 9.99, mo.Items[0].TotalPrice)

	require.NotNil(t, mo.RawData)
	assert.Equal(t, "dm-inpost", mo.RawData["delivery_method_id"])
	assert.Equal(t, "InPost Paczkomaty 24/7", mo.RawData["delivery_method_name"])
	assert.Equal(t, "WAW01A", mo.RawData["pickup_point_id"])
	assert.Equal(t, "Paczkomat WAW01A", mo.RawData["pickup_point_name"])
}

func TestMapAllegroOrder_BuyerLoginWhenDeliveryNameEmpty(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	form := sampleCheckoutForm()
	form.Delivery.Address.FirstName = ""
	form.Delivery.Address.LastName = ""
	mo := p.mapAllegroOrder(&form)

	assert.Equal(t, "jan_k", mo.CustomerName)
}

func TestMapAllegroOrder_UnpaidUsesLineItemTotal(t *testing.T) {
	p := &Provider{logger: slog.Default()}
	form := sampleCheckoutForm()
	form.Payment.PaidAmount.Amount = "0.00"
	form.LineItems[0].Quantity = 2
	form.LineItems[0].Price.Amount = "9.99"
	mo := p.mapAllegroOrder(&form)

	assert.Equal(t, "pending", mo.PaymentStatus)
	assert.Equal(t, 19.98, mo.TotalAmount)
}

func TestPollOrders_ListsCheckoutForms(t *testing.T) {
	var hitEvents bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/order/events" {
			hitEvents = true
			http.NotFound(w, r)
			return
		}
		if r.URL.Path != "/order/checkout-forms" {
			t.Errorf("path = %q, want /order/checkout-forms", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"checkoutForms": [{
				"id": "cf-7781994292",
				"buyer": {"id": "buyer-1", "login": "jan_k", "email": "jan@test.pl"},
				"payment": {"id": "pay-1", "type": "ONLINE", "paidAmount": {"amount": "9.99", "currency": "PLN"}},
				"status": "READY_FOR_PROCESSING",
				"fulfillment": {"status": "NEW"},
				"delivery": {
					"address": {"firstName": "Jan", "lastName": "Kowalski", "street": "Marszalkowska 1", "city": "Warszawa", "zipCode": "00-001", "countryCode": "PL"},
					"method": {"id": "dm-inpost", "name": "InPost Paczkomaty 24/7"},
					"pickupPoint": {"id": "WAW01A", "name": "Paczkomat WAW01A"}
				},
				"invoice": {"required": false},
				"lineItems": [
					{"id": "li-1", "offer": {"id": "7781994292", "name": "Sandbox widget", "external": "SKU-SANDBOX"}, "quantity": 1, "price": {"amount": "9.99", "currency": "PLN"}}
				],
				"updatedAt": "2026-08-19T10:00:00Z"
			}],
			"count": 1,
			"totalCount": 1
		}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	orders, cursor, err := p.PollOrders(context.Background(), "")

	require.NoError(t, err)
	assert.False(t, hitEvents, "PollOrders must list checkout-forms, not order events")
	require.Len(t, orders, 1)
	assert.Equal(t, "cf-7781994292", orders[0].ExternalID)
	assert.Equal(t, "Jan Kowalski", orders[0].CustomerName)
	assert.Equal(t, "2026-08-19T10:00:00Z", cursor)
}

func TestPollOrders_UsesUpdatedAtCursor(t *testing.T) {
	var gotGte string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotGte = r.URL.Query().Get("updatedAt.gte")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkoutForms":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	orders, cursor, err := p.PollOrders(context.Background(), "2026-08-01T00:00:00Z")

	require.NoError(t, err)
	assert.Empty(t, orders)
	assert.Equal(t, "2026-08-01T00:00:00Z", cursor)
	assert.Equal(t, "2026-08-01T00:00:00Z", gotGte)
}

func TestPollOrders_IgnoresLegacyEventIDCursor(t *testing.T) {
	var gotGte string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotGte = r.URL.Query().Get("updatedAt.gte")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkoutForms":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	_, _, err := p.PollOrders(context.Background(), "legacy-event-id-abc")

	require.NoError(t, err)
	assert.Empty(t, gotGte, "non-RFC3339 cursors must be ignored so leftover event IDs do a full list")
}

func ptrOrder(o allegrosdk.Order) *allegrosdk.Order {
	return &o
}
