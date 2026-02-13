package carriers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	inpostsdk "github.com/openoms-org/openoms/packages/inpost-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// floatClose checks if two float64 values are within epsilon of each other.
func floatClose(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// newTestProvider creates an InPostProvider backed by the given httptest server
// URL. Both the ShipX and Points APIs are routed to the same mock server.
func newTestProvider(t *testing.T, serverURL string) *InPostProvider {
	t.Helper()
	client := inpostsdk.NewClient(
		"test-token",
		"test-org-123",
		inpostsdk.WithBaseURL(serverURL),
		inpostsdk.WithPointsBaseURL(serverURL),
	)
	return &InPostProvider{
		client: client,
		logger: slog.Default().With("provider", "inpost-test"),
	}
}

// ---------------------------------------------------------------------------
// Test 1: CreateShipment — Locker delivery
// ---------------------------------------------------------------------------

func TestInPostCreateShipment_Locker(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string
	var receivedMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedMethod = r.Method

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 12345,
			"tracking_number": "INPOST123456",
			"status": "confirmed",
			"service": "inpost_locker_standard",
			"receiver": {
				"name": "Jan Kowalski",
				"phone": "500600700",
				"email": "jan@example.com"
			},
			"parcels": [{
				"id": 99,
				"template": "small",
				"dimensions": {"height": 80, "width": 380, "length": 640},
				"weight": {"amount": 2.5, "unit": "kg"}
			}],
			"custom_attributes": {
				"target_point": "KRA01M"
			},
			"created_at": "2025-01-15T10:30:00Z",
			"updated_at": "2025-01-15T10:30:00Z"
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		OrderID:     "ORD-001",
		TargetPoint: "KRA01M",
		Reference:   "ref-locker-001",
		Receiver: integration.CarrierReceiver{
			Name:  "Jan Kowalski",
			Phone: "500600700",
			Email: "jan@example.com",
		},
		Parcel: integration.CarrierParcel{
			SizeCode: "small",
			WeightKg: 2.5,
		},
	}

	resp, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() returned error: %v", err)
	}

	// Verify the request was sent to the correct endpoint.
	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if !strings.Contains(receivedPath, "/v1/organizations/test-org-123/shipments") {
		t.Errorf("expected path to contain /v1/organizations/test-org-123/shipments, got %s", receivedPath)
	}

	// Verify the request body has the locker service type and target point.
	if svc, ok := receivedBody["service"].(string); !ok || svc != "inpost_locker_standard" {
		t.Errorf("expected service inpost_locker_standard, got %v", receivedBody["service"])
	}
	if ca, ok := receivedBody["custom_attributes"].(map[string]any); ok {
		if tp, ok := ca["target_point"].(string); !ok || tp != "KRA01M" {
			t.Errorf("expected target_point KRA01M, got %v", ca["target_point"])
		}
	} else {
		t.Error("expected custom_attributes with target_point in request body")
	}

	// Verify the response mapping.
	if resp.ExternalID != "12345" {
		t.Errorf("expected ExternalID 12345, got %s", resp.ExternalID)
	}
	if resp.TrackingNumber != "INPOST123456" {
		t.Errorf("expected TrackingNumber INPOST123456, got %s", resp.TrackingNumber)
	}
	if resp.Status != "confirmed" {
		t.Errorf("expected Status confirmed, got %s", resp.Status)
	}
}

// ---------------------------------------------------------------------------
// Test 2: CreateShipment — Courier delivery (address-based)
// ---------------------------------------------------------------------------

