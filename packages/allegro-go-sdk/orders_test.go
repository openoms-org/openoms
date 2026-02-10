package allegro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOrdersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/checkout-forms" {
			t.Errorf("path = %q, want /order/checkout-forms", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if lim := r.URL.Query().Get("limit"); lim != "10" {
			t.Errorf("limit = %q, want 10", lim)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"checkoutForms": [
				{
					"id": "order-1",
					"buyer": {"id": "buyer-1", "login": "jan", "email": "jan@test.pl"},
					"payment": {"id": "pay-1", "type": "ONLINE", "paidAmount": {"amount": "99.99", "currency": "PLN"}},
					"status": "READY_FOR_PROCESSING",
					"fulfillment": {"status": "NEW"},
					"delivery": {
						"address": {"firstName": "Jan", "lastName": "Kowalski", "street": "Marszalkowska 1", "city": "Warszawa", "zipCode": "00-001", "countryCode": "PL"},
						"method": {"id": "dm-1", "name": "InPost Paczkomaty"}
					},
					"invoice": {"required": false},
					"lineItems": [
						{"id": "li-1", "offer": {"id": "off-1", "name": "Widget", "external": "SKU-001"}, "quantity": 2, "price": {"amount": "49.99", "currency": "PLN"}}
					],
					"updatedAt": "2024-01-15T10:30:00Z"
				}
			],
			"count": 1,
			"totalCount": 1
		}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	list, err := c.Orders.List(context.Background(), &ListOrdersParams{Limit: 10})
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
	if list.Count != 1 {
		t.Errorf("Count = %d, want 1", list.Count)
	}
	if len(list.CheckoutForms) != 1 {
		t.Fatalf("len(CheckoutForms) = %d, want 1", len(list.CheckoutForms))
	}
	order := list.CheckoutForms[0]
	if order.ID != "order-1" {
		t.Errorf("ID = %q, want %q", order.ID, "order-1")
	}
	if order.Buyer.Email != "jan@test.pl" {
		t.Errorf("Buyer.Email = %q, want %q", order.Buyer.Email, "jan@test.pl")
	}
	if order.Payment.PaidAmount.Amount != "99.99" {
		t.Errorf("PaidAmount.Amount = %q, want %q", order.Payment.PaidAmount.Amount, "99.99")
	}
	if len(order.LineItems) != 1 {
		t.Fatalf("len(LineItems) = %d, want 1", len(order.LineItems))
	}
	if order.LineItems[0].Quantity != 2 {
		t.Errorf("LineItems[0].Quantity = %d, want 2", order.LineItems[0].Quantity)
	}
}

func TestOrdersListNilParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		w.Write([]byte(`{"checkoutForms":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	list, err := c.Orders.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
	if list.Count != 0 {
		t.Errorf("Count = %d, want 0", list.Count)
	}
}

func TestOrdersListAllParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("offset") != "5" {
			t.Errorf("offset = %q, want 5", q.Get("offset"))
		}
		if q.Get("status") != "READY_FOR_PROCESSING" {
			t.Errorf("status = %q, want READY_FOR_PROCESSING", q.Get("status"))
		}
		if q.Get("fulfillment.status") != "NEW" {
			t.Errorf("fulfillment.status = %q, want NEW", q.Get("fulfillment.status"))
		}
		if q.Get("updatedAt.gte") == "" {
			t.Error("updatedAt.gte should be set")
		}
		if q.Get("updatedAt.lte") == "" {
			t.Error("updatedAt.lte should be set")
		}
		w.Write([]byte(`{"checkoutForms":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	gte := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lte := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err := c.Orders.List(context.Background(), &ListOrdersParams{
		Limit:             10,
		Offset:            5,
		Status:            "READY_FOR_PROCESSING",
		FulfillmentStatus: "NEW",
		UpdatedAtGte:      &gte,
		UpdatedAtLte:      &lte,
	})
	if err != nil {
		t.Fatalf("Orders.List error: %v", err)
	}
}

func TestOrdersListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"server error"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Orders.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOrdersGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Orders.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOrdersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/order/checkout-forms/order-abc" {
			t.Errorf("path = %q, want /order/checkout-forms/order-abc", r.URL.Path)
		}
		w.Write([]byte(`{"id":"order-abc","status":"READY_FOR_PROCESSING","updatedAt":"2024-01-15T10:30:00Z"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	order, err := c.Orders.Get(context.Background(), "order-abc")
	if err != nil {
		t.Fatalf("Orders.Get error: %v", err)
	}
	if order.ID != "order-abc" {
		t.Errorf("ID = %q, want %q", order.ID, "order-abc")
	}

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !order.UpdatedAt.Equal(expectedTime) {
		t.Errorf("UpdatedAt = %v, want %v", order.UpdatedAt, expectedTime)
	}
}
