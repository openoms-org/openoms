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
// GLS Provider — Production Readiness Specification Tests
//
// These tests define REQUIRED behavior for GLS production use:
//   - Service type mapping: standard → PARCEL, express_10 → EXPRESS_10, express_12 → EXPRESS_12
//   - Unknown service types must return error (not silently default)
//   - GetLabel must return clear error (GLS labels are inline in create response)
//   - COD defaults currency to PLN when not specified
//   - Insurance maps to service_addonliability with amount/currency
//   - Shipper address is NOT mapped to GLS ContactID (GLS limitation)
//   - Reference field maps to References array
//
// Reference: dpd_production_test.go (same pattern, DPD-specific checks).
// GLS uses JSON REST API with Basic Auth, assertions check JSON body fields.
// =============================================================================

// glsOKResponse is a minimal valid GLS create-shipment response for test mocks.
const glsOKResponse = `{"CreatedShipment":{"ShipmentReference":"REF1","ParcelData":[{"TrackID":"GLS0001234567"}],"PrintData":[{"Data":"JVBERi0xLjQK"}]}}`

// --- Service Type Mapping ---

func TestGLS_CreateShipment_ServiceTypeMapping(t *testing.T) {
	// Frontend sends standard / express_10 / express_12 as service types.
	// The provider must translate these to GLS Product codes before sending.
	tests := []struct {
		name        string
		inputType   string
		wantProduct string
	}{
		{"standard maps to PARCEL", "standard", "PARCEL"},
		{"express_10 maps to EXPRESS_10", "express_10", "EXPRESS_10"},
		{"express_12 maps to EXPRESS_12", "express_12", "EXPRESS_12"},
		{"empty defaults to PARCEL", "", "PARCEL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var receivedBody map[string]any

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &receivedBody)

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(glsOKResponse))
			}))
			defer srv.Close()

			provider := newTestGLSProvider(t, srv.URL)

			req := integration.CarrierShipmentRequest{
				ServiceType: tc.inputType,
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

			product, _ := receivedBody["Product"].(string)
			if product != tc.wantProduct {
				t.Errorf("JSON body Product = %q, want %q", product, tc.wantProduct)
			}
		})
	}
}

func TestGLS_CreateShipment_UnknownServiceTypeReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("API request should not be sent for unknown service type")
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

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

// --- GetLabel (GLS labels are inline in create response) ---

func TestGLS_GetLabel_ReturnsErrorForMissingLabel(t *testing.T) {
	// GLS labels are now persisted to storage after CreateShipment.
	// GetLabel for a non-existent tracking number should return an error.
	provider := newTestGLSProvider(t, "http://unused")

	_, err := provider.GetLabel(context.Background(), "GLS0001234567", "pdf")
	if err == nil {
		t.Fatal("GetLabel() should return error for non-existent label")
	}
}

// --- COD Service Mapping ---

func TestGLS_CreateShipment_CODDefaultsCurrencyToPLN(t *testing.T) {
	// When CODAmount is set but CODCurrency is empty, the provider must
	// default to PLN (Polish domestic shipments).
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(glsOKResponse))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Phone: "500100200",
			Street: "Testowa 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel:    integration.CarrierParcel{WeightKg: 1.0},
		CODAmount: 199.99,
		// CODCurrency intentionally empty — should default to PLN
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	// Navigate to Service.Service[0].Cash.Currency
	svcSection, ok := receivedBody["Service"].(map[string]any)
	if !ok {
		t.Fatal("request should contain 'Service' section when CODAmount > 0")
	}
	svcArr, ok := svcSection["Service"].([]any)
	if !ok || len(svcArr) == 0 {
		t.Fatal("Service section should contain at least one service entry")
	}

	var foundCash bool
	for _, s := range svcArr {
		svc, isMap := s.(map[string]any)
		if !isMap {
			continue
		}
		if svc["ServiceName"] != "service_cash" {
			continue
		}
		foundCash = true
		cash, ok := svc["Cash"].(map[string]any)
		if !ok {
			t.Fatal("service_cash should have structured 'Cash' field")
		}
		currency, _ := cash["Currency"].(string)
		if currency != "PLN" {
			t.Errorf("Cash.Currency = %q, want PLN (default)", currency)
		}
		amount, _ := cash["Amount"].(float64)
		if amount != 199.99 {
			t.Errorf("Cash.Amount = %v, want 199.99", amount)
		}
	}
	if !foundCash {
		t.Error("Service array should contain service_cash entry when CODAmount > 0")
	}
}

