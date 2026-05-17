package carriers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// =============================================================================
// DPD Provider — Production Readiness Specification Tests
//
// These tests define REQUIRED behavior for DPD production use:
//   - Service type mapping: dpd_classic → STANDARD, dpd_pickup → PICKUP
//   - Target point validation for dpd_pickup service
//   - Shipper address mapped to DPD Sender in API request
//   - Tracking uses DPD InfoServices SOAP while parcel creation/labels stay on DPD Services REST
//   - Cancel returns a clear unsupported-operation error
//
// Reference: dhl_production_test.go (same pattern, DHL-specific checks).
// DPD shipment creation uses JSON REST API; DPD tracking uses InfoServices SOAP.
// =============================================================================

// dpdOKResponse is a minimal valid DPD create-parcel response for test mocks.
const dpdOKResponse = `{"status":"OK","sessionId":42,"packages":[{"statusInfo":{"status":"OK"},"parcels":[{"status":"OK","reference":"REF-001","waybill":"0000012345678"}]}]}`

// --- Service Type Mapping ---

func TestDPD_CreateShipment_ServiceTypeMapping(t *testing.T) {
	// Frontend sends dpd_classic / dpd_pickup as service types.
	// The provider must translate these to DPD API codes before sending.
	tests := []struct {
		name            string
		inputType       string
		targetPoint     string
		wantServiceType string
	}{
		{"dpd_classic maps to STANDARD", "dpd_classic", "", "STANDARD"},
		{"dpd_pickup maps to PICKUP", "dpd_pickup", "PL12345", "PICKUP"},
		{"empty defaults to STANDARD", "", "", "STANDARD"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedBody map[string]any

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &receivedBody)

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(dpdOKResponse))
			}))
			defer srv.Close()

			provider := newTestDPDProvider(t, srv.URL)

			req := integration.CarrierShipmentRequest{
				ServiceType: tc.inputType,
				TargetPoint: tc.targetPoint,
				Receiver: integration.CarrierReceiver{
					Name: "Test Receiver", Phone: "500100200",
					Street: "Testowa 1", City: "Warszawa",
					PostalCode: "00-001", Country: "PL",
				},
				Parcel: integration.CarrierParcel{WeightKg: 1.0},
			}

			_, err := provider.CreateShipment(context.Background(), req)
			if err != nil {
				t.Fatalf("CreateShipment() error: %v", err)
			}

			serviceType, _ := receivedBody["serviceType"].(string)
			if serviceType != tc.wantServiceType {
				t.Errorf("JSON body serviceType = %q, want %q", serviceType, tc.wantServiceType)
			}
		})
	}
}

func TestDPD_CreateShipment_UnknownServiceTypeReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("API request should not be sent for unknown service type")
	}))
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "invalid_type",
		Receiver: integration.CarrierReceiver{
			Name: "Test", Phone: "500100200",
			Street: "Testowa 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err == nil {
		t.Fatal("CreateShipment() should return error for unknown service type")
	}
	if !strings.Contains(err.Error(), "unknown service type") {
		t.Errorf("error should mention 'unknown service type', got: %v", err)
	}
}

// --- Target Point for Pickup ---

func TestDPD_CreateShipment_PickupRequiresTargetPoint(t *testing.T) {
	// dpd_pickup without a target point ID should fail early, before calling
	// the DPD API, so the user gets a clear validation error.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("API request should not be sent when pickup target point is missing")
	}))
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "dpd_pickup",
		// TargetPoint intentionally empty
		Receiver: integration.CarrierReceiver{
			Name: "Test", Phone: "500100200",
			Street: "Testowa 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err == nil {
		t.Fatal("CreateShipment() should return error when dpd_pickup has no target point")
	}
	if !strings.Contains(err.Error(), "target point") {
		t.Errorf("error should mention 'target point', got: %v", err)
	}
}

