package ebay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBulkUpdatePriceQuantity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/bulk_update_price_quantity" {
			t.Errorf("path = %q, want /sell/inventory/v1/bulk_update_price_quantity", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string][]map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		requests := reqBody["requests"]
		if len(requests) != 3 {
			t.Fatalf("len(requests) = %d, want 3", len(requests))
		}

		// Verify first request has stock only
		if requests[0]["offerId"] != "OFFER-1" {
			t.Errorf("requests[0].offerId = %v, want OFFER-1", requests[0]["offerId"])
		}
		if _, hasPricing := requests[0]["pricingSummary"]; hasPricing {
			t.Error("requests[0] should not have pricingSummary (stock-only update)")
		}

		// Verify second request has price only
		if requests[1]["offerId"] != "OFFER-2" {
			t.Errorf("requests[1].offerId = %v, want OFFER-2", requests[1]["offerId"])
		}
		if _, hasQty := requests[1]["availableQuantity"]; hasQty {
			t.Error("requests[1] should not have availableQuantity (price-only update)")
		}

		// Verify third request has both
		if requests[2]["offerId"] != "OFFER-3" {
			t.Errorf("requests[2].offerId = %v, want OFFER-3", requests[2]["offerId"])
		}

		resp := BulkUpdateResponse{
			Responses: []BulkUpdateItemResponse{
				{OfferID: "OFFER-1", StatusCode: 200},
				{OfferID: "OFFER-2", StatusCode: 200},
				{OfferID: "OFFER-3", StatusCode: 200},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	qty := 10
	results, err := c.Inventory.BulkUpdatePriceQuantity(context.Background(), []BulkPriceQuantityRequest{
		{OfferID: "OFFER-1", AvailableQuantity: &qty},
		{OfferID: "OFFER-2", Price: &Amount{Value: "29.99", Currency: "PLN"}},
		{OfferID: "OFFER-3", AvailableQuantity: &qty, Price: &Amount{Value: "19.99", Currency: "EUR"}},
	})
	if err != nil {
		t.Fatalf("BulkUpdatePriceQuantity error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.StatusCode != 200 {
			t.Errorf("results[%d].StatusCode = %d, want 200", i, r.StatusCode)
		}
	}
}

func TestBulkUpdatePriceQuantity_PartialError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := BulkUpdateResponse{
			Responses: []BulkUpdateItemResponse{
				{OfferID: "OFFER-1", StatusCode: 200},
				{OfferID: "OFFER-2", StatusCode: 400, Errors: []EbErr{{Message: "Invalid offer"}}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	qty := 5
	results, err := c.Inventory.BulkUpdatePriceQuantity(context.Background(), []BulkPriceQuantityRequest{
		{OfferID: "OFFER-1", AvailableQuantity: &qty},
		{OfferID: "OFFER-2", AvailableQuantity: &qty},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if results == nil {
		t.Fatal("expected non-nil results even on partial error")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "OFFER-2") {
		t.Errorf("error should mention OFFER-2, got %q", errMsg)
	}
	if !strings.Contains(errMsg, "Invalid offer") {
		t.Errorf("error should mention 'Invalid offer', got %q", errMsg)
	}
}
