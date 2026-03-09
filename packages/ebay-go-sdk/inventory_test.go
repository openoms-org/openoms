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
		_ = json.NewEncoder(w).Encode(resp)
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
		_ = json.NewEncoder(w).Encode(resp)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		_ = json.NewEncoder(w).Encode(resp)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[{"errorId":1001,"domain":"API_INVENTORY","message":"Invalid access token"}]}`))
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

func TestCreateOrReplaceInventoryItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/inventory_item/TEST-SKU-001" {
			t.Errorf("path = %q, want /sell/inventory/v1/inventory_item/TEST-SKU-001", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var item InventoryItem
		if err := json.Unmarshal(body, &item); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if item.Product.Title != "Test Widget" {
			t.Errorf("product.title = %q, want Test Widget", item.Product.Title)
		}
		if item.Condition != "NEW" {
			t.Errorf("condition = %q, want NEW", item.Condition)
		}
		if item.Availability == nil || item.Availability.ShipToLocationAvailability == nil {
			t.Fatal("availability.shipToLocationAvailability is nil")
		}
		if item.Availability.ShipToLocationAvailability.Quantity != 10 {
			t.Errorf("quantity = %d, want 10", item.Availability.ShipToLocationAvailability.Quantity)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Inventory.CreateOrReplaceInventoryItem(context.Background(), "TEST-SKU-001", InventoryItem{
		Product: InventoryProduct{
			Title:       "Test Widget",
			Description: "A test widget",
			ImageURLs:   []string{"https://example.com/img.jpg"},
		},
		Condition: "NEW",
		Availability: &Availability{
			ShipToLocationAvailability: &ShipToLocationAvailability{
				Quantity: 10,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateOrReplaceInventoryItem error: %v", err)
	}
}

func TestGetInventoryItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/inventory_item/TEST-SKU-001" {
			t.Errorf("path = %q, want /sell/inventory/v1/inventory_item/TEST-SKU-001", r.URL.Path)
		}

		resp := InventoryItem{
			SKU:       "TEST-SKU-001",
			Locale:    "pl_PL",
			Condition: "NEW",
			Product: InventoryProduct{
				Title:       "Test Widget",
				Description: "A test widget",
				Brand:       "TestBrand",
				EAN:         []string{"5901234123457"},
				ImageURLs:   []string{"https://example.com/img.jpg"},
			},
			Availability: &Availability{
				ShipToLocationAvailability: &ShipToLocationAvailability{
					Quantity: 10,
				},
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

	result, err := c.Inventory.GetInventoryItem(context.Background(), "TEST-SKU-001")
	if err != nil {
		t.Fatalf("GetInventoryItem error: %v", err)
	}
	if result.SKU != "TEST-SKU-001" {
		t.Errorf("SKU = %q, want TEST-SKU-001", result.SKU)
	}
	if result.Product.Title != "Test Widget" {
		t.Errorf("Product.Title = %q, want Test Widget", result.Product.Title)
	}
	if result.Product.Brand != "TestBrand" {
		t.Errorf("Product.Brand = %q, want TestBrand", result.Product.Brand)
	}
	if result.Condition != "NEW" {
		t.Errorf("Condition = %q, want NEW", result.Condition)
	}
	if result.Availability == nil || result.Availability.ShipToLocationAvailability == nil {
		t.Fatal("Availability.ShipToLocationAvailability is nil")
	}
	if result.Availability.ShipToLocationAvailability.Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", result.Availability.ShipToLocationAvailability.Quantity)
	}
}

func TestListInventoryItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/inventory_item" {
			t.Errorf("path = %q, want /sell/inventory/v1/inventory_item", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "25" {
			t.Errorf("limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "0" {
			t.Errorf("offset = %q, want 0", r.URL.Query().Get("offset"))
		}

		resp := InventoryItemsResponse{
			Total: 2,
			Size:  2,
			Href:  "/sell/inventory/v1/inventory_item?limit=25&offset=0",
			InventoryItems: []InventoryItem{
				{
					SKU:       "SKU-001",
					Condition: "NEW",
					Product:   InventoryProduct{Title: "Widget A"},
					Availability: &Availability{
						ShipToLocationAvailability: &ShipToLocationAvailability{Quantity: 5},
					},
				},
				{
					SKU:       "SKU-002",
					Condition: "USED_EXCELLENT",
					Product:   InventoryProduct{Title: "Widget B"},
					Availability: &Availability{
						ShipToLocationAvailability: &ShipToLocationAvailability{Quantity: 3},
					},
				},
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

	result, err := c.Inventory.ListInventoryItems(context.Background(), 25, 0)
	if err != nil {
		t.Fatalf("ListInventoryItems error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if len(result.InventoryItems) != 2 {
		t.Fatalf("len(InventoryItems) = %d, want 2", len(result.InventoryItems))
	}
	if result.InventoryItems[0].SKU != "SKU-001" {
		t.Errorf("InventoryItems[0].SKU = %q, want SKU-001", result.InventoryItems[0].SKU)
	}
	if result.InventoryItems[1].SKU != "SKU-002" {
		t.Errorf("InventoryItems[1].SKU = %q, want SKU-002", result.InventoryItems[1].SKU)
	}
	if result.InventoryItems[1].Condition != "USED_EXCELLENT" {
		t.Errorf("InventoryItems[1].Condition = %q, want USED_EXCELLENT", result.InventoryItems[1].Condition)
	}
}

func TestDeleteInventoryItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/sell/inventory/v1/inventory_item/TEST-SKU-001" {
			t.Errorf("path = %q, want /sell/inventory/v1/inventory_item/TEST-SKU-001", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	err := c.Inventory.DeleteInventoryItem(context.Background(), "TEST-SKU-001")
	if err != nil {
		t.Fatalf("DeleteInventoryItem error: %v", err)
	}
}
