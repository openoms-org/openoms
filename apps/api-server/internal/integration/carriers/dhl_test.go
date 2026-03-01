package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dhlsdk "github.com/openoms-org/openoms/packages/dhl-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// =============================================================================
// DHL Provider — Integration Specification Tests
//
// These tests verify the DHL integration provider correctly:
//   - Creates providers via factory with proper credentials
//   - Maps CarrierShipmentRequest → DHL SOAP request (all fields)
//   - Handles COD and Insurance in SOAP XML
//   - Defaults service type to "AH" (DHL Parcel domestic)
//   - Populates ShipperAccountNumber from account number
//   - Returns ExternalID, TrackingNumber, Status from SOAP response
// =============================================================================

func newTestDHLProvider(t *testing.T, serverURL string) *DHLProvider {
	t.Helper()
	client := dhlsdk.NewClient(
		"test-user",
		"test-pass",
		"ACC123",
		dhlsdk.WithBaseURL(serverURL),
	)
	return &DHLProvider{
		client: client,
		logger: slog.Default().With("provider", "dhl-test"),
	}
}

// SOAP XML response helpers

func dhlCreateSOAPResponse(shipmentID, tracking string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
  <Body>
    <createShipmentsResponse>
      <shipmentId>%s</shipmentId>
      <trackingNumber>%s</trackingNumber>
      <status>CREATED</status>
    </createShipmentsResponse>
  </Body>
</Envelope>`, shipmentID, tracking)
}

func dhlLabelsSOAPResponse(labelData string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
  <Body>
    <getLabelsResponse>
      <labelData>%s</labelData>
    </getLabelsResponse>
  </Body>
</Envelope>`, labelData)
}

func dhlTrackingSOAPResponse(events string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
  <Body>
    <getTrackAndTraceInfoResponse>
      <events>%s</events>
    </getTrackAndTraceInfoResponse>
  </Body>
</Envelope>`, events)
}

func dhlEmptySOAPResponse() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
  <Body>
    <deleteShipmentResponse/>
  </Body>
</Envelope>`
}

// --- Factory Registration ---

func TestDHL_FactoryRegistration(t *testing.T) {
	creds, _ := json.Marshal(map[string]any{
		"username":       "test-user",
		"password":       "test-pass",
		"account_number": "ACC123",
		"sandbox":        true,
	})
	provider, err := integration.NewCarrierProvider("dhl", creds, nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if provider.ProviderName() != "dhl" {
		t.Errorf("ProviderName() = %q, want %q", provider.ProviderName(), "dhl")
	}
}

func TestDHL_FactoryRegistration_InvalidCredentials(t *testing.T) {
	_, err := integration.NewCarrierProvider("dhl", []byte(`{invalid`), nil)
	if err == nil {
		t.Error("expected error for invalid JSON credentials")
	}
}

// --- CreateShipment ---

func TestDHL_CreateShipment_MapsAllFields(t *testing.T) {
	var requestXML string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlCreateSOAPResponse("SHP-001", "1234567890"))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		OrderID:     "ORD-001",
		ServiceType: "AH",
		Receiver: integration.CarrierReceiver{
			Name:       "Jan Kowalski",
			Email:      "jan@test.pl",
			Phone:      "500100200",
			Street:     "Marszalkowska 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		Parcel: integration.CarrierParcel{
			WeightKg: 2.5,
			WidthCm:  30,
			HeightCm: 20,
			DepthCm:  40,
		},
		Reference: "REF-001",
	}

	resp, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if resp.ExternalID != "SHP-001" {
		t.Errorf("ExternalID = %q, want SHP-001", resp.ExternalID)
	}
	if resp.TrackingNumber != "1234567890" {
		t.Errorf("TrackingNumber = %q, want 1234567890", resp.TrackingNumber)
	}
	if resp.Status != "CREATED" {
		t.Errorf("Status = %q, want CREATED", resp.Status)
	}

	// Verify receiver was mapped into SOAP XML
	if !strings.Contains(requestXML, "<name>Jan Kowalski</name>") {
		t.Error("SOAP request should contain receiver name")
	}
	if !strings.Contains(requestXML, "<city>Warszawa</city>") {
		t.Error("SOAP request should contain receiver city")
	}
	if !strings.Contains(requestXML, "<postalCode>00-001</postalCode>") {
		t.Error("SOAP request should contain receiver postalCode")
	}

	// Verify piece dimensions
	if !strings.Contains(requestXML, "<weight>2.5</weight>") {
		t.Error("SOAP request should contain piece weight")
	}
}

func TestDHL_CreateShipment_DefaultsServiceTypeToAH(t *testing.T) {
	var requestXML string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlCreateSOAPResponse("S", "T"))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "", // empty — should default to AH
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if !strings.Contains(requestXML, "<serviceType>AH</serviceType>") {
		t.Error("SOAP request should contain serviceType=AH as default")
	}
}

