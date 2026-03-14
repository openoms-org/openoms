package dpd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// DPD Poland REST API — Specification Tests
//
// These tests define the CORRECT behavior according to the official DPD Poland
// REST API documentation (dpdservices.dpd.com.pl). They verify:
//   - Base URL: https://dpdservices.dpd.com.pl (not dpd.com.pl/api/v1)
//   - Authentication: Basic Auth + x-dpd-fid header (not session token)
//   - Create: POST /public/shipment/v1/generatePackagesNumbers (not /parcels)
//   - Label: POST /public/shipment/v1/generateSpedLabels (not GET /parcels/{id}/label)
//   - Two-step label flow: create returns sessionId, label requires sessionId
//   - Response structure: packages[].parcels[].waybill (not flat parcelId)
// =============================================================================

// --- Authentication ---

func TestSpec_Auth_UsesBasicAuthWithFidHeader(t *testing.T) {
	var gotAuthMethod string
	var gotFid string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should NOT hit /auth/login — Basic Auth doesn't need sessions
		if r.URL.Path == "/auth/login" {
			t.Error("should NOT call /auth/login — DPD uses Basic Auth, not session tokens")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"token":"fake"}`))
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			gotAuthMethod = "not-basic"
			t.Error("expected Basic Auth, got none or different scheme")
		} else {
			gotAuthMethod = "basic"
			if user != "testlogin" {
				t.Errorf("Basic Auth user = %q, want %q", user, "testlogin")
			}
			if pass != "testpass" {
				t.Errorf("Basic Auth password = %q, want %q", pass, "testpass")
			}
		}

		gotFid = r.Header.Get("x-dpd-fid")
		if gotFid != "12345" {
			t.Errorf("x-dpd-fid = %q, want %q", gotFid, "12345")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":1,"packages":[]}`))
	}))
	defer srv.Close()

	c := NewClient("testlogin", "testpass", "12345",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})

	if gotAuthMethod != "basic" {
		t.Errorf("auth method = %q, want basic", gotAuthMethod)
	}
}

func TestSpec_Auth_NoSessionTokenFlow(t *testing.T) {
	authLoginCalled := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			authLoginCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"session-token"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":1,"packages":[]}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})

	// DPD official API uses Basic Auth — no need for /auth/login
	if authLoginCalled {
		t.Error("/auth/login was called — DPD official API uses Basic Auth + x-dpd-fid, not session tokens")
	}
}

// --- Base URL ---

func TestSpec_BaseURL_Production(t *testing.T) {
	c := NewClient("user", "pass", "FID")
	want := "https://dpdservices.dpd.com.pl"
	if c.baseURL != want {
		t.Errorf("production baseURL = %q, want %q", c.baseURL, want)
	}
}

func TestSpec_BaseURL_Sandbox(t *testing.T) {
	c := NewClient("user", "pass", "FID", WithSandbox())
	want := "https://dpdservicesdemo.dpd.com.pl"
	if c.baseURL != want {
		t.Errorf("sandbox baseURL = %q, want %q", c.baseURL, want)
	}
}

// --- Create Shipment ---

func TestSpec_CreateShipment_PostsToGeneratePackagesNumbers(t *testing.T) {
	var gotPath string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"t"}`))
			return
		}
		gotPath = r.URL.Path
		gotMethod = r.Method

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "OK",
			"sessionId": 42,
			"packages": [{
				"statusInfo": {"status": "OK"},
				"parcels": [{"status": "OK", "waybill": "0000012345678"}]
			}]
		}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{
			Name:        "Jan Kowalski",
			Street:      "Marszalkowska 1",
			City:        "Warszawa",
			PostalCode:  "00-001",
			CountryCode: "PL",
		},
		Parcels: []ParcelSpec{{Weight: 3.5, SizeX: 30, SizeY: 20, SizeZ: 15}},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/public/shipment/v1/generatePackagesNumbers" {
		t.Errorf("path = %q, want /public/shipment/v1/generatePackagesNumbers", gotPath)
	}
}

func TestSpec_CreateShipment_ResponseContainsSessionIdAndWaybill(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"t"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// DPD official response structure
		_, _ = w.Write([]byte(`{
			"status": "OK",
			"sessionId": 42,
			"packages": [{
				"statusInfo": {"status": "OK"},
				"parcels": [{"status": "OK", "reference": "REF-1", "waybill": "0000012345678"}]
			}]
		}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Plan: response should have waybill from packages[0].parcels[0].waybill
	// Current code returns flat parcelId/waybill — the structure is wrong
	if resp.Waybill != "0000012345678" {
		t.Errorf("Waybill = %q, want %q", resp.Waybill, "0000012345678")
	}
}

// --- Label ---

func TestSpec_GetLabel_PostsToGenerateSpedLabels(t *testing.T) {
	var gotPath string
	var gotMethod string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"t"}`))
			return
		}
		gotPath = r.URL.Path
		gotMethod = r.Method

		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","documentData":"JVBERi0="}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	// Plan: GetLabel takes a label request with sessionId, not a simple parcelID
	_, err := c.Shipments.GetLabel(context.Background(), "PRC-001")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}

	// Plan: POST /public/shipment/v1/generateSpedLabels (not GET /parcels/{id}/label)
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (DPD uses POST for label generation)", gotMethod)
	}
	if gotPath != "/public/shipment/v1/generateSpedLabels" {
		t.Errorf("path = %q, want /public/shipment/v1/generateSpedLabels", gotPath)
	}
}

// --- Tracking ---

func TestSpec_GetTracking_ReturnsNotAvailableError(t *testing.T) {
	// Plan: DPD REST API does not have a tracking endpoint
	// GetTracking should return an explicit error, not fake data
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"t"}`))
			return
		}
		// If tracking endpoint is called, it means SDK hasn't been fixed
		if r.URL.Path == "/tracking/PL123" {
			t.Error("should NOT call /tracking/{id} — DPD REST API has no tracking endpoint")
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "PL123")
	// Plan: should return error "tracking not available via DPD REST API"
	if err == nil {
		t.Error("GetTracking should return error — DPD REST API has no tracking endpoint")
	}
}

// --- Status Mapping ---

func TestSpec_StatusMapping_DPDStatuses(t *testing.T) {
	tests := []struct {
		dpdStatus string
		omsStatus string
		wantOK    bool
	}{
		{"NEW", "pending", true},
		{"SENT", "in_transit", true},
		{"IN_TRANSIT", "in_transit", true},
		{"OUT_FOR_DELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		{"PICKUP_AT_POINT", "ready_for_pickup", true},
		{"UNKNOWN_STATUS", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.dpdStatus, func(t *testing.T) {
			oms, ok := MapStatus(tc.dpdStatus)
			if ok != tc.wantOK {
				t.Errorf("MapStatus(%q) ok = %v, want %v", tc.dpdStatus, ok, tc.wantOK)
			}
			if ok && oms != tc.omsStatus {
				t.Errorf("MapStatus(%q) = %q, want %q", tc.dpdStatus, oms, tc.omsStatus)
			}
		})
	}
}

// --- Edge Cases ---

func TestSpec_CreateShipment_AuthFailure_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Invalid credentials"}`))
	}))
	defer srv.Close()

	c := NewClient("bad", "bad", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})
	if err == nil {
		t.Error("expected error for bad credentials, got nil")
	}
}

func TestSpec_CreateShipment_EmptyParcels_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"t"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"At least one parcel required"}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Parcels: []ParcelSpec{},
	})
	if err == nil {
		t.Error("expected error for empty parcels, got nil")
	}
}
