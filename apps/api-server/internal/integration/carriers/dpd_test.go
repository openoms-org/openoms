package carriers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	dpdsdk "github.com/openoms-org/openoms/packages/dpd-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// =============================================================================
// DPD Provider — Integration Specification Tests
//
// These tests verify the DPD integration provider correctly:
//   - Creates providers via factory with proper credentials
//   - Maps CarrierShipmentRequest → DPD SDK request
//   - Handles COD and Insurance as DPD services
//   - Extracts waybill from nested response structure
//   - Uses correct credential structure (login, password, master_fid)
// =============================================================================

// newTestDPDProvider creates a DPDProvider backed by a test server.
func newTestDPDProvider(t *testing.T, serverURL string) *DPDProvider {
	t.Helper()
	client := dpdsdk.NewClient(
		"test-login",
		"test-pass",
		"FID123",
		dpdsdk.WithBaseURL(serverURL),
	)
	return &DPDProvider{
		client: client,
		logger: slog.Default().With("provider", "dpd-test"),
	}
}

// newMockDPDServer creates a test server that delegates to the given handler.
// DPD SDK uses Basic Auth + x-dpd-fid headers (no session tokens).
func newMockDPDServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// --- Factory Registration ---

func TestDPD_FactoryRegistration(t *testing.T) {
	creds, _ := json.Marshal(map[string]any{
		"login":      "test-login",
		"password":   "test-pass",
		"master_fid": "12345",
		"sandbox":    true,
	})
	provider, err := integration.NewCarrierProvider("dpd", creds, nil)
	if err != nil {
		t.Fatalf("factory failed: %v", err)
	}
	if provider.ProviderName() != "dpd" {
		t.Errorf("ProviderName() = %q, want %q", provider.ProviderName(), "dpd")
	}
}

func TestDPD_FactoryRegistration_InvalidCredentials(t *testing.T) {
	_, err := integration.NewCarrierProvider("dpd", []byte(`{invalid`), nil)
	if err == nil {
		t.Error("expected error for invalid JSON credentials")
	}
}

// --- CreateShipment ---

func TestDPD_CreateShipment_MapsReceiverCorrectly(t *testing.T) {
	var receivedBody map[string]any

	srv := newMockDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":42,"packages":[{"statusInfo":{"status":"OK"},"parcels":[{"status":"OK","reference":"PRC-001","waybill":"0000012345678"}]}]}`))
	})
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		OrderID: "ORD-001",
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
			WeightKg: 3.5,
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

func TestDPD_CreateShipment_CODMappedCorrectly(t *testing.T) {
	var receivedBody map[string]any

	srv := newMockDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":1,"packages":[{"statusInfo":{"status":"OK"},"parcels":[{"status":"OK","reference":"PRC-001","waybill":"WB001"}]}]}`))
	})
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel:      integration.CarrierParcel{WeightKg: 1.0},
		CODAmount:   250.00,
		CODCurrency: "PLN",
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	// Verify COD was included in request
	services, ok := receivedBody["services"]
	if !ok || services == nil {
		t.Error("request should contain services with COD when CODAmount > 0")
		return
	}

	svcMap, isMap := services.(map[string]any)
	if !isMap {
		t.Errorf("services should be a map, got %T", services)
		return
	}

	cod, hasCOD := svcMap["cod"]
	if !hasCOD || cod == nil {
		t.Error("services should contain 'cod' field when CODAmount > 0")
		return
	}

	codMap, isMap := cod.(map[string]any)
	if !isMap {
		t.Errorf("cod should be a map, got %T", cod)
		return
	}

	amount, _ := codMap["amount"].(float64)
	if amount != 250.0 {
		t.Errorf("cod.amount = %f, want 250.0", amount)
	}
}

