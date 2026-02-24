package allegro

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOffersList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sale/offers" {
			t.Errorf("path = %q, want /sale/offers", r.URL.Path)
		}
		if lim := r.URL.Query().Get("limit"); lim != "20" {
			t.Errorf("limit = %q, want 20", lim)
		}
		if name := r.URL.Query().Get("name"); name != "Widget" {
			t.Errorf("name = %q, want Widget", name)
		}
		if ps := r.URL.Query().Get("publication.status"); ps != "ACTIVE" {
			t.Errorf("publication.status = %q, want ACTIVE", ps)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"offers": [
				{"id": "off-1", "name": "Widget Pro"},
				{"id": "off-2", "name": "Widget Lite"}
			],
			"count": 2,
			"totalCount": 50
		}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	list, err := c.Offers.List(context.Background(), &ListOffersParams{
		Limit:             20,
		Name:              "Widget",
		PublicationStatus: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("Offers.List error: %v", err)
	}
	if list.Count != 2 {
		t.Errorf("Count = %d, want 2", list.Count)
	}
	if list.TotalCount != 50 {
		t.Errorf("TotalCount = %d, want 50", list.TotalCount)
	}
	if len(list.Offers) != 2 {
		t.Fatalf("len(Offers) = %d, want 2", len(list.Offers))
	}
	if list.Offers[0].Name != "Widget Pro" {
		t.Errorf("Offers[0].Name = %q, want %q", list.Offers[0].Name, "Widget Pro")
	}
}

func TestOffersListWithOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("offset"); q != "10" {
			t.Errorf("offset = %q, want 10", q)
		}
		w.Write([]byte(`{"offers":[],"count":0,"totalCount":0}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Offers.List(context.Background(), &ListOffersParams{Offset: 10})
	if err != nil {
		t.Fatalf("Offers.List error: %v", err)
	}
}

func TestOffersListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Offers.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOffersGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	_, err := c.Offers.Get(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOffersGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sale/product-offers/off-abc" {
			t.Errorf("path = %q, want /sale/product-offers/off-abc", r.URL.Path)
		}
		w.Write([]byte(`{"id":"off-abc","name":"Super Widget"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	defer c.Close()

	offer, err := c.Offers.Get(context.Background(), "off-abc")
	if err != nil {
		t.Fatalf("Offers.Get error: %v", err)
	}
	if offer.ID != "off-abc" {
		t.Errorf("ID = %q, want %q", offer.ID, "off-abc")
	}
	if offer.Name != "Super Widget" {
		t.Errorf("Name = %q, want %q", offer.Name, "Super Widget")
	}
}

func TestBulkUpdateStock_GroupsByQuantity(t *testing.T) {
	var requests []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/sale/offer-quantity-change-commands/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "PUT" {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		requests = append(requests, parsed)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	err := c.Offers.BulkUpdateStock(context.Background(), []StockUpdate{
		{OfferID: "a", Quantity: 5},
		{OfferID: "b", Quantity: 5},
		{OfferID: "c", Quantity: 10},
	})
	if err != nil {
		t.Fatalf("BulkUpdateStock error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 commands (grouped by qty), got %d", len(requests))
	}

	for _, req := range requests {
		mod, ok := req["modification"].(map[string]any)
		if !ok {
			t.Fatal("missing modification object")
		}
		if mod["changeType"] != "FIXED" {
			t.Errorf("changeType = %v, want FIXED", mod["changeType"])
		}
		criteria, ok := req["offerCriteria"].([]any)
		if !ok || len(criteria) == 0 {
			t.Fatal("missing offerCriteria")
		}
		crit := criteria[0].(map[string]any)
		if crit["type"] != "CONTAINS_OFFERS" {
			t.Errorf("criteria type = %v, want CONTAINS_OFFERS", crit["type"])
		}
	}
}

func TestBulkUpdateStock_EmptyNoop(t *testing.T) {
	c := NewClient("id", "secret")
	defer c.Close()

	err := c.Offers.BulkUpdateStock(context.Background(), nil)
	if err != nil {
		t.Fatalf("BulkUpdateStock(nil) error: %v", err)
	}
}

func TestBulkUpdatePrice_GroupsByPrice(t *testing.T) {
	var requests []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/sale/offer-price-change-commands/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		requests = append(requests, parsed)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	err := c.Offers.BulkUpdatePrice(context.Background(), []PriceUpdate{
		{OfferID: "a", Amount: 9.99, Currency: "PLN"},
		{OfferID: "b", Amount: 9.99, Currency: "PLN"},
		{OfferID: "c", Amount: 19.50, Currency: "PLN"},
	})
	if err != nil {
		t.Fatalf("BulkUpdatePrice error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 commands (grouped by price), got %d", len(requests))
	}

	for _, req := range requests {
		mod, ok := req["modification"].(map[string]any)
		if !ok {
			t.Fatal("missing modification object")
		}
		if mod["type"] != "FIXED_PRICE" {
			t.Errorf("modification type = %v, want FIXED_PRICE", mod["type"])
		}
		price, ok := mod["price"].(map[string]any)
		if !ok {
			t.Fatal("missing price in modification")
		}
		if price["currency"] != "PLN" {
			t.Errorf("currency = %v, want PLN", price["currency"])
		}
	}
}

func TestBulkUpdatePrice_EmptyNoop(t *testing.T) {
	c := NewClient("id", "secret")
	defer c.Close()

	err := c.Offers.BulkUpdatePrice(context.Background(), nil)
	if err != nil {
		t.Fatalf("BulkUpdatePrice(nil) error: %v", err)
	}
}

func TestBulkUpdateStock_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"BAD_REQUEST","message":"Invalid offers"}`))
	}))
	defer srv.Close()

	c := NewClient("id", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	defer c.Close()

	err := c.Offers.BulkUpdateStock(context.Background(), []StockUpdate{
		{OfferID: "a", Quantity: 5},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bulk stock update") {
		t.Errorf("error should contain 'bulk stock update', got: %v", err)
	}
}