func TestInPostCreateShipment_Courier(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 67890,
			"tracking_number": "COURIER789012",
			"status": "confirmed",
			"service": "inpost_courier_standard",
			"receiver": {
				"name": "Anna Nowak",
				"phone": "600700800",
				"email": "anna@example.com",
				"address": {
					"street": "Marszalkowska 10",
					"city": "Warszawa",
					"post_code": "00-001",
					"country_code": "PL"
				}
			},
			"parcels": [{
				"id": 100,
				"template": "",
				"dimensions": {"height": 200, "width": 300, "length": 400},
				"weight": {"amount": 5.0, "unit": "kg"}
			}],
			"created_at": "2025-01-15T11:00:00Z",
			"updated_at": "2025-01-15T11:00:00Z"
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		OrderID:   "ORD-002",
		Reference: "ref-courier-002",
		Receiver: integration.CarrierReceiver{
			Name:       "Anna Nowak",
			Phone:      "600700800",
			Email:      "anna@example.com",
			Street:     "Marszalkowska 10",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		Parcel: integration.CarrierParcel{
			WeightKg: 5.0,
			WidthCm:  30,
			HeightCm: 20,
			DepthCm:  40,
		},
	}

	resp, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() returned error: %v", err)
	}

	// Verify the request body has courier service type and address, no custom_attributes.
	if svc, ok := receivedBody["service"].(string); !ok || svc != "inpost_courier_standard" {
		t.Errorf("expected service inpost_courier_standard, got %v", receivedBody["service"])
	}
	if receivedBody["custom_attributes"] != nil {
		t.Error("expected no custom_attributes for courier delivery")
	}

	// Verify receiver address was sent.
	if rcv, ok := receivedBody["receiver"].(map[string]any); ok {
		if addr, ok := rcv["address"].(map[string]any); ok {
			if addr["street"] != "Marszalkowska 10" {
				t.Errorf("expected street Marszalkowska 10, got %v", addr["street"])
			}
			if addr["city"] != "Warszawa" {
				t.Errorf("expected city Warszawa, got %v", addr["city"])
			}
			if addr["post_code"] != "00-001" {
				t.Errorf("expected post_code 00-001, got %v", addr["post_code"])
			}
			if addr["country_code"] != "PL" {
				t.Errorf("expected country_code PL, got %v", addr["country_code"])
			}
		} else {
			t.Error("expected receiver.address in request body")
		}
	} else {
		t.Error("expected receiver in request body")
	}

	// Verify dimensions were converted from cm to mm.
	if parcels, ok := receivedBody["parcels"].([]any); ok && len(parcels) > 0 {
		parcel := parcels[0].(map[string]any)
		if dims, ok := parcel["dimensions"].(map[string]any); ok {
			// 30 cm width -> 300 mm
			if dims["width"] != 300.0 {
				t.Errorf("expected width 300 (mm), got %v", dims["width"])
			}
			// 20 cm height -> 200 mm
			if dims["height"] != 200.0 {
				t.Errorf("expected height 200 (mm), got %v", dims["height"])
			}
			// 40 cm depth -> 400 mm (mapped to length)
			if dims["length"] != 400.0 {
				t.Errorf("expected length 400 (mm), got %v", dims["length"])
			}
		} else {
			t.Error("expected parcel dimensions in request body")
		}
	} else {
		t.Error("expected parcels in request body")
	}

	// Verify response mapping.
	if resp.ExternalID != "67890" {
		t.Errorf("expected ExternalID 67890, got %s", resp.ExternalID)
	}
	if resp.TrackingNumber != "COURIER789012" {
		t.Errorf("expected TrackingNumber COURIER789012, got %s", resp.TrackingNumber)
	}
	if resp.Status != "confirmed" {
		t.Errorf("expected Status confirmed, got %s", resp.Status)
	}
}

// ---------------------------------------------------------------------------
// Test 3: GetLabel — PDF format
// ---------------------------------------------------------------------------

func TestInPostGetLabel_PDF(t *testing.T) {
	fakePDF := []byte("%PDF-1.4 fake-label-content-for-testing")
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write(fakePDF)
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	data, err := provider.GetLabel(context.Background(), "12345", "pdf")
	if err != nil {
		t.Fatalf("GetLabel() returned error: %v", err)
	}

	// Verify the correct endpoint and format were requested.
	if !strings.Contains(receivedPath, "/v1/shipments/12345/label") {
		t.Errorf("expected path containing /v1/shipments/12345/label, got %s", receivedPath)
	}
	if !strings.Contains(receivedPath, "format=Pdf") {
		t.Errorf("expected query param format=Pdf, got %s", receivedPath)
	}

	// Verify we got data back.
	if len(data) == 0 {
		t.Fatal("expected non-empty label data")
	}
	if string(data) != string(fakePDF) {
		t.Errorf("label data mismatch: got %q, want %q", string(data), string(fakePDF))
	}
}

// ---------------------------------------------------------------------------
// Test 4: GetLabel — ZPL format
// ---------------------------------------------------------------------------

func TestInPostGetLabel_ZPL(t *testing.T) {
	fakeZPL := []byte("^XA^FO50,50^AND,36,20^FDInPost Label^FS^XZ")
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/x-zpl")
		w.WriteHeader(http.StatusOK)
		w.Write(fakeZPL)
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	data, err := provider.GetLabel(context.Background(), "12345", "zpl")
	if err != nil {
		t.Fatalf("GetLabel() returned error: %v", err)
	}

	// Verify ZPL format was requested.
	if !strings.Contains(receivedPath, "format=Zpl") {
		t.Errorf("expected query param format=Zpl, got %s", receivedPath)
	}

	// Verify we got data back.
	if len(data) == 0 {
		t.Fatal("expected non-empty label data")
	}
	if string(data) != string(fakeZPL) {
		t.Errorf("label data mismatch: got %q, want %q", string(data), string(fakeZPL))
	}
}

// ---------------------------------------------------------------------------
// Test 5: GetTracking
// ---------------------------------------------------------------------------