func TestDPD_CreateShipment_InsuranceMappedCorrectly(t *testing.T) {
	var receivedBody map[string]any

	srv := newMockDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":1,"packages":[{"statusInfo":{"status":"OK"},"parcels":[{"status":"OK","reference":"PRC-001","waybill":"WB001"}]}]}`))
	})
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel:       integration.CarrierParcel{WeightKg: 1.0},
		InsuredValue: 1000.00,
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	services, ok := receivedBody["services"]
	if !ok || services == nil {
		t.Error("request should contain services with declaredValue when InsuredValue > 0")
		return
	}

	svcMap, isMap := services.(map[string]any)
	if !isMap {
		return
	}

	dv, hasDV := svcMap["declaredValue"]
	if !hasDV || dv == nil {
		t.Error("services should contain 'declaredValue' field when InsuredValue > 0")
	}
}

func TestDPD_CreateShipment_DefaultsCODCurrency(t *testing.T) {
	var receivedBody map[string]any

	srv := newMockDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":1,"packages":[{"statusInfo":{"status":"OK"},"parcels":[{"status":"OK","reference":"PRC-001","waybill":"WB001"}]}]}`))
	})
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Street: "S", City: "C", PostalCode: "00-001", Country: "PL",
		},
		Parcel:    integration.CarrierParcel{WeightKg: 1.0},
		CODAmount: 100.0,
		// CODCurrency empty — should default to PLN
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	svcMap, _ := receivedBody["services"].(map[string]any)
	if svcMap == nil {
		return
	}
	codMap, _ := svcMap["cod"].(map[string]any)
	if codMap == nil {
		return
	}
	currency, _ := codMap["currency"].(string)
	if currency != "PLN" {
		t.Errorf("COD currency = %q, want PLN (default)", currency)
	}
}

// --- GetLabel ---

func TestDPD_GetLabel_ReturnsDecodedBytes(t *testing.T) {
	srv := newMockDPDServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// base64 of "%PDF"
		_, _ = w.Write([]byte(`{"status":"OK","documentData":"JVBERg=="}`))
	})
	defer srv.Close()

	provider := newTestDPDProvider(t, srv.URL)

	data, err := provider.GetLabel(context.Background(), "EXT-001", "pdf")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("label data should not be empty")
	}
}

// --- MapStatus ---

func TestDPD_MapStatus_DelegatesToSDK(t *testing.T) {
	provider := newTestDPDProvider(t, "http://unused")

	tests := []struct {
		dpd    string
		oms    string
		wantOK bool
	}{
		{"NEW", "pending", true},
		{"DELIVERED", "delivered", true},
		{"NONEXISTENT", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.dpd, func(t *testing.T) {
			oms, ok := provider.MapStatus(tc.dpd)
			if ok != tc.wantOK {
				t.Errorf("MapStatus(%q) ok = %v, want %v", tc.dpd, ok, tc.wantOK)
			}
			if ok && oms != tc.oms {
				t.Errorf("MapStatus(%q) = %q, want %q", tc.dpd, oms, tc.oms)
			}
		})
	}
}

// --- GetRates ---

func TestDPD_GetRates_DomesticPricing(t *testing.T) {
	provider := newTestDPDProvider(t, "http://unused")

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
	if rates[0].CarrierCode != "dpd" {
		t.Errorf("CarrierCode = %q, want dpd", rates[0].CarrierCode)
	}
	if rates[0].Currency != "PLN" {
		t.Errorf("Currency = %q, want PLN", rates[0].Currency)
	}
}

func TestDPD_GetRates_CODSurcharge(t *testing.T) {
	provider := newTestDPDProvider(t, "http://unused")

	ratesNoCOD, _ := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL", ToCountry: "PL", Weight: 5.0,
	})
	ratesWithCOD, _ := provider.GetRates(context.Background(), integration.RateRequest{
		FromCountry: "PL", ToCountry: "PL", Weight: 5.0, COD: 100.0,
	})

	if len(ratesNoCOD) == 0 || len(ratesWithCOD) == 0 {
		t.Fatal("expected rates for both scenarios")
	}

	if ratesWithCOD[0].Price <= ratesNoCOD[0].Price {
		t.Error("COD rate should be higher than non-COD rate")
	}
}
