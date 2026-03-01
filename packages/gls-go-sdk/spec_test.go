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
//   - Tracking: POST /shipments/parceldetails with TrackID (singular string)
//   - Cancel: POST /shipments/cancel/{trackID} (not DELETE /parcels/{id})
//   - Response: PrintData at CreatedShipment level (not inside ParcelData)
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

	c := NewClient("testuser", "testpass",
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

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product:      "PARCEL",
		Consignee:    Consignee{Address: ConsigneeAddress{Name1: "Test"}},
		ShippingUnit: []ShipmentUnit{{Weight: 1.0}},
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
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"T1"}]}}`))
	}))
	defer srv.Close()

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product: "PARCEL",
		Consignee: Consignee{
			Address: ConsigneeAddress{
				Name1:       "Jan Kowalski",
				Street:      "Marszalkowska 1",
				City:        "Warszawa",
				ZIPCode:     "00-001",
				CountryCode: "PL",
			},
		},
		ShippingUnit: []ShipmentUnit{{Weight: 2.0}},
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

func TestSpec_CreateShipment_RequestUsesCorrectFieldNames(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ParcelData":[{"TrackID":"T1"}]}}`))
	}))
	defer srv.Close()

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product: "PARCEL",
		Consignee: Consignee{
			Address: ConsigneeAddress{
				Name1:       "Jan Kowalski",
				Street:      "Marszalkowska 1",
				City:        "Warszawa",
				ZIPCode:     "00-001",
				CountryCode: "PL",
				EMail:       "jan@test.pl",
			},
		},
		ShippingUnit: []ShipmentUnit{{Weight: 2.0}},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Verify Consignee uses nested Address structure with correct field names
	consignee, _ := gotBody["Consignee"].(map[string]any)
	if consignee == nil {
		t.Fatal("request should have 'Consignee' field")
	}
	address, _ := consignee["Address"].(map[string]any)
	if address == nil {
		t.Fatal("Consignee should have nested 'Address' field")
	}
	if address["Name1"] != "Jan Kowalski" {
		t.Errorf("Consignee.Address.Name1 = %v, want Jan Kowalski", address["Name1"])
	}
	// ZIPCode (not ZipCode) per GLS API spec
	if address["ZIPCode"] != "00-001" {
		t.Errorf("Consignee.Address.ZIPCode = %v, want 00-001 (must use ZIPCode not ZipCode)", address["ZIPCode"])
	}
	// eMail (not Email) per GLS API spec
	if address["eMail"] != "jan@test.pl" {
		t.Errorf("Consignee.Address.eMail = %v, want jan@test.pl (must use eMail not Email)", address["eMail"])
	}

	// ShippingUnit (not Parcels) per GLS API spec
	units, _ := gotBody["ShippingUnit"].([]any)
	if len(units) == 0 {
		t.Error("request should have 'ShippingUnit' array (not 'Parcels')")
	}
}

func TestSpec_CreateShipment_ResponsePrintDataAtCreatedShipmentLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// PrintData is at CreatedShipment level (sibling of ParcelData), NOT inside ParcelData
		w.Write([]byte(`{
			"CreatedShipment": {
				"ShipmentReference": "ORDER-001",
				"ParcelData": [
					{"TrackID": "GLS-TRK-001"}
				],
				"PrintData": [
					{"Data": "JVBERi0xLjQ=", "Sequence": 1}
				]
			}
		}`))
	}))
	defer srv.Close()

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product:      "PARCEL",
		Consignee:    Consignee{Address: ConsigneeAddress{Name1: "Test"}},
		ShippingUnit: []ShipmentUnit{{Weight: 1.0}},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if len(resp.TrackIDs) == 0 {
		t.Error("response should contain TrackIDs from CreatedShipment.ParcelData")
	}
	if resp.TrackIDs[0] != "GLS-TRK-001" {
		t.Errorf("TrackIDs[0] = %q, want GLS-TRK-001", resp.TrackIDs[0])
	}
	// PrintData must come from CreatedShipment.PrintData[].Data (not inside ParcelData)
	if len(resp.PrintData) == 0 {
		t.Error("response should contain PrintData from CreatedShipment.PrintData")
	}
	if resp.PrintData[0] != "JVBERi0xLjQ=" {
		t.Errorf("PrintData[0] = %q, want JVBERi0xLjQ=", resp.PrintData[0])
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

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "TRK-001")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (GLS ShipIT uses POST for tracking)", gotMethod)
	}
	if gotPath != "/shipments/parceldetails" {
		t.Errorf("path = %q, want /shipments/parceldetails", gotPath)
	}
	// GLS API expects singular TrackID string, not an array
	trackID, ok := gotBody["TrackID"].(string)
	if !ok || trackID != "TRK-001" {
		t.Errorf("request body TrackID = %v, want singular string 'TRK-001' (not TrackIDs array)", gotBody["TrackID"])
	}
	if _, hasArray := gotBody["TrackIDs"]; hasArray {
		t.Error("should send TrackID (singular string), not TrackIDs (array)")
	}
}

// --- GetLabel ---

func TestSpec_GetLabel_ReturnsErrorNotSupported(t *testing.T) {
	c := NewClient("key", "pass", WithBaseURL("http://unused"))

	_, err := c.Shipments.GetLabel(context.Background(), "TRK-001")
	if err == nil {
		t.Fatal("GetLabel should return error — GLS does not support separate label retrieval")
	}
	// Error should explain where labels actually come from
	msg := err.Error()
	if !strings.Contains(msg, "create") || !strings.Contains(msg, "PrintData") {
		t.Errorf("error should explain label is in create response PrintData, got: %v", msg)
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

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Shipments.Cancel(context.Background(), "TRK-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (GLS ShipIT uses POST for cancel)", gotMethod)
	}
	if gotPath != "/shipments/cancel/TRK-001" {
		t.Errorf("path = %q, want /shipments/cancel/TRK-001", gotPath)
	}
}

// --- Status Mapping ---

func TestSpec_StatusMapping_IncludesAllGLSStatuses(t *testing.T) {
	// GLS ShipIT returns these statuses — all must be mapped
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

func TestSpec_CreateShipment_EmptyShippingUnit_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"At least one shipping unit is required"}`))
	}))
	defer srv.Close()

	c := NewClient("key", "pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product:      "PARCEL",
		Consignee:    Consignee{Address: ConsigneeAddress{Name1: "Test"}},
		ShippingUnit: []ShipmentUnit{},
	})
	if err == nil {
		t.Error("expected error for empty ShippingUnit, got nil")
	}
}

func TestSpec_CreateShipment_ServerUnavailable_ReturnsError(t *testing.T) {
	c := NewClient("key", "pass",
		WithBaseURL("http://localhost:1"), // unreachable
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product:      "PARCEL",
		Consignee:    Consignee{Address: ConsigneeAddress{Name1: "Test"}},
		ShippingUnit: []ShipmentUnit{{Weight: 1.0}},
	})
	if err == nil {
		t.Error("expected error for unreachable server, got nil")
	}
}
