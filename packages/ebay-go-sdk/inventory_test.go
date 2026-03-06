package ebay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInventoryUpdateStock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/bulk_update_price_quantity" {
			t.Errorf("path = %q, want /sell/inventory/v1/bulk_update_price_quantity", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		requests, ok := payload["requests"].([]any)
		if !ok || len(requests) != 1 {
			t.Fatalf("expected 1 request, got %v", payload["requests"])
		}
		req := requests[0].(map[string]any)
		if req["offerId"] != "OFFER-123" {
			t.Errorf("offerId = %v, want OFFER-123", req["offerId"])
		}
		if int(req["availableQuantity"].(float64)) != 15 {
			t.Errorf("availableQuantity = %v, want 15", req["availableQuantity"])
		}

		resp := bulkUpdateResponse{
			Responses: []bulkUpdateItemResponse{
				{OfferID: "OFFER-123", StatusCode: 200},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Inventory.UpdateStock(context.Background(), "OFFER-123", 15)
	if err != nil {
		t.Fatalf("UpdateStock error: %v", err)
	}
}

func TestInventoryUpdatePrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sell/inventory/v1/bulk_update_price_quantity" {
			t.Errorf("path = %q, want /sell/inventory/v1/bulk_update_price_quantity", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}

		requests := payload["requests"].([]any)
		req := requests[0].(map[string]any)
		if req["offerId"] != "OFFER-456" {
			t.Errorf("offerId = %v, want OFFER-456", req["offerId"])
		}
		pricing := req["pricingSummary"].(map[string]any)
		price := pricing["price"].(map[string]any)
		if price["value"] != "49.99" {
			t.Errorf("price.value = %v, want 49.99", price["value"])
		}
		if price["currency"] != "PLN" {
			t.Errorf("price.currency = %v, want PLN", price["currency"])
		}

		resp := bulkUpdateResponse{
			Responses: []bulkUpdateItemResponse{
				{OfferID: "OFFER-456", StatusCode: 200},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Inventory.UpdatePrice(context.Background(), "OFFER-456", 49.99, "PLN")
	if err != nil {
		t.Fatalf("UpdatePrice error: %v", err)
	}
}

func TestInventoryUpdateStock_BulkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := bulkUpdateResponse{
			Responses: []bulkUpdateItemResponse{
				{
					OfferID:    "OFFER-789",
					StatusCode: 400,
					Errors: []EbErr{
						{ErrorID: 25710, Message: "Invalid offer ID"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Inventory.UpdateStock(context.Background(), "OFFER-789", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "ebay: bulk update for OFFER-789: Invalid offer ID" {
		t.Errorf("error = %q, want 'ebay: bulk update for OFFER-789: Invalid offer ID'", got)
	}
}

func TestInventoryUpdateStock_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"errorId":1001,"domain":"API_INVENTORY","message":"Invalid access token"}]}`))
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Inventory.UpdateStock(context.Background(), "OFFER-001", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
