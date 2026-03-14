package dhl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// =============================================================================
// DHL24 WebAPI2 — Production Readiness Specification Tests
//
// These tests define REQUIRED fixes before production use:
//   - Shipper struct must include Phone field (DHL24 requires shipper phone)
//   - No duplicate <service> + <serviceType> in SOAP XML (only serviceType)
//   - Tracking timestamp parse errors must not be silently ignored
// =============================================================================

// --- Shipper Phone ---

func TestSpec_Shipper_HasPhoneField(t *testing.T) {
	// DHL24 SOAP API requires shipper phone in the address block.
	// The Shipper model must have a Phone field so callers can set it.
	typ := reflect.TypeFor[Shipper]()
	_, found := typ.FieldByName("Phone")
	if !found {
		t.Error("Shipper struct must have a Phone field — DHL24 SOAP requires shipper phone in address")
	}
}

// --- Duplicate Service Field ---

func TestSpec_CreateShipment_NoDuplicateServiceField(t *testing.T) {
	// DHL24 uses <serviceType> for the service code (e.g. "AH", "DR").
	// The SOAP XML must NOT contain a separate <service> element with the
	// same value — this is a duplicate that confuses the API.
	var requestBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)

		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<createShipmentsResponse>
					<shipmentId>S</shipmentId>
					<trackingNumber>T</trackingNumber>
					<status>CREATED</status>
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
		Receiver:    Receiver{Name: "Test", Street: "Testowa", City: "Warszawa", PostalCode: "00-001", Country: "PL"},
		Piece:       Piece{Weight: 1.0},
		ServiceType: "AH",
	})

	// <service>AH</service> should NOT appear — only <serviceType>AH</serviceType>
	if strings.Contains(requestBody, "<service>") {
		t.Error("SOAP XML must not contain <service> element — only <serviceType> should be used for DHL24")
	}
}

// --- Timestamp Parse Error ---

func TestSpec_GetTracking_InvalidTimestamp_MustReturnError(t *testing.T) {
	// When DHL returns a tracking event with an unparseable timestamp,
	// the SDK must return an error — not silently produce a zero time.Time.
	// Silent failures hide data corruption from callers.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
		<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<getTrackAndTraceInfoResponse>
					<events>
						<event>
							<status>IN_TRANSIT</status>
							<location>Warszawa</location>
							<timestamp>not-a-valid-timestamp</timestamp>
							<details>Package in transit</details>
						</event>
					</events>
				</getTrackAndTraceInfoResponse>
			</soap:Body>
		</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "1234567890")
	if err == nil {
		t.Error("GetTracking must return error when event timestamp is invalid — silent parse failure produces zero time")
	}
}
