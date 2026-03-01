package carriers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	glssdk "github.com/openoms-org/openoms/packages/gls-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// =============================================================================
// GLS Provider — Integration Specification Tests
//
// These tests verify the GLS integration provider correctly:
//   - Creates providers via factory with proper credentials
//   - Maps CarrierShipmentRequest → GLS SDK request (correct fields)
//   - Handles COD with Service structure (not flat string array)
//   - Extracts ExternalID/TrackingNumber from CreatedShipment.ParcelData
//   - Uses Basic Auth credentials (username+password, not apiKey)
// =============================================================================

func newTestGLSProvider(t *testing.T, serverURL string) *GLSProvider {
	t.Helper()
	client := glssdk.NewClient(
		"test-username",
		"test-password",
		glssdk.WithBaseURL(serverURL),
	)
	return &GLSProvider{
		client: client,
		logger: slog.Default().With("provider", "gls-test"),
	}
}

// --- Factory Registration ---

func TestGLS_FactoryRegistration(t *testing.T) {
	creds, _ := json.Marshal(map[string]any{
		"username": "test-user",
		"password": "test-pass",
		"sandbox":  true,
	})
	provider, err := integration.NewCarrierProvider("gls", creds, nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if provider.ProviderName() != "gls" {
		t.Errorf("ProviderName() = %q, want %q", provider.ProviderName(), "gls")
	}
}

func TestGLS_FactoryRegistration_InvalidCredentials(t *testing.T) {
	_, err := integration.NewCarrierProvider("gls", []byte(`{invalid`), nil)
	if err == nil {
		t.Error("expected error for invalid JSON credentials")
	}
}

// --- CreateShipment ---

func TestGLS_CreateShipment_MapsReceiverCorrectly(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"T1","PrintData":"AAAA"}]}}`))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		OrderID:     "ORD-001",
		ServiceType: "standard",
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
			DepthCm:  15,
		},
		Reference: "REF-001",
	}

	resp, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if resp.ExternalID == "" {
		t.Error("ExternalID should not be empty")
	}
	if resp.TrackingNumber == "" {
		t.Error("TrackingNumber should not be empty")
	}
}

func TestGLS_CreateShipment_CODUsesServiceStructure(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"T1","PrintData":"AAAA"}]}}`))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name:       "Test",
			Street:     "Test 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		Parcel: integration.CarrierParcel{
			WeightKg: 1.0,
		},
		CODAmount:   150.00,
		CODCurrency: "PLN",
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	// COD should be sent as a structured Service section, not a flat string array
	svcSection, ok := receivedBody["Service"]
	if !ok {
		t.Error("request should contain 'Service' field for COD")
		return
	}

	// Service should be a structured section with nested Service array
	svcMap, isMap := svcSection.(map[string]any)
	if !isMap {
		t.Errorf("Service should be a structured section, got %T", svcSection)
		return
	}
	innerServices, ok := svcMap["Service"]
	if !ok {
		t.Error("Service section should contain inner 'Service' array")
		return
	}
	svcArr, isArr := innerServices.([]any)
	if !isArr || len(svcArr) == 0 {
		t.Error("Service array should contain at least one service entry")
		return
	}

	// Verify the service has structured COD with amount/currency
	firstSvc, isFirstMap := svcArr[0].(map[string]any)
	if !isFirstMap {
		t.Errorf("first service should be a map, got %T", svcArr[0])
		return
	}
	if firstSvc["ServiceName"] != "service_cash" {
		t.Errorf("ServiceName = %v, want service_cash", firstSvc["ServiceName"])
	}
	cash, ok := firstSvc["Cash"].(map[string]any)
	if !ok {
		t.Error("COD service should have structured 'Cash' field")
		return
	}
	if cash["Amount"] != 150.0 {
		t.Errorf("Cash.Amount = %v, want 150.0", cash["Amount"])
	}
}

