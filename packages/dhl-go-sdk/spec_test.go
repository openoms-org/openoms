package dhl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// DHL24 WebAPI2 (SOAP) — Specification Tests
//
// These tests define the CORRECT behavior according to the official DHL24
// WebAPI2 documentation (dhl24.com.pl/webapi2). They verify:
//   - Base URL: https://dhl24.com.pl/webapi2 (not api-pl.dhl.com)
//   - Transport: SOAP/XML (not JSON REST)
//   - Content-Type: text/xml (not application/json)
//   - AuthData embedded in each SOAP request
//   - Create: SOAP createShipments method
//   - Label: SOAP getLabels method
//   - Tracking: SOAP getTrackAndTraceInfo method
//   - Cancel: SOAP deleteShipment method
// =============================================================================

// --- Base URL ---

func TestSpec_BaseURL_Production(t *testing.T) {
	c := NewClient("user", "pass", "ACC123")
	want := "https://dhl24.com.pl/webapi2"
	if c.baseURL != want {
		t.Errorf("production baseURL = %q, want %q", c.baseURL, want)
	}
}

// --- SOAP Transport ---

func TestSpec_CreateShipment_SendsSOAPXML(t *testing.T) {
	var gotContentType string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		// Return a valid response so the SDK doesn't error on response parsing
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<createShipmentsResponse>
					<shipmentId>SHP-001</shipmentId>
					<trackingNumber>1234567890</trackingNumber>
				</createShipmentsResponse>
			</soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		ShipperAccount: "ACC",
		Receiver: Receiver{
			Name:       "Jan Kowalski",
			Street:     "Marszalkowska 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		Piece:       Piece{Weight: 2.5},
		ServiceType: "AH",
	})

	// Plan: DHL24 WebAPI2 uses SOAP — Content-Type must be text/xml
	if !strings.Contains(gotContentType, "text/xml") {
		t.Errorf("Content-Type = %q, want text/xml (DHL24 uses SOAP)", gotContentType)
	}

	// Plan: body must contain SOAP envelope
	if !strings.Contains(gotBody, "Envelope") && !strings.Contains(gotBody, "envelope") {
		t.Error("request body should contain SOAP Envelope (DHL24 uses SOAP, not JSON)")
	}

	// Plan: body must contain AuthData with credentials
	if !strings.Contains(gotBody, "authData") && !strings.Contains(gotBody, "AuthData") {
		t.Error("SOAP request should contain AuthData element with credentials")
	}

	// Should NOT be JSON
	if strings.HasPrefix(strings.TrimSpace(gotBody), "{") {
		t.Error("request body is JSON — DHL24 WebAPI2 requires SOAP/XML")
	}
}

func TestSpec_CreateShipment_ContainsSOAPAction(t *testing.T) {
	var gotSOAPAction string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSOAPAction = r.Header.Get("SOAPAction")

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body><createShipmentsResponse></createShipmentsResponse></soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		Receiver: Receiver{Name: "Test"},
		Piece:    Piece{Weight: 1.0},
	})

	// SOAP requests should include SOAPAction header
	if gotSOAPAction == "" {
		t.Error("expected SOAPAction header for SOAP request")
	}
}

// --- GetLabel ---

func TestSpec_GetLabel_UsesSOAPGetLabelsMethod(t *testing.T) {
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<getLabelsResponse>
					<labelData>JVBERi0xLjQ=</labelData>
				</getLabelsResponse>
			</soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.GetLabel(context.Background(), "SHP-001")

	// Plan: GetLabel should call SOAP getLabels method
	if !strings.Contains(gotBody, "getLabels") && !strings.Contains(gotBody, "GetLabels") {
		t.Error("GetLabel should use SOAP getLabels method, not REST endpoint")
	}
}

// --- GetTracking ---

func TestSpec_GetTracking_UsesSOAPMethod(t *testing.T) {
	var gotBody string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<getTrackAndTraceInfoResponse>
					<events></events>
				</getTrackAndTraceInfoResponse>
			</soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.GetTracking(context.Background(), "1234567890")

	// Plan: SOAP uses POST for all operations
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (SOAP uses POST)", gotMethod)
	}

	// Plan: body should contain getTrackAndTraceInfo SOAP method
	if !strings.Contains(gotBody, "getTrackAndTrace") && !strings.Contains(gotBody, "GetTrackAndTrace") {
		t.Error("GetTracking should use SOAP getTrackAndTraceInfo method")
	}
}

// --- Cancel ---

func TestSpec_Cancel_UsesSOAPDeleteShipmentMethod(t *testing.T) {
	var gotBody string
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body><deleteShipmentResponse></deleteShipmentResponse></soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_ = c.Shipments.Cancel(context.Background(), "SHP-001")

	// Plan: SOAP uses POST for all operations
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST (SOAP uses POST for cancel)", gotMethod)
	}

	// Plan: body should contain deleteShipment SOAP method
	if !strings.Contains(gotBody, "deleteShipment") && !strings.Contains(gotBody, "DeleteShipment") {
		t.Error("Cancel should use SOAP deleteShipment method")
	}
}

// --- AuthData ---

func TestSpec_AuthData_EmbeddedInEveryRequest(t *testing.T) {
	var bodies []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body><response></response></soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("dhluser", "dhlpass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	// Make multiple different API calls
	_, _ = c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		Receiver: Receiver{Name: "Test"},
		Piece:    Piece{Weight: 1.0},
	})
	_, _ = c.Shipments.GetLabel(context.Background(), "SHP-001")
	_, _ = c.Shipments.GetTracking(context.Background(), "TRK-001")

	// Plan: every SOAP request should contain AuthData
	for i, body := range bodies {
		if !strings.Contains(body, "dhluser") {
			t.Errorf("request %d: should contain username in AuthData", i)
		}
		if !strings.Contains(body, "dhlpass") {
			t.Errorf("request %d: should contain password in AuthData", i)
		}
	}
}

// --- Status Mapping ---

func TestSpec_StatusMapping_DHL24Statuses(t *testing.T) {
	tests := []struct {
		dhlStatus string
		omsStatus string
		wantOK    bool
	}{
		{"CREATED", "created", true},
		{"PICKED_UP", "picked_up", true},
		{"IN_TRANSIT", "in_transit", true},
		{"OUT_FOR_DELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		{"FAILED", "failed", true},
		{"LABEL_CREATED", "label_ready", true},
		{"CUSTOMS", "in_transit", true},
		{"NONEXISTENT_STATUS", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.dhlStatus, func(t *testing.T) {
			oms, ok := MapStatus(tc.dhlStatus)
			if ok != tc.wantOK {
				t.Errorf("MapStatus(%q) ok = %v, want %v", tc.dhlStatus, ok, tc.wantOK)
			}
			if ok && oms != tc.omsStatus {
				t.Errorf("MapStatus(%q) = %q, want %q", tc.dhlStatus, oms, tc.omsStatus)
			}
		})
	}
}

// --- Edge Cases ---

func TestSpec_CreateShipment_ReturnsErrorOnSOAPFault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<soap:Fault>
					<faultcode>soap:Server</faultcode>
					<faultstring>Invalid shipment data</faultstring>
				</soap:Fault>
			</soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{})
	if err == nil {
		t.Error("expected error for SOAP fault, got nil")
	}
}
