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
					External: &allegrosdk.OfferExternal{ID: "SKU-SANDBOX"},
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
					{"id": "li-1", "offer": {"id": "7781994292", "name": "Sandbox widget", "external": {"id": "SKU-SANDBOX"}}, "quantity": 1, "price": {"amount": "9.99", "currency": "PLN"}}
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
	require.NotEmpty(t, gotGte, "non-RFC3339 leftover event IDs must not be forwarded; send a lookback filter instead")
	assert.NotEqual(t, "legacy-event-id-abc", gotGte)
	gte, err := time.Parse(time.RFC3339, gotGte)
	require.NoError(t, err)
	assert.True(t, gte.Before(time.Now().UTC()), "lookback must be in the past")
}

func TestPollOrders_EmptyCursorSendsDateFilter(t *testing.T) {
	var gotURL *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checkoutForms":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	_, _, err := p.PollOrders(context.Background(), "")
	require.NoError(t, err)
	require.NotNil(t, gotURL)

	q := gotURL.URL.Query()
	assert.Equal(t, "100", q.Get("limit"))
	gte := q.Get("updatedAt.gte")
	require.NotEmpty(t, gte, "manual SyncOrders calls PollOrders with an empty cursor; Allegro 400s a limit-only list")
	parsed, err := time.Parse(time.RFC3339, gte)
	require.NoError(t, err, "updatedAt.gte must be RFC3339, got %q", gte)
	assert.True(t, parsed.Before(time.Now().UTC()), "lookback must be in the past")
	assert.Empty(t, q.Get("status"), "do not filter to READY_FOR_PROCESSING; unpaid forms stay in the list")
}

func TestPollOrders_AcceptsOfferExternalObjectAndNull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/checkout-forms" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"checkoutForms": [{
				"id": "19640fd0-9c54-11f1-bd08-9328d2ed1733",
				"buyer": {"id": "buyer-1", "login": "anna", "email": "anna@test.pl"},
				"payment": {"id": "pay-1", "type": "ONLINE", "paidAmount": {"amount": "22.48", "currency": "PLN"}},
				"status": "READY_FOR_PROCESSING",
				"fulfillment": {"status": "NEW"},
				"delivery": {
					"address": {"firstName": "Anna", "lastName": "Testowa", "street": "Testowa 1", "city": "Poznan", "zipCode": "60-001", "countryCode": "PL"},
					"method": {"id": "dm-1", "name": "InPost"}
				},
				"invoice": {"required": false},
				"lineItems": [
					{"id": "li-1", "offer": {"id": "7781994292", "name": "BTP SKU", "external": {"id": "SKU-1"}}, "quantity": 1, "price": {"amount": "22.48", "currency": "PLN"}},
					{"id": "li-2", "offer": {"id": "66681830", "name": "Leftover", "external": null}, "quantity": 1, "price": {"amount": "1.00", "currency": "PLN"}}
				],
				"updatedAt": "2026-08-20T10:00:00Z"
			}],
			"count": 1,
			"totalCount": 1
		}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	orders, cursor, err := p.PollOrders(context.Background(), "")

	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, "19640fd0-9c54-11f1-bd08-9328d2ed1733", orders[0].ExternalID)
	require.Len(t, orders[0].Items, 2)
	assert.Equal(t, "SKU-1", orders[0].Items[0].SKU)
	assert.Empty(t, orders[0].Items[1].SKU)
	assert.Equal(t, "2026-08-20T10:00:00Z", cursor)
}

func TestListDeliveryServices_OfficialServicesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shipment-management/delivery-services" {
			t.Errorf("path = %q, want /shipment-management/delivery-services", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"services": [
				{
					"id": {
						"deliveryMethodId": "c3066682-97a3-42fe-9eb5-3beeccab840c",
						"credentialsId": null
					},
					"name": "Allegro miniKurier24 InPost",
					"carrierId": "INPOST",
					"owner": "ALLEGRO"
				}
			]
		}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	services, err := p.ListDeliveryServices(context.Background())

	require.NoError(t, err)
	require.Len(t, services, 1)
	assert.Equal(t, "c3066682-97a3-42fe-9eb5-3beeccab840c", services[0].ID)
	assert.Equal(t, "Allegro miniKurier24 InPost", services[0].Name)
	assert.Equal(t, "INPOST", services[0].CarrierID)
	assert.Equal(t, "ALLEGRO", services[0].Owner)
}

func TestGetDeliveryProposals_OfficialSuggestedInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/shipment-management/delivery-proposals/19829450-9c54-11f1-bd08-9328d2ed1733" {
			t.Errorf("path = %q, want delivery-proposals/{orderId}", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"orderId": "19829450-9c54-11f1-bd08-9328d2ed1733",
			"suggestedInput": {
				"deliveryMethodId": "c3066682-97a3-42fe-9eb5-3beeccab840c",
				"sender": {"street":"Główna 30","postalCode":"10-200","city":"Warszawa","countryCode":"PL"},
				"receiver": {"street":"Marszałkowska 1","postalCode":"00-001","city":"Warszawa","countryCode":"PL"}
			}
		}`))
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	proposals, err := p.GetDeliveryProposals(context.Background(), "19829450-9c54-11f1-bd08-9328d2ed1733")

	require.NoError(t, err)
	require.NotNil(t, proposals)
	assert.Equal(t, "c3066682-97a3-42fe-9eb5-3beeccab840c", proposals.SuggestedInput.DeliveryMethodID)
	assert.Equal(t, "10-200", proposals.SuggestedInput.Sender.PostalCode)
}

func ptrOrder(o allegrosdk.Order) *allegrosdk.Order {
	return &o
}
