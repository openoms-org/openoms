package ebay

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateShippingFulfillment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/fulfillment/v1/order/04-12345-67890/shipping_fulfillment" {
			t.Errorf("path = %q, want /sell/fulfillment/v1/order/04-12345-67890/shipping_fulfillment", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var req ShippingFulfillmentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if req.ShippingCarrierCode != "INPOST" {
			t.Errorf("shippingCarrierCode = %q, want INPOST", req.ShippingCarrierCode)
		}
		if req.TrackingNumber != "TRACK-999" {
			t.Errorf("trackingNumber = %q, want TRACK-999", req.TrackingNumber)
		}
		if len(req.LineItems) != 1 {
			t.Fatalf("len(lineItems) = %d, want 1", len(req.LineItems))
		}
		if req.LineItems[0].LineItemID != "1001" {
			t.Errorf("lineItemId = %q, want 1001", req.LineItems[0].LineItemID)
		}
		if req.LineItems[0].Quantity != 2 {
			t.Errorf("quantity = %d, want 2", req.LineItems[0].Quantity)
		}

		resp := ShippingFulfillment{
			FulfillmentID:          "FULFILLMENT-001",
			ShipmentTrackingNumber: "TRACK-999",
			ShippingCarrierCode:    "INPOST",
			ShippedDate:            "2024-06-15T14:00:00.000Z",
			LineItems: []FulfillmentLineItem{
				{LineItemID: "1001", Quantity: 2},
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

	result, err := c.Fulfillment.CreateShippingFulfillment(context.Background(), "04-12345-67890", ShippingFulfillmentRequest{
		LineItems: []FulfillmentLineItem{
			{LineItemID: "1001", Quantity: 2},
		},
		ShippingCarrierCode: "INPOST",
		TrackingNumber:      "TRACK-999",
	})
	if err != nil {
		t.Fatalf("CreateShippingFulfillment error: %v", err)
	}
	if result.FulfillmentID != "FULFILLMENT-001" {
		t.Errorf("FulfillmentID = %q, want FULFILLMENT-001", result.FulfillmentID)
	}
	if result.ShipmentTrackingNumber != "TRACK-999" {
		t.Errorf("ShipmentTrackingNumber = %q, want TRACK-999", result.ShipmentTrackingNumber)
	}
	if result.ShippingCarrierCode != "INPOST" {
		t.Errorf("ShippingCarrierCode = %q, want INPOST", result.ShippingCarrierCode)
	}
	if len(result.LineItems) != 1 {
		t.Fatalf("len(LineItems) = %d, want 1", len(result.LineItems))
	}
}

func TestGetShippingFulfillments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/sell/fulfillment/v1/order/04-12345-67890/shipping_fulfillment" {
			t.Errorf("path = %q, want /sell/fulfillment/v1/order/04-12345-67890/shipping_fulfillment", r.URL.Path)
		}

		resp := ShippingFulfillmentPagedCollection{
			Total: 2,
			Fulfillments: []ShippingFulfillment{
				{
					FulfillmentID:          "FULFILLMENT-001",
					ShipmentTrackingNumber: "TRACK-111",
					ShippingCarrierCode:    "INPOST",
					ShippedDate:            "2024-06-15T14:00:00.000Z",
					LineItems: []FulfillmentLineItem{
						{LineItemID: "1001", Quantity: 1},
					},
				},
				{
					FulfillmentID:          "FULFILLMENT-002",
					ShipmentTrackingNumber: "TRACK-222",
					ShippingCarrierCode:    "DHL",
					ShippedDate:            "2024-06-16T10:00:00.000Z",
					LineItems: []FulfillmentLineItem{
						{LineItemID: "1002", Quantity: 3},
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

	result, err := c.Fulfillment.GetShippingFulfillments(context.Background(), "04-12345-67890")
	if err != nil {
		t.Fatalf("GetShippingFulfillments error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("Total = %d, want 2", result.Total)
	}
	if len(result.Fulfillments) != 2 {
		t.Fatalf("len(Fulfillments) = %d, want 2", len(result.Fulfillments))
	}
	if result.Fulfillments[0].FulfillmentID != "FULFILLMENT-001" {
		t.Errorf("Fulfillments[0].FulfillmentID = %q, want FULFILLMENT-001", result.Fulfillments[0].FulfillmentID)
	}
	if result.Fulfillments[1].ShippingCarrierCode != "DHL" {
		t.Errorf("Fulfillments[1].ShippingCarrierCode = %q, want DHL", result.Fulfillments[1].ShippingCarrierCode)
	}
}

func TestCreateShippingFulfillment_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":[{"errorId":35000,"domain":"API_FULFILLMENT","category":"REQUEST","message":"Invalid shipping carrier code"}]}`))
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	_, err := c.Fulfillment.CreateShippingFulfillment(context.Background(), "04-99999-00000", ShippingFulfillmentRequest{
		LineItems: []FulfillmentLineItem{
			{LineItemID: "9999", Quantity: 1},
		},
		ShippingCarrierCode: "INVALID",
		TrackingNumber:      "TRACK-BAD",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error should wrap the APIError from client.do
	// Verify the error message contains both the wrapper and the API error
	got := err.Error()
	want := "ebay: create shipping fulfillment for order 04-99999-00000: ebay: HTTP 400: Invalid shipping carrier code"
	if got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}