// --- Insurance Service Mapping ---

func TestGLS_CreateShipment_InsuranceMapsToAddOnLiability(t *testing.T) {
	// Insurance must be sent as GLS service_addonliability with structured
	// AddOnLiability field containing amount and currency.
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(glsOKResponse))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Receiver: integration.CarrierReceiver{
			Name: "Test", Phone: "500100200",
			Street: "Testowa 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel:       integration.CarrierParcel{WeightKg: 1.0},
		InsuredValue: 5000.00,
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	svcSection, ok := receivedBody["Service"].(map[string]any)
	if !ok {
		t.Fatal("request should contain 'Service' section when InsuredValue > 0")
	}
	svcArr, ok := svcSection["Service"].([]any)
	if !ok || len(svcArr) == 0 {
		t.Fatal("Service section should contain at least one service entry")
	}

	var foundLiability bool
	for _, s := range svcArr {
		svc, isMap := s.(map[string]any)
		if !isMap {
			continue
		}
		if svc["ServiceName"] != "service_addonliability" {
			continue
		}
		foundLiability = true
		liability, ok := svc["AddOnLiability"].(map[string]any)
		if !ok {
			t.Fatal("service_addonliability should have structured 'AddOnLiability' field")
		}
		amount, _ := liability["Amount"].(float64)
		if amount != 5000.00 {
			t.Errorf("AddOnLiability.Amount = %v, want 5000.00", amount)
		}
		currency, _ := liability["Currency"].(string)
		if currency != "PLN" {
			t.Errorf("AddOnLiability.Currency = %q, want PLN", currency)
		}
	}
	if !foundLiability {
		t.Error("Service array should contain service_addonliability entry when InsuredValue > 0")
	}
}

// --- Shipper / ContactID Handling ---

func TestGLS_CreateShipment_ShipperAddressNotMappedToContactID(t *testing.T) {
	// GLS uses ContactID (a pre-registered shipper reference), NOT a full
	// address like DHL/DPD. When a Shipper address is provided in the request
	// but no ContactID is configured, the adapter must NOT attempt to map
	// address fields into the GLS Shipper.ContactID field.
	// The shipment should still be created (using the default GLS account shipper).
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(glsOKResponse))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

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

	// The GLS API request should NOT contain a Shipper field with address data
	// mapped into ContactID (that would be incorrect — ContactID is a GLS concept).
	shipper, hasShipper := receivedBody["Shipper"]
	if hasShipper && shipper != nil {
		shipperMap, isMap := shipper.(map[string]any)
		if isMap {
			contactID, _ := shipperMap["ContactID"].(string)
			// ContactID must never contain address data like name, street, city
			if contactID == "Firma Nadawcza Sp. z o.o." ||
				contactID == "Przemyslowa 5" ||
				contactID == "Warszawa" {
				t.Errorf("Shipper.ContactID should not contain address data, got: %q", contactID)
			}
		}
	}
}

// --- Reference Field Mapping ---

func TestGLS_CreateShipment_ReferenceMapsToReferencesArray(t *testing.T) {
	// The Reference field from CarrierShipmentRequest must be sent in the
	// GLS References array so it appears on the label and in GLS systems.
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(glsOKResponse))
	}))
	defer srv.Close()

	provider := newTestGLSProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		Reference: "ORD-2024-00123",
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

	refs, ok := receivedBody["References"].([]any)
	if !ok || len(refs) == 0 {
		t.Fatal("request should contain 'References' array when Reference is provided")
	}

	found := false
	for _, ref := range refs {
		if ref == "ORD-2024-00123" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("References array should contain %q, got %v", "ORD-2024-00123", refs)
	}
}
