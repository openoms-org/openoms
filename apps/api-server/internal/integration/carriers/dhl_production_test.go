package carriers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// =============================================================================
// DHL Provider — Production Readiness Specification Tests
//
// These tests define REQUIRED behavior for DHL production use:
//   - Service type mapping: dhl_parcel → AH, dhl_courier → DR
//   - Street/house number splitting for Polish addresses (DHL24 requires
//     separate street and houseNo fields)
//   - Shipper address populated in SOAP (DHL24 rejects without shipper)
// =============================================================================

// --- Service Type Mapping ---

func TestDHL_CreateShipment_ServiceTypeMapping(t *testing.T) {
	// Frontend sends dhl_parcel / dhl_courier as service types.
	// DHL24 SOAP API requires internal codes: AH (parcel domestic), DR (courier domestic).
	// The provider must translate frontend values to DHL24 codes.
	tests := []struct {
		name         string
		inputType    string
		wantSOAPType string
	}{
		{"dhl_parcel maps to AH", "dhl_parcel", "AH"},
		{"dhl_courier maps to DR", "dhl_courier", "DR"},
		{"AH passes through unchanged", "AH", "AH"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
				ServiceType: tc.inputType,
				Shipper:     testShipperAddr,
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

			wantTag := fmt.Sprintf("<serviceType>%s</serviceType>", tc.wantSOAPType)
			if !strings.Contains(requestXML, wantTag) {
				t.Errorf("SOAP XML should contain %s for input %q", wantTag, tc.inputType)
			}
		})
	}
}

func TestDHL_CreateShipment_UnknownServiceTypeReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("SOAP request should not be sent for unknown service type")
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		ServiceType: "invalid_type",
		Shipper:     testShipperAddr,
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

// --- Street / House Number Splitting ---

func TestDHL_CreateShipment_ReceiverStreetSplit(t *testing.T) {
	// DHL24 SOAP API requires separate <street> and <houseNo> fields.
	// Polish addresses come as a single string (e.g. "ul. Krakowska 15A").
	// The provider must split them into street name and house number.
	tests := []struct {
		name        string
		input       string
		wantStreet  string
		wantHouseNo string
	}{
		{"simple number", "Marszalkowska 10", "Marszalkowska", "10"},
		{"with ul. prefix", "ul. Krakowska 15A", "ul. Krakowska", "15A"},
		{"apartment number", "Aleje Jerozolimskie 100/2", "Aleje Jerozolimskie", "100/2"},
		{"multi-word street", "Plac Zamkowy 4", "Plac Zamkowy", "4"},
		{"no house number", "Krakowska", "Krakowska", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
				Shipper: testShipperAddr,
				Receiver: integration.CarrierReceiver{
					Name: "Test Receiver", Phone: "500100200",
					Street: tc.input, City: "Warszawa",
					PostalCode: "00-001", Country: "PL",
				},
				Parcel: integration.CarrierParcel{WeightKg: 1.0},
			}

			_, err := provider.CreateShipment(context.Background(), req)
			if err != nil {
				t.Fatalf("CreateShipment() error: %v", err)
			}

			// Isolate receiver section to avoid matching shipper fields
			receiverStart := strings.Index(requestXML, "<receiver>")
			receiverEnd := strings.Index(requestXML, "</receiver>")
			if receiverStart == -1 || receiverEnd == -1 {
				t.Fatal("SOAP XML should contain <receiver>...</receiver> section")
			}
			receiverSection := requestXML[receiverStart:receiverEnd]

			wantStreetTag := fmt.Sprintf("<street>%s</street>", tc.wantStreet)
			if !strings.Contains(receiverSection, wantStreetTag) {
				t.Errorf("receiver should have %s, got section:\n%s", wantStreetTag, receiverSection)
			}

			if tc.wantHouseNo != "" {
				wantHouseTag := fmt.Sprintf("<houseNo>%s</houseNo>", tc.wantHouseNo)
				if !strings.Contains(receiverSection, wantHouseTag) {
					t.Errorf("receiver should have %s, got section:\n%s", wantHouseTag, receiverSection)
				}
			}
		})
	}
}

// --- Shipper Address ---

func TestDHL_CreateShipment_SOAPMustContainShipperAddress(t *testing.T) {
	// DHL24 SOAP API requires shipper address in every createShipments request.
	// The provider returns an error when no shipper is provided, and maps all
	// shipper fields (name, street, city, postalCode, country, phone, email)
	// into the SOAP <shipper> section when provided.
	var requestXML string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestXML = string(body)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, dhlCreateSOAPResponse("SHP-001", "1234567890"))
	}))
	defer srv.Close()

	provider := newTestDHLProvider(t, srv.URL)

	// Verify: nil shipper returns a clear error (not a silent DHL API failure).
	reqNoShipper := integration.CarrierShipmentRequest{
		ServiceType: "AH",
		Receiver: integration.CarrierReceiver{
			Name: "Jan Kowalski", Phone: "500100200",
			Street: "Marszalkowska 1", City: "Warszawa",
			PostalCode: "00-001", Country: "PL",
		},
		Parcel: integration.CarrierParcel{WeightKg: 2.0},
	}
	_, err := provider.CreateShipment(context.Background(), reqNoShipper)
	if err == nil {
		t.Fatal("CreateShipment() should return error when Shipper is nil")
	}
	if !strings.Contains(err.Error(), "shipper address is required") {
		t.Errorf("error should mention 'shipper address is required', got: %v", err)
	}

	// Verify: provided shipper is correctly mapped into SOAP XML.
	req := integration.CarrierShipmentRequest{
		ServiceType: "AH",
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

	_, err = provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	shipperStart := strings.Index(requestXML, "<shipper>")
	shipperEnd := strings.Index(requestXML, "</shipper>")
	if shipperStart == -1 || shipperEnd == -1 {
		t.Fatal("SOAP XML should contain <shipper>...</shipper> section")
	}
	shipperSection := requestXML[shipperStart:shipperEnd]

	if !strings.Contains(shipperSection, "<name>Firma Nadawcza Sp. z o.o.</name>") {
		t.Errorf("shipper section should contain sender name, got:\n%s", shipperSection)
	}
	if !strings.Contains(shipperSection, "<city>Warszawa</city>") {
		t.Errorf("shipper section should contain sender city, got:\n%s", shipperSection)
	}
	if !strings.Contains(shipperSection, "<postalCode>00-950</postalCode>") {
		t.Errorf("shipper section should contain sender postalCode, got:\n%s", shipperSection)
	}
	if !strings.Contains(shipperSection, "<email>wysylki@firma.pl</email>") {
		t.Errorf("shipper section should contain sender email, got:\n%s", shipperSection)
	}
}
