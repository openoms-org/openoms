package inpost

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShipmentCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/organizations/org42/shipments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req CreateShipmentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.Service != ServiceLockerStandard {
			t.Fatalf("expected service %s, got %s", ServiceLockerStandard, req.Service)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 100,
			"status": "created",
			"tracking_number": "TRK123456",
			"service": "inpost_locker_standard",
			"receiver": {"name": "Jan Kowalski", "phone": "500100200", "email": "jan@example.com"},
			"parcels": [{"id": 1, "template": "small", "dimensions": {"height": 80, "width": 380, "length": 640}, "weight": {"amount": 2.5, "unit": "kg"}}],
			"custom_attributes": {"target_point": "KRA010"},
			"created_at": "2025-01-01T10:00:00Z",
			"updated_at": "2025-01-01T10:00:00Z"
		}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org42", WithBaseURL(srv.URL))
	shipment, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		Receiver: Receiver{
			Name:  "Jan Kowalski",
			Phone: "500100200",
			Email: "jan@example.com",
		},
		Parcels: []Parcel{
			{Template: ParcelSmall, Weight: Weight{Amount: 2.5, Unit: "kg"}},
		},
		Service:          ServiceLockerStandard,
		CustomAttributes: &CustomAttributes{TargetPoint: "KRA010"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shipment.ID != 100 {
		t.Fatalf("expected ID 100, got %d", shipment.ID)
	}
	if shipment.TrackingNumber != "TRK123456" {
		t.Fatalf("expected tracking TRK123456, got %s", shipment.TrackingNumber)
	}
	if shipment.Status != "created" {
		t.Fatalf("expected status created, got %s", shipment.Status)
	}
}

func TestShipmentGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/shipments/999" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 999,
			"status": "delivered",
			"tracking_number": "TRK999",
			"service": "inpost_courier_standard",
			"receiver": {"name": "Anna Nowak", "phone": "600200300", "email": "anna@example.com"},
			"parcels": [],
			"created_at": "2025-01-01T10:00:00Z",
			"updated_at": "2025-01-02T15:00:00Z"
		}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org1", WithBaseURL(srv.URL))
	shipment, err := c.Shipments.Get(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shipment.ID != 999 {
		t.Fatalf("expected ID 999, got %d", shipment.ID)
	}
	if shipment.Status != "delivered" {
		t.Fatalf("expected status delivered, got %s", shipment.Status)
	}
}

func TestShipmentCreateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message":"validation error","details":{"receiver.phone":"required"}}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org42", WithBaseURL(srv.URL))
	_, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		Service: ServiceLockerStandard,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
}

func TestShipmentGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org1", WithBaseURL(srv.URL))
	_, err := c.Shipments.Get(context.Background(), 12345)
	if err == nil {
		t.Fatal("expected error for 404")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d", apiErr.StatusCode)
	}
}