func TestDHL_CreateShipment_CODMappedCorrectly(t *testing.T) {
	var requestXML string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlCreateSOAPResponse("S", "T"))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel:      integration.CarrierParcel{WeightKg: 1.0},
		CODAmount:   200.00,
		CODCurrency: "PLN",
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if !strings.Contains(requestXML, "<cod>") {
		t.Error("SOAP request should contain <cod> element when CODAmount > 0")
	}
	if !strings.Contains(requestXML, "<amount>200</amount>") {
		t.Error("SOAP request should contain COD amount")
	}
	if !strings.Contains(requestXML, "<currency>PLN</currency>") {
		t.Error("SOAP request should contain COD currency")
	}
}

func TestDHL_CreateShipment_InsuranceMappedCorrectly(t *testing.T) {
	var requestXML string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlCreateSOAPResponse("S", "T"))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel:       integration.CarrierParcel{WeightKg: 1.0},
		InsuredValue: 5000.00,
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if !strings.Contains(requestXML, "<insurance>") {
		t.Error("SOAP request should contain <insurance> element when InsuredValue > 0")
	}
	if !strings.Contains(requestXML, "<amount>5000</amount>") {
		t.Error("SOAP request should contain insurance amount")
	}
}

func TestDHL_CreateShipment_AccountNumberInRequest(t *testing.T) {
	var requestXML string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlCreateSOAPResponse("S", "T"))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if !strings.Contains(requestXML, "<shipperAccountNumber>ACC123</shipperAccountNumber>") {
		t.Error("SOAP request should contain shipperAccountNumber=ACC123")
	}
}

// --- GetLabel ---

func TestDHL_GetLabel_ReturnsDecodedPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		// base64 of "%PDF"
		fmt.Fprint(w, dhlLabelsSOAPResponse("JVBERg=="))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	data, err := provider.GetLabel(context.Background(), "SHP-001", "pdf")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("label data should not be empty")
	}
}

// --- GetTracking ---

func TestDHL_GetTracking_MapsEventsCorrectly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		events := `
        <event><status>PICKED_UP</status><location>Warszawa</location><details>Collected</details></event>
        <event><status>DELIVERED</status><location>Krakow</location><details>Delivered</details></event>`
		fmt.Fprint(w, dhlTrackingSOAPResponse(events))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	events, err := provider.GetTracking(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Status != "PICKED_UP" {
		t.Errorf("events[0].Status = %q, want PICKED_UP", events[0].Status)
	}
	if events[0].Location != "Warszawa" {
		t.Errorf("events[0].Location = %q, want Warszawa", events[0].Location)
	}
	if events[1].Status != "DELIVERED" {
		t.Errorf("events[1].Status = %q, want DELIVERED", events[1].Status)
	}
}

// --- CancelShipment ---

func TestDHL_CancelShipment_CallsSDK(t *testing.T) {
	var gotRequest bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		gotRequest = true
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlEmptySOAPResponse())
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	err := provider.CancelShipment(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("CancelShipment() error: %v", err)
	}

	if !gotRequest {
		t.Error("expected a request to be made for cancel")
	}
}

// --- MapStatus ---

func TestDHL_MapStatus_DelegatesToSDK(t *testing.T) {
	provider := newTestDHLProvider(t, "http://unused")

	tests := []struct {
		dhl    string
		oms    string
		wantOK bool
	}{
		{"CREATED", "created", true},
		{"DELIVERED", "delivered", true},
		{"IN_TRANSIT", "in_transit", true},
		{"NONEXISTENT", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.dhl, func(t *testing.T) {
			oms, ok := provider.MapStatus(tc.dhl)
			if ok != tc.wantOK {
				t.Errorf("MapStatus(%q) ok = %v, want %v", tc.dhl, ok, tc.wantOK)
			}
			if ok && oms != tc.oms {
				t.Errorf("MapStatus(%q) = %q, want %q", tc.dhl, oms, tc.oms)
			}
		})
	}
}

// --- GetRates ---

func TestDHL_GetRates_DomesticPricing_WeightTiers(t *testing.T) {
	provider := newTestDHLProvider(t, "http://unused")

	// Light package — should get multiple tier options
	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      3.0,
	})
	if err != nil {
		t.Fatalf("GetRates() error: %v", err)
	}
	if len(rates) == 0 {
		t.Fatal("expected at least one rate")
	}

	// All rates should be DHL
	for _, r := range rates {
		if r.CarrierCode != "dhl" {
			t.Errorf("CarrierCode = %q, want dhl", r.CarrierCode)
		}
	}

	// Heavier package should have fewer tier options
	heavyRates, _ := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      25.0,
	})
	if len(heavyRates) >= len(rates) {
		t.Error("heavy package should have fewer tier options than light package")
	}
}

func TestDHL_GetRates_InternationalReturnsEmpty(t *testing.T) {
	provider := newTestDHLProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "DE",
		Weight:      5.0,
	})
	if err != nil {
		t.Fatalf("GetRates() error: %v", err)
	}
	if len(rates) != 0 {
		t.Errorf("expected 0 rates for international, got %d", len(rates))
	}
}

func TestDHL_SupportsPickupPoints_ReturnsFalse(t *testing.T) {
	provider := newTestDHLProvider(t, "http://unused")

	if provider.SupportsPickupPoints() {
		t.Error("DHL should not support pickup points")
	}
}