func TestInPostGetTracking(t *testing.T) {
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tracking_number": "INPOST123456",
			"service": "inpost_locker_standard",
			"tracking_details": [
				{
					"status": "confirmed",
					"origin_status": "Przesyłka potwierdzona",
					"datetime": "2025-01-15T10:30:00Z",
					"agency": ""
				},
				{
					"status": "dispatched_by_sender",
					"origin_status": "Nadana przez nadawcę",
					"datetime": "2025-01-15T14:00:00Z",
					"agency": "Sorting Center Krakow"
				},
				{
					"status": "delivered",
					"origin_status": "Dostarczona",
					"datetime": "2025-01-16T09:15:00Z",
					"agency": "KRA01M"
				}
			]
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	events, err := provider.GetTracking(context.Background(), "INPOST123456")
	if err != nil {
		t.Fatalf("GetTracking() returned error: %v", err)
	}

	// Verify path.
	if receivedPath != "/v1/tracking/INPOST123456" {
		t.Errorf("expected path /v1/tracking/INPOST123456, got %s", receivedPath)
	}

	// Verify we got 3 events.
	if len(events) != 3 {
		t.Fatalf("expected 3 tracking events, got %d", len(events))
	}

	// Verify first event.
	if events[0].Status != "confirmed" {
		t.Errorf("event[0].Status = %q, want %q", events[0].Status, "confirmed")
	}
	if events[0].Details != "Przesyłka potwierdzona" {
		t.Errorf("event[0].Details = %q, want %q", events[0].Details, "Przesyłka potwierdzona")
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	if !events[0].Timestamp.Equal(expectedTime) {
		t.Errorf("event[0].Timestamp = %v, want %v", events[0].Timestamp, expectedTime)
	}

	// Verify second event has location.
	if events[1].Status != "dispatched_by_sender" {
		t.Errorf("event[1].Status = %q, want %q", events[1].Status, "dispatched_by_sender")
	}
	if events[1].Location != "Sorting Center Krakow" {
		t.Errorf("event[1].Location = %q, want %q", events[1].Location, "Sorting Center Krakow")
	}

	// Verify third (delivered) event.
	if events[2].Status != "delivered" {
		t.Errorf("event[2].Status = %q, want %q", events[2].Status, "delivered")
	}
	if events[2].Location != "KRA01M" {
		t.Errorf("event[2].Location = %q, want %q", events[2].Location, "KRA01M")
	}
	deliveredTime, _ := time.Parse(time.RFC3339, "2025-01-16T09:15:00Z")
	if !events[2].Timestamp.Equal(deliveredTime) {
		t.Errorf("event[2].Timestamp = %v, want %v", events[2].Timestamp, deliveredTime)
	}
}

// ---------------------------------------------------------------------------
// Test 6: SearchPickupPoints
// ---------------------------------------------------------------------------

func TestInPostSearchPickupPoints(t *testing.T) {
	var receivedPath string
	var receivedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"items": [
				{
					"name": "KRA01M",
					"type": ["parcel_locker"],
					"address": {
						"line1": "Mogilska 1",
						"line2": "31-516 Kraków"
					},
					"address_details": {
						"city": "Kraków",
						"province": "małopolskie",
						"post_code": "31-516",
						"street": "Mogilska",
						"building_number": "1"
					},
					"location": {
						"latitude": 50.0647,
						"longitude": 19.9450
					},
					"location_description": "Przy wejściu do galerii",
					"opening_hours": "24/7",
					"status": "Operating"
				},
				{
					"name": "KRA02N",
					"type": ["parcel_locker"],
					"address": {
						"line1": "Pawia 5",
						"line2": "31-154 Kraków"
					},
					"address_details": {
						"city": "Kraków",
						"province": "małopolskie",
						"post_code": "31-154",
						"street": "Pawia",
						"building_number": "5"
					},
					"location": {
						"latitude": 50.0688,
						"longitude": 19.9453
					},
					"location_description": "Galeria Krakowska",
					"opening_hours": "06:00-23:00",
					"status": "Operating"
				}
			],
			"count": 2,
			"page": 1,
			"per_page": 10,
			"total_pages": 1
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	points, err := provider.SearchPickupPoints(context.Background(), "Kraków")
	if err != nil {
		t.Fatalf("SearchPickupPoints() returned error: %v", err)
	}

	// Verify the points search endpoint was called.
	if receivedPath != "/v1/points" {
		t.Errorf("expected path /v1/points, got %s", receivedPath)
	}
	// "Kraków" is not a point code, so it should search by city.
	if !strings.Contains(receivedQuery, "city=") {
		t.Errorf("expected city= query param for city search, got %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "type=parcel_locker") {
		t.Errorf("expected type=parcel_locker query param, got %s", receivedQuery)
	}

	// Verify we got 2 points.
	if len(points) != 2 {
		t.Fatalf("expected 2 pickup points, got %d", len(points))
	}

	// Verify first point.
	p1 := points[0]
	if p1.ID != "KRA01M" {
		t.Errorf("point[0].ID = %q, want %q", p1.ID, "KRA01M")
	}
	if p1.Name != "KRA01M" {
		t.Errorf("point[0].Name = %q, want %q", p1.Name, "KRA01M")
	}
	if p1.City != "Kraków" {
		t.Errorf("point[0].City = %q, want %q", p1.City, "Kraków")
	}
	if p1.PostalCode != "31-516" {
		t.Errorf("point[0].PostalCode = %q, want %q", p1.PostalCode, "31-516")
	}
	if p1.Street != "Mogilska 1" {
		t.Errorf("point[0].Street = %q, want %q", p1.Street, "Mogilska 1")
	}
	if p1.Latitude != 50.0647 {
		t.Errorf("point[0].Latitude = %f, want 50.0647", p1.Latitude)
	}
	if p1.Longitude != 19.9450 {
		t.Errorf("point[0].Longitude = %f, want 19.9450", p1.Longitude)
	}
	if p1.Type != "parcel_locker" {
		t.Errorf("point[0].Type = %q, want %q", p1.Type, "parcel_locker")
	}

	// Verify second point.
	p2 := points[1]
	if p2.ID != "KRA02N" {
		t.Errorf("point[1].ID = %q, want %q", p2.ID, "KRA02N")
	}
	if p2.Latitude != 50.0688 {
		t.Errorf("point[1].Latitude = %f, want 50.0688", p2.Latitude)
	}
}

// ---------------------------------------------------------------------------
// Test 7: GetRates — various package sizes
// ---------------------------------------------------------------------------

func TestInPostGetRates(t *testing.T) {
	provider := newTestProvider(t, "http://unused") // GetRates doesn't make API calls

	t.Run("small parcel fits all locker sizes", func(t *testing.T) {
		rates, err := provider.GetRates(context.Background(), integration.RateRequest{
			FromCountry: "PL",
			ToCountry:   "PL",
			Weight:      2,
			Width:       20,
			Height:      5,
			Length:      30,
		})
		if err != nil {
			t.Fatalf("GetRates() error: %v", err)
		}

		// Small parcel: fits A, B, C + courier = 4 rates
		if len(rates) != 4 {
			t.Fatalf("expected 4 rates for small parcel, got %d", len(rates))
		}

		// Verify Paczkomat A rate.
		if rates[0].ServiceName != "Paczkomat A (mała)" {
			t.Errorf("rates[0].ServiceName = %q, want %q", rates[0].ServiceName, "Paczkomat A (mała)")
		}
		if rates[0].Price != 12.99 {
			t.Errorf("rates[0].Price = %f, want 12.99", rates[0].Price)
		}
		if rates[0].Currency != "PLN" {
			t.Errorf("rates[0].Currency = %q, want PLN", rates[0].Currency)
		}
		if !rates[0].PickupPoint {
			t.Error("rates[0].PickupPoint should be true")
		}

		// Verify Paczkomat B rate.
		if rates[1].ServiceName != "Paczkomat B (średnia)" {
			t.Errorf("rates[1].ServiceName = %q, want %q", rates[1].ServiceName, "Paczkomat B (średnia)")
		}
		if rates[1].Price != 13.99 {
			t.Errorf("rates[1].Price = %f, want 13.99", rates[1].Price)
		}

		// Verify Paczkomat C rate.
		if rates[2].ServiceName != "Paczkomat C (duża)" {
			t.Errorf("rates[2].ServiceName = %q, want %q", rates[2].ServiceName, "Paczkomat C (duża)")
		}
		if rates[2].Price != 15.49 {
			t.Errorf("rates[2].Price = %f, want 15.49", rates[2].Price)
		}

		// Verify Courier rate.
		if rates[3].ServiceName != "Kurier Standard" {
			t.Errorf("rates[3].ServiceName = %q, want %q", rates[3].ServiceName, "Kurier Standard")
		}
		if rates[3].Price != 16.99 {
			t.Errorf("rates[3].Price = %f, want 16.99", rates[3].Price)
		}
		if rates[3].PickupPoint {
			t.Error("courier rate PickupPoint should be false")
		}
		if rates[3].EstimatedDays != 1 {
			t.Errorf("courier EstimatedDays = %d, want 1", rates[3].EstimatedDays)
		}
	})

	t.Run("medium parcel fits B and C only", func(t *testing.T) {
		rates, err := provider.GetRates(context.Background(), integration.RateRequest{
			FromCountry: "PL",
			ToCountry:   "PL",
			Weight:      11,
			Width:       38,
			Height:      15,
			Length:      64,
		})
		if err != nil {
			t.Fatalf("GetRates() error: %v", err)
		}

		// A: max 8 kg, fits 38x64x8 cm -- weight=11 > 8, so A doesn't fit.
		// B: max 25 kg, fits 38x64x19 cm -- weight=11<=25, width=38<=38, height=15<=19, length=64<=64 -> fits
		// C: max 25 kg, fits 41x38x64 cm -- fits
		// Courier: weight<=25 -> fits, weight>10 -> price 19.99
		if len(rates) != 3 {
			t.Fatalf("expected 3 rates for medium parcel, got %d", len(rates))
		}

		if rates[0].ServiceName != "Paczkomat B (średnia)" {
			t.Errorf("rates[0].ServiceName = %q, want Paczkomat B", rates[0].ServiceName)
		}
		if rates[1].ServiceName != "Paczkomat C (duża)" {
			t.Errorf("rates[1].ServiceName = %q, want Paczkomat C", rates[1].ServiceName)
		}
		if rates[2].ServiceName != "Kurier Standard" {
			t.Errorf("rates[2].ServiceName = %q, want Kurier Standard", rates[2].ServiceName)
		}
		// Weight > 10 kg -> courier price 19.99.
		if rates[2].Price != 19.99 {
			t.Errorf("courier price = %f, want 19.99 (>10kg)", rates[2].Price)
		}
	})

	t.Run("oversized parcel gets only courier", func(t *testing.T) {
		rates, err := provider.GetRates(context.Background(), integration.RateRequest{
			FromCountry: "PL",
			ToCountry:   "PL",
			Weight:      20,
			Width:       50, // exceeds all locker width limits
			Height:      40,
			Length:      70,
		})
		if err != nil {
			t.Fatalf("GetRates() error: %v", err)
		}

		// No locker sizes fit, only courier (weight<=25).
		if len(rates) != 1 {
			t.Fatalf("expected 1 rate for oversized parcel, got %d", len(rates))
		}
		if rates[0].ServiceName != "Kurier Standard" {
			t.Errorf("expected Kurier Standard, got %q", rates[0].ServiceName)
		}
	})

	t.Run("very heavy parcel gets no rates", func(t *testing.T) {
		rates, err := provider.GetRates(context.Background(), integration.RateRequest{
			FromCountry: "PL",
			ToCountry:   "PL",
			Weight:      30, // exceeds 25 kg limit for all services
			Width:       50,
			Height:      50,
			Length:      70,
		})
		if err != nil {
			t.Fatalf("GetRates() error: %v", err)
		}

		if len(rates) != 0 {
			t.Errorf("expected 0 rates for very heavy parcel, got %d", len(rates))
		}
	})

	t.Run("COD adds surcharge", func(t *testing.T) {
		rates, err := provider.GetRates(context.Background(), integration.RateRequest{
			FromCountry: "PL",
			ToCountry:   "PL",
			Weight:      2,
			Width:       20,
			Height:      5,
			Length:      30,
			COD:         100,
		})
		if err != nil {
			t.Fatalf("GetRates() error: %v", err)
		}

		if len(rates) != 4 {
			t.Fatalf("expected 4 rates with COD, got %d", len(rates))
		}

		// Paczkomat A with COD: 12.99 + 3.50 = 16.49.
		if !floatClose(rates[0].Price, 16.49, 0.001) {
			t.Errorf("Paczkomat A with COD: price = %f, want ~16.49", rates[0].Price)
		}

		// Courier with COD: 16.99 + 4.00 = 20.99.
		if !floatClose(rates[3].Price, 20.99, 0.001) {
			t.Errorf("Courier with COD: price = %f, want ~20.99", rates[3].Price)
		}
	})

	t.Run("international shipment gets no rates", func(t *testing.T) {
		rates, err := provider.GetRates(context.Background(), integration.RateRequest{
			FromCountry: "PL",
			ToCountry:   "DE",
			Weight:      2,
			Width:       20,
			Height:      5,
			Length:      30,
		})
		if err != nil {
			t.Fatalf("GetRates() error: %v", err)
		}

		if len(rates) != 0 {
			t.Errorf("expected 0 rates for international shipment, got %d", len(rates))
		}
	})
}

// ---------------------------------------------------------------------------
// Test 8: MapStatus — all InPost status to OpenOMS status mappings
// ---------------------------------------------------------------------------

func TestInPostMapStatus(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	tests := []struct {
		inpostStatus string
		wantOMS      string
		wantOK       bool
	}{
		// created group
		{"created", "created", true},
		{"offers_prepared", "created", true},
		{"offer_selected", "created", true},
		// label_ready
		{"confirmed", "label_ready", true},
		// picked_up
		{"dispatched_by_sender", "picked_up", true},
		{"collected_from_sender", "picked_up", true},
		// in_transit
		{"taken_by_courier", "in_transit", true},
		{"adopted_at_source_branch", "in_transit", true},
		{"sent_from_source_branch", "in_transit", true},
		{"adopted_at_sorting_center", "in_transit", true},
		{"sent_from_sorting_center", "in_transit", true},
		{"adopted_at_target_branch", "in_transit", true},
		// out_for_delivery
		{"out_for_delivery", "out_for_delivery", true},
		{"ready_to_pickup", "out_for_delivery", true},
		{"avizo", "out_for_delivery", true},
		// delivered
		{"delivered", "delivered", true},
		{"picked_up", "delivered", true},
		// returned
		{"returned_to_sender", "returned", true},
		// failed
		{"missing", "failed", true},
		{"claim_rejected", "failed", true},
		// unknown status
		{"completely_unknown_status", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.inpostStatus, func(t *testing.T) {
			gotOMS, gotOK := provider.MapStatus(tc.inpostStatus)
			if gotOK != tc.wantOK {
				t.Errorf("MapStatus(%q) ok = %v, want %v", tc.inpostStatus, gotOK, tc.wantOK)
			}
			if gotOMS != tc.wantOMS {
				t.Errorf("MapStatus(%q) = %q, want %q", tc.inpostStatus, gotOMS, tc.wantOMS)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test 9: CreateShipment — API validation error
// ---------------------------------------------------------------------------

func TestInPostCreateShipment_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"message": "Validation failed",
			"details": {
				"receiver.phone": "is invalid",
				"parcels[0].weight.amount": "must be greater than 0"
			}
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		OrderID:     "ORD-BAD",
		TargetPoint: "KRA01M",
		Receiver: integration.CarrierReceiver{
			Name:  "Bad Request",
			Phone: "invalid",
		},
		Parcel: integration.CarrierParcel{
			SizeCode: "small",
			WeightKg: 0, // invalid
		},
	}

	_, err := provider.CreateShipment(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from API validation failure")
	}

	// Verify the error message contains useful information.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "inpost") {
		t.Errorf("expected error to mention 'inpost', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Validation failed") {
		t.Errorf("expected error to contain 'Validation failed', got: %s", errMsg)
	}
}

// ---------------------------------------------------------------------------
// Test 10: Credential parsing and base URL selection
// ---------------------------------------------------------------------------

func TestInPostCredentialParsing(t *testing.T) {
	t.Run("production mode", func(t *testing.T) {
		creds := `{"api_token":"prod-token-abc","organization_id":"org-prod-456","sandbox":false}`
		settings := `{}`

		provider, err := NewInPostProvider(json.RawMessage(creds), json.RawMessage(settings))
		if err != nil {
			t.Fatalf("NewInPostProvider() error: %v", err)
		}

		if provider.ProviderName() != "inpost" {
			t.Errorf("ProviderName() = %q, want %q", provider.ProviderName(), "inpost")
		}
		if provider.client == nil {
			t.Fatal("expected client to be initialized")
		}
		if !provider.SupportsPickupPoints() {
			t.Error("expected SupportsPickupPoints() to return true")
		}
	})

	t.Run("sandbox mode", func(t *testing.T) {
		creds := `{"api_token":"sandbox-token-xyz","organization_id":"org-sandbox-789","sandbox":true}`
		settings := `{}`

		provider, err := NewInPostProvider(json.RawMessage(creds), json.RawMessage(settings))
		if err != nil {
			t.Fatalf("NewInPostProvider() error: %v", err)
		}

		if provider.client == nil {
			t.Fatal("expected client to be initialized")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		creds := `{invalid json}`
		settings := `{}`

		_, err := NewInPostProvider(json.RawMessage(creds), json.RawMessage(settings))
		if err == nil {
			t.Fatal("expected error for invalid JSON credentials")
		}
		if !strings.Contains(err.Error(), "parse credentials") {
			t.Errorf("expected 'parse credentials' in error, got: %s", err.Error())
		}
	})

	t.Run("missing fields uses defaults", func(t *testing.T) {
		// Empty but valid JSON — token and org will be empty strings.
		creds := `{}`
		settings := `{}`

		provider, err := NewInPostProvider(json.RawMessage(creds), json.RawMessage(settings))
		if err != nil {
			t.Fatalf("NewInPostProvider() error: %v", err)
		}
		if provider.client == nil {
			t.Fatal("expected client to be initialized even with empty credentials")
		}
	})
}

// ---------------------------------------------------------------------------
// Test: GetLabel with invalid external ID
// ---------------------------------------------------------------------------

func TestInPostGetLabel_InvalidID(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	_, err := provider.GetLabel(context.Background(), "not-a-number", "pdf")
	if err == nil {
		t.Fatal("expected error for non-numeric external ID")
	}
	if !strings.Contains(err.Error(), "invalid shipment ID") {
		t.Errorf("expected 'invalid shipment ID' in error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Test: GetLabel EPL format (verifying format mapping)
// ---------------------------------------------------------------------------

func TestInPostGetLabel_EPL(t *testing.T) {
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("N\nq812\nQ1218,24\nZT\nP1\n"))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	data, err := provider.GetLabel(context.Background(), "99999", "epl")
	if err != nil {
		t.Fatalf("GetLabel(epl) error: %v", err)
	}
	if !strings.Contains(receivedPath, "format=Epl") {
		t.Errorf("expected format=Epl in path, got %s", receivedPath)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty EPL label data")
	}
}

// ---------------------------------------------------------------------------
// Test: GetLabel unknown format defaults to PDF
// ---------------------------------------------------------------------------

func TestInPostGetLabel_UnknownFormatDefaultsPDF(t *testing.T) {
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("%PDF-default"))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	_, err := provider.GetLabel(context.Background(), "11111", "unknown_format")
	if err != nil {
		t.Fatalf("GetLabel(unknown) error: %v", err)
	}
	if !strings.Contains(receivedPath, "format=Pdf") {
		t.Errorf("expected unknown format to default to Pdf, got %s", receivedPath)
	}
}

// ---------------------------------------------------------------------------
// Test: CancelShipment returns not-implemented error
// ---------------------------------------------------------------------------

func TestInPostCancelShipment_NotImplemented(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	err := provider.CancelShipment(context.Background(), "12345")
	if err == nil {
		t.Fatal("expected error from CancelShipment")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("expected 'not implemented' error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Test: CreateShipment with explicit service type override
// ---------------------------------------------------------------------------

func TestInPostCreateShipment_ExplicitServiceType(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 55555,
			"tracking_number": "CUSTOM555",
			"status": "confirmed",
			"service": "inpost_courier_standard",
			"receiver": {"name": "Test", "phone": "111", "email": ""},
			"parcels": [],
			"created_at": "2025-01-20T12:00:00Z",
			"updated_at": "2025-01-20T12:00:00Z"
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	// When TargetPoint is empty and ServiceType is set, it should use the provided service type.
	req := integration.CarrierShipmentRequest{
		OrderID:     "ORD-SVC",
		ServiceType: "inpost_courier_standard",
		Receiver: integration.CarrierReceiver{
			Name:       "Test",
			Phone:      "111",
			Street:     "Testowa 1",
			City:       "Testowo",
			PostalCode: "00-000",
			Country:    "PL",
		},
		Parcel: integration.CarrierParcel{
			WeightKg: 1,
		},
	}

	resp, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if svc, ok := receivedBody["service"].(string); !ok || svc != "inpost_courier_standard" {
		t.Errorf("expected service inpost_courier_standard, got %v", receivedBody["service"])
	}

	if resp.ExternalID != "55555" {
		t.Errorf("ExternalID = %q, want 55555", resp.ExternalID)
	}
}

// ---------------------------------------------------------------------------
// Test: SearchPickupPoints by exact point code
// ---------------------------------------------------------------------------

func TestInPostSearchPickupPoints_ByCode(t *testing.T) {
	var receivedQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"items": [{
				"name": "KRA01M",
				"type": ["parcel_locker"],
				"address": {"line1": "Test 1", "line2": "31-000 Kraków"},
				"location": {"latitude": 50.06, "longitude": 19.94},
				"status": "Operating"
			}],
			"count": 1,
			"page": 1,
			"per_page": 10,
			"total_pages": 1
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	points, err := provider.SearchPickupPoints(context.Background(), "KRA01M")
	if err != nil {
		t.Fatalf("SearchPickupPoints() error: %v", err)
	}

	// Point code pattern should trigger name= search instead of city=.
	if !strings.Contains(receivedQuery, "name=KRA01M") {
		t.Errorf("expected name=KRA01M for point code search, got %s", receivedQuery)
	}

	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Name != "KRA01M" {
		t.Errorf("point name = %q, want KRA01M", points[0].Name)
	}
}

// ---------------------------------------------------------------------------
// Test: SearchPickupPoints with no address_details
// ---------------------------------------------------------------------------

func TestInPostSearchPickupPoints_NoAddressDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"items": [{
				"name": "WAW100A",
				"type": ["parcel_locker"],
				"address": {"line1": "Somewhere", "line2": "Warszawa"},
				"location": {"latitude": 52.23, "longitude": 21.01},
				"status": "Operating"
			}],
			"count": 1,
			"page": 1,
			"per_page": 10,
			"total_pages": 1
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	points, err := provider.SearchPickupPoints(context.Background(), "Warszawa")
	if err != nil {
		t.Fatalf("SearchPickupPoints() error: %v", err)
	}

	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}

	// Without address_details, street/city/postal_code should be empty.
	if points[0].Street != "" {
		t.Errorf("expected empty street without address_details, got %q", points[0].Street)
	}
	if points[0].City != "" {
		t.Errorf("expected empty city without address_details, got %q", points[0].City)
	}
	if points[0].PostalCode != "" {
		t.Errorf("expected empty postal_code without address_details, got %q", points[0].PostalCode)
	}
	// Location should still be present.
	if points[0].Latitude != 52.23 {
		t.Errorf("point latitude = %f, want 52.23", points[0].Latitude)
	}
}

// ---------------------------------------------------------------------------
// Test: CreateShipment verifies authorization header
// ---------------------------------------------------------------------------

func TestInPostCreateShipment_AuthorizationHeader(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 1,
			"tracking_number": "T1",
			"status": "created",
			"service": "inpost_locker_standard",
			"receiver": {"name": "X", "phone": "1", "email": ""},
			"parcels": [],
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-01-01T00:00:00Z"
		}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	req := integration.CarrierShipmentRequest{
		TargetPoint: "KRA01M",
		Receiver:    integration.CarrierReceiver{Name: "X", Phone: "1"},
		Parcel:      integration.CarrierParcel{WeightKg: 1},
	}
	_, err := provider.CreateShipment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateShipment() error: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
}

// ---------------------------------------------------------------------------
// Test: GetTracking API error
// ---------------------------------------------------------------------------

func TestInPostGetTracking_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Tracking not found"}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	_, err := provider.GetTracking(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for non-existent tracking number")
	}
	if !strings.Contains(err.Error(), "Tracking not found") {
		t.Errorf("expected 'Tracking not found' in error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Test: SearchPickupPoints API error
// ---------------------------------------------------------------------------

func TestInPostSearchPickupPoints_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)

	_, err := provider.SearchPickupPoints(context.Background(), "Kraków")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

// ---------------------------------------------------------------------------
// Test: Full shipment lifecycle (create -> label -> track)
// ---------------------------------------------------------------------------

func TestInPostFullShipmentLifecycle(t *testing.T) {
	requestCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Step 1: Create shipment.
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/shipments"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": 42000,
				"tracking_number": "LIFECYCLE42",
				"status": "confirmed",
				"service": "inpost_locker_standard",
				"receiver": {"name": "Lifecycle Test", "phone": "111222333", "email": "life@test.com"},
				"parcels": [{"id": 1, "template": "small", "dimensions": {"height": 80, "width": 380, "length": 640}, "weight": {"amount": 1.5, "unit": "kg"}}],
				"custom_attributes": {"target_point": "WRO01A"},
				"created_at": "2025-02-01T08:00:00Z",
				"updated_at": "2025-02-01T08:00:00Z"
			}`))

		// Step 2: Get label.
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/label"):
			w.Header().Set("Content-Type", "application/pdf")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("%PDF-lifecycle-label"))

		// Step 3: Get tracking.
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tracking/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"tracking_number": "LIFECYCLE42",
				"service": "inpost_locker_standard",
				"tracking_details": [
					{"status": "confirmed", "origin_status": "Confirmed", "datetime": "2025-02-01T08:00:00Z", "agency": ""},
					{"status": "dispatched_by_sender", "origin_status": "Dispatched", "datetime": "2025-02-01T12:00:00Z", "agency": "Hub Warsaw"},
					{"status": "ready_to_pickup", "origin_status": "Ready", "datetime": "2025-02-02T06:00:00Z", "agency": "WRO01A"}
				]
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"message":"not found: %s %s"}`, r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	provider := newTestProvider(t, srv.URL)
	ctx := context.Background()

	// Step 1: Create shipment.
	shipResp, err := provider.CreateShipment(ctx, integration.CarrierShipmentRequest{
		OrderID:     "ORD-LIFECYCLE",
		TargetPoint: "WRO01A",
		Reference:   "lifecycle-test",
		Receiver: integration.CarrierReceiver{
			Name:  "Lifecycle Test",
			Phone: "111222333",
			Email: "life@test.com",
		},
		Parcel: integration.CarrierParcel{
			SizeCode: "small",
			WeightKg: 1.5,
		},
	})
	if err != nil {
		t.Fatalf("Step 1 (CreateShipment) failed: %v", err)
	}
	if shipResp.ExternalID != "42000" {
		t.Errorf("Step 1: ExternalID = %q, want 42000", shipResp.ExternalID)
	}
	if shipResp.TrackingNumber != "LIFECYCLE42" {
		t.Errorf("Step 1: TrackingNumber = %q, want LIFECYCLE42", shipResp.TrackingNumber)
	}

	// Step 2: Get label using the external ID from Step 1.
	label, err := provider.GetLabel(ctx, shipResp.ExternalID, "pdf")
	if err != nil {
		t.Fatalf("Step 2 (GetLabel) failed: %v", err)
	}
	if len(label) == 0 {
		t.Error("Step 2: expected non-empty label data")
	}

	// Step 3: Get tracking using the tracking number from Step 1.
	events, err := provider.GetTracking(ctx, shipResp.TrackingNumber)
	if err != nil {
		t.Fatalf("Step 3 (GetTracking) failed: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("Step 3: expected 3 tracking events, got %d", len(events))
	}

	// Verify we can map the final status.
	lastStatus := events[len(events)-1].Status
	omsStatus, ok := provider.MapStatus(lastStatus)
	if !ok {
		t.Errorf("Step 3: MapStatus(%q) returned ok=false", lastStatus)
	}
	if omsStatus != "out_for_delivery" {
		t.Errorf("Step 3: MapStatus(%q) = %q, want out_for_delivery", lastStatus, omsStatus)
	}

	// Verify all 3 API calls were made.
	if requestCount != 3 {
		t.Errorf("expected 3 API requests in lifecycle, got %d", requestCount)
	}
}
