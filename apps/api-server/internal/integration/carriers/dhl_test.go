package carriers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	dhlsdk "github.com/openoms-org/openoms/packages/dhl-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// =============================================================================
// DHL Provider — Integration Specification Tests
//
// These tests verify the DHL integration provider correctly:
//   - Creates providers via factory with proper credentials
//   - Maps CarrierShipmentRequest → DHL SDK request (all fields)
//   - Handles COD and Insurance
//   - Defaults service type to "AH" (DHL Parcel domestic)
//   - Populates Shipper from account number
//   - Returns ExternalID, TrackingNumber, Status from response
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
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"shipmentId": "SHP-001",
			"trackingNumber": "1234567890",
			"status": "CREATED"
		}`))
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

	// Verify receiver was mapped
	receiver, _ := receivedBody["receiver"].(map[string]any)
	if receiver == nil {
		t.Fatal("request should contain 'receiver' field")
	}
	if receiver["name"] != "Jan Kowalski" {
		t.Errorf("receiver.name = %v, want Jan Kowalski", receiver["name"])
	}
	if receiver["city"] != "Warszawa" {
		t.Errorf("receiver.city = %v, want Warszawa", receiver["city"])
	}

	// Verify piece dimensions
	piece, _ := receivedBody["piece"].(map[string]any)
	if piece == nil {
		t.Fatal("request should contain 'piece' field")
	}
	if piece["weight"] != 2.5 {
		t.Errorf("piece.weight = %v, want 2.5", piece["weight"])
	}
}

func TestDHL_CreateShipment_DefaultsServiceTypeToAH(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"shipmentId":"S","trackingNumber":"T","status":"CREATED"}`))
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

	svcType, _ := receivedBody["serviceType"].(string)
	if svcType != "AH" {
		t.Errorf("serviceType = %q, want AH (default)", svcType)
	}
}

func TestDHL_CreateShipment_CODMappedCorrectly(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"shipmentId":"S","trackingNumber":"T","status":"CREATED"}`))
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

	cod, ok := receivedBody["cod"]
	if !ok || cod == nil {
		t.Error("request should contain 'cod' when CODAmount > 0")
		return
	}

	codMap, isMap := cod.(map[string]any)
	if !isMap {
		t.Errorf("cod should be a map, got %T", cod)
		return
	}

	amount, _ := codMap["amount"].(float64)
	if amount != 200.0 {
		t.Errorf("cod.amount = %f, want 200.0", amount)
	}
	currency, _ := codMap["currency"].(string)
	if currency != "PLN" {
		t.Errorf("cod.currency = %q, want PLN", currency)
	}
}

func TestDHL_CreateShipment_InsuranceMappedCorrectly(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"shipmentId":"S","trackingNumber":"T","status":"CREATED"}`))
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

	ins, ok := receivedBody["insurance"]
	if !ok || ins == nil {
		t.Error("request should contain 'insurance' when InsuredValue > 0")
		return
	}

	insMap, isMap := ins.(map[string]any)
	if !isMap {
		return
	}

	amount, _ := insMap["amount"].(float64)
	if amount != 5000.0 {
		t.Errorf("insurance.amount = %f, want 5000.0", amount)
	}
}

func TestDHL_CreateShipment_AccountNumberInRequest(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"shipmentId":"S","trackingNumber":"T","status":"CREATED"}`))
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

	acct, _ := receivedBody["shipperAccount"].(string)
	if acct != "ACC123" {
		t.Errorf("shipperAccount = %q, want ACC123", acct)
	}
}

// --- GetLabel ---

func TestDHL_GetLabel_ReturnsDecodedPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// base64 of "%PDF"
		w.Write([]byte(`{"labelData":"JVBERg==","labelFormat":"PDF"}`))
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"shipmentId": "SHP-001",
			"trackingNumber": "1234567890",
			"events": [
				{"status":"PICKED_UP","location":"Warszawa","details":"Collected"},
				{"status":"DELIVERED","location":"Krakow","details":"Delivered"}
			]
		}`))
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
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	err := provider.CancelShipment(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("CancelShipment() error: %v", err)
	}

	if gotPath == "" {
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
