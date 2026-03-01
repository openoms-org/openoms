package gls

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// GLS ShipIT REST API — Specification Tests
//
// These tests define the CORRECT behavior according to the official GLS ShipIT
// REST API documentation. They verify:
//   - Authentication: Basic Auth (username:password), NOT Bearer token
//   - Content-Type: application/glsVersion1+json
//   - Create: POST /shipments (not /parcels)
//   - Tracking: POST /shipments/parceldetails (not GET /tracking/{id})
//   - Cancel: POST /shipments/cancel/{trackID} (not DELETE /parcels/{id})
//   - Status mapping: includes CANCELLED, CANCELLATION_PENDING, SCANNED
// =============================================================================

// --- Authentication ---

func TestSpec_Auth_UsesBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth, got none or different auth scheme")
		}
		if user != "testuser" {
			t.Errorf("Basic Auth username = %q, want %q", user, "testuser")
		}
		if pass != "testpass" {
			t.Errorf("Basic Auth password = %q, want %q", pass, "testpass")
		}

		// Must NOT have Bearer token
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			t.Error("should use Basic Auth, not Bearer token")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	// Plan: NewClient(username, password string, opts ...Option)
	// Current code takes apiKey — this documents the expected signature change
	c := NewClient("testuser",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	_ = c.do(context.Background(), "GET", "/test", nil, nil)
}

func TestSpec_ContentType_UsesGLSVersionHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/glsVersion1+json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/glsVersion1+json")
		}

		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "application/glsVersion1+json") {
			t.Errorf("Accept should contain 'application/glsVersion1+json', got %q", accept)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ParcelData":[{"TrackID":"T1"}]}}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Consignee: Party{Name: "Test"},
		Parcels:   []Parcel{{Weight: 1.0}},
	})
}

// --- Create Shipment ---

func TestSpec_CreateShipment_PostsToShipmentsEndpoint(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"T1","PrintData":"AAAA"}]}}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Consignee: Party{
			Name:        "Jan Kowalski",
			Street:      "Marszalkowska 1",
			City:        "Warszawa",
			ZipCode:     "00-001",
			CountryCode: "PL",
		},
		Parcels: []Parcel{{Weight: 2.0}},
	})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/shipments" {
		t.Errorf("path = %q, want /shipments", gotPath)
	}
}

func TestSpec_CreateShipment_ResponseHasCreatedShipmentStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// GLS ShipIT returns CreatedShipment with ParcelData array
		w.Write([]byte(`{
			"CreatedShipment": {
				"ShipmentReference": "ORDER-001",
				"ParcelData": [
					{"TrackID": "GLS-TRK-001", "PrintData": "JVBERi0xLjQ="}
				]
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Consignee: Party{Name: "Test"},
		Parcels:   []Parcel{{Weight: 1.0}},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Plan: response should be CreateParcelResponse with CreatedShipment.ParcelData
	// Current code expects {parcel_ids: [], track_ids: []} — this should fail
	if len(resp.TrackIDs) == 0 && len(resp.ParcelIDs) == 0 {
		t.Error("response should contain track IDs extracted from CreatedShipment.ParcelData")
	}
}

// --- Tracking ---

func TestSpec_GetTracking_UsesPostMethod(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method

		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"events":[{"status":"INTRANSIT","location":"Lodz"}]}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "TRK-001")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}

	// Plan: POST /shipments/parceldetails with TrackIDs in body
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (GLS ShipIT uses POST for tracking)", gotMethod)
	}
	if gotPath != "/shipments/parceldetails" {
		t.Errorf("path = %q, want /shipments/parceldetails", gotPath)
	}
}

// --- Cancel ---

func TestSpec_Cancel_UsesPostToShipmentsCancel(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"CANCELLED"}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Shipments.Cancel(context.Background(), "TRK-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}

	// Plan: POST /shipments/cancel/{trackID} (not DELETE /parcels/{id})
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (GLS ShipIT uses POST for cancel)", gotMethod)
	}
	if gotPath != "/shipments/cancel/TRK-001" {
		t.Errorf("path = %q, want /shipments/cancel/TRK-001", gotPath)
	}
}

// --- Status Mapping ---

func TestSpec_StatusMapping_IncludesAllGLSStatuses(t *testing.T) {
	// Plan: GLS ShipIT returns these statuses — all must be mapped
	tests := []struct {
		glsStatus string
		omsStatus string
		wantOK    bool
	}{
		{"PREADVICE", "pending", true},
		{"INTRANSIT", "in_transit", true},
		{"INWAREHOUSE", "in_transit", true},
		{"INDELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		// These are MISSING in current code — tests should fail
		{"CANCELLED", "cancelled", true},
		{"CANCELLATION_PENDING", "cancelled", true},
		{"SCANNED", "in_transit", true},
		// Unknown status should return false
		{"NONEXISTENT_STATUS", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.glsStatus, func(t *testing.T) {
			oms, ok := MapStatus(tc.glsStatus)
			if ok != tc.wantOK {
				t.Errorf("MapStatus(%q) ok = %v, want %v", tc.glsStatus, ok, tc.wantOK)
			}
			if ok && oms != tc.omsStatus {
				t.Errorf("MapStatus(%q) = %q, want %q", tc.glsStatus, oms, tc.omsStatus)
			}
		})
	}
}

// --- Edge Cases ---

func TestSpec_CreateShipment_EmptyParcels_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"At least one parcel is required"}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Parcels: []Parcel{},
	})
	if err == nil {
		t.Error("expected error for empty parcels, got nil")
	}
}

func TestSpec_CreateShipment_ServerUnavailable_ReturnsError(t *testing.T) {
	c := NewClient("key",
		WithBaseURL("http://localhost:1"), // unreachable
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Parcels: []Parcel{{Weight: 1.0}},
	})
	if err == nil {
		t.Error("expected error for unreachable server, got nil")
	}
}