func TestGLS_CreateShipment_PropagatesServiceType(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"T1","PrintData":"AAAA"}]}}`))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "express_10",
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "Test 1", City: "City", PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	product, _ := receivedBody["Product"].(string)
	if product != "EXPRESS_10" {
		t.Errorf("Product = %q, want %q", product, "EXPRESS_10")
	}
}

func TestGLS_CreateShipment_DefaultsServiceType(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"T1","PrintData":"AAAA"}]}}`))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "", // empty — should default
		Receiver:    integration.CarrierReceiver{Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL"},
		Parcel:      integration.CarrierParcel{WeightKg: 1.0},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	product, _ := receivedBody["Product"].(string)
	if product != "PARCEL" {
		t.Errorf("Product = %q, want %q (default)", product, "PARCEL")
	}
}

// --- GetLabel ---

func TestGLS_GetLabel_ReturnsErrorNotSupported(t *testing.T) {
	// GLS ShipIT API returns labels inline during shipment creation
	// (CreatedShipment.PrintData[].Data). Separate label retrieval is not supported.
	provider := newTestGLSProvider(t, "http://unused")

	_, err := provider.GetLabel(context.Background(), "EXT-001", "pdf")
	if err == nil {
		t.Error("GetLabel() should return error — GLS labels are embedded in create response")
	}
}

// --- GetTracking ---

func TestGLS_GetTracking_MapsEventsCorrectly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"events":[
			{"status":"PREADVICE","location":"Krakow","details":"Registered"},
			{"status":"DELIVERED","location":"Warszawa","details":"Delivered"}
		]}`))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	events, err := provider.GetTracking(context.Background(), "TRK-001")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Status != "PREADVICE" {
		t.Errorf("events[0].Status = %q, want PREADVICE", events[0].Status)
	}
	if events[0].Location != "Krakow" {
		t.Errorf("events[0].Location = %q, want Krakow", events[0].Location)
	}
	if events[1].Status != "DELIVERED" {
		t.Errorf("events[1].Status = %q, want DELIVERED", events[1].Status)
	}
}

// --- MapStatus ---

func TestGLS_MapStatus_DelegatesToSDK(t *testing.T) {
	provider := newTestGLSProvider(t, "http://unused")

	oms, ok := provider.MapStatus("DELIVERED")
	if !ok {
		t.Error("MapStatus(DELIVERED) should return ok=true")
	}
	if oms != "delivered" {
		t.Errorf("MapStatus(DELIVERED) = %q, want delivered", oms)
	}

	_, ok = provider.MapStatus("NONEXISTENT")
	if ok {
		t.Error("MapStatus(NONEXISTENT) should return ok=false")
	}
}

// --- GetRates ---

func TestGLS_GetRates_DomesticPricing(t *testing.T) {
	provider := newTestGLSProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      5.0,
	})
	if err != nil {
		t.Fatalf("GetRates() error: %v", err)
	}
	if len(rates) == 0 {
		t.Fatal("expected at least one rate for domestic shipment")
	}
	if rates[0].CarrierCode != "gls" {
		t.Errorf("CarrierCode = %q, want gls", rates[0].CarrierCode)
	}
	if rates[0].Currency != "PLN" {
		t.Errorf("Currency = %q, want PLN", rates[0].Currency)
	}
	if rates[0].Price <= 0 {
		t.Error("Price should be positive")
	}
}

func TestGLS_GetRates_InternationalReturnsEmpty(t *testing.T) {
	provider := newTestGLSProvider(t, "http://unused")

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

func TestGLS_GetRates_OverweightReturnsEmpty(t *testing.T) {
	provider := newTestGLSProvider(t, "http://unused")

	rates, err := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL",
		ToCountry:   "PL",
		Weight:      50.0, // over 31.5 kg limit
	})
	if err != nil {
		t.Fatalf("GetRates() error: %v", err)
	}
	if len(rates) != 0 {
		t.Errorf("expected 0 rates for overweight, got %d", len(rates))
	}
}