func TestDPD_CreateShipment_TargetPointSentForPickup(t *testing.T) {
	// When dpd_pickup is used with a valid target point, that point ID must
	// appear in the JSON request sent to the DPD API.
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(dpdOKResponse))
	}))
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "dpd_pickup",
		TargetPoint: "PL12345",
		Receiver: integration.CarrierReceiver{
			Name: "Test", Phone: "500100200",
			Street: "Testowa 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	targetPoint, _ := receivedBody["targetPoint"].(string)
	if targetPoint != "PL12345" {
		t.Errorf("JSON body targetPoint = %q, want %q", targetPoint, "PL12345")
	}
}

// --- Shipper Address ---

func TestDPD_CreateShipment_ShipperMapsToSender(t *testing.T) {
	// When Shipper is provided in the request, the provider must map all fields
	// into the DPD Sender (Address) section of the API request.
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(dpdOKResponse))
	}))
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Shipper: &integration.CarrierSender{
			Name:       "Firma Nadawcza Sp. z o.o.",
			Phone:      "221234567",
			Email:      "wysylki@firma.pl",
			Street:     "Przemyslowa 5",
			City:       "Warszawa",
			PostalCode: "00-950",
			Country:    "PL",
		},
		Receiver: integration.CarrierReceiver{
			Name: "Jan Kowalski", Phone: "500100200",
			Street: "Marszalkowska 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 2.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	sender, ok := receivedBody["sender"].(map[string]any)
	if !ok {
		t.Fatal("request JSON should contain populated 'sender' when Shipper is provided")
	}

	if name, _ := sender["name"].(string); name != "Firma Nadawcza Sp. z o.o." {
		t.Errorf("sender.name = %q, want %q", name, "Firma Nadawcza Sp. z o.o.")
	}
	if city, _ := sender["city"].(string); city != "Warszawa" {
		t.Errorf("sender.city = %q, want %q", city, "Warszawa")
	}
	if postal, _ := sender["postalCode"].(string); postal != "00-950" {
		t.Errorf("sender.postalCode = %q, want %q", postal, "00-950")
	}
	if country, _ := sender["countryCode"].(string); country != "PL" {
		t.Errorf("sender.countryCode = %q, want %q", country, "PL")
	}
	if email, _ := sender["email"].(string); email != "wysylki@firma.pl" {
		t.Errorf("sender.email = %q, want %q", email, "wysylki@firma.pl")
	}
	if phone, _ := sender["phone"].(string); phone != "221234567" {
		t.Errorf("sender.phone = %q, want %q", phone, "221234567")
	}
}

// --- Tracking & Cancel ---

func TestDPD_GetTracking_UsesInfoServicesSOAP(t *testing.T) {
	// DPD tracking is served by InfoServices SOAP, not by the DPD Services
	// REST parcel endpoint used for shipment creation and labels.
	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		_, _ = w.Write([]byte(dpdInfoServicesProviderOKResponse))
	}))
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	events, err := provider.GetTracking(context.Background(), "0000012345678")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if !strings.Contains(gotBody, "getEventsForWaybillV1") {
		t.Fatalf("expected SOAP getEventsForWaybillV1 request, got:\n%s", gotBody)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[len(events)-1].Status != "190101" {
		t.Fatalf("newest event status = %q, want 190101", events[len(events)-1].Status)
	}
	omsStatus, ok := provider.MapStatus(events[len(events)-1].Status)
	if !ok || omsStatus != "delivered" {
		t.Fatalf("MapStatus(%q) = %q, %v; want delivered, true", events[len(events)-1].Status, omsStatus, ok)
	}
}

func TestDPD_CancelShipment_ReturnsUnsupportedError(t *testing.T) {
	// DPD does not support parcel cancellation via REST API. The provider must
	// return a clear error directing users to DPD support.
	provider := newTestDPDProvider(t, "http://unused")

	err := provider.CancelShipment(context.Background(), "0000012345678")
	if err == nil {
		t.Fatal("CancelShipment() should return error — DPD does not support cancel via REST API")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention cancellation 'not supported', got: %v", err)
	}
}
