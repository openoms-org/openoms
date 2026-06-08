package carriers

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apaczka "github.com/openoms-org/openoms/packages/apaczka-go-sdk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
)

// apaczkaEnvelope is the standard Apaczka API response wrapper used in mocks.
type apaczkaEnvelope struct {
	Status   int             `json:"status"`
	Message  string          `json:"message"`
	Response json.RawMessage `json:"response"`
}

func apaczkaWriteJSON(w http.ResponseWriter, body any) {
	data, _ := json.Marshal(body)
	env := apaczkaEnvelope{Status: 200, Response: data}
	out, _ := json.Marshal(env)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

// newApaczkaMockServer creates an httptest.Server that routes Apaczka API calls
// based on URL path suffix and returns canned JSON responses.
func newApaczkaMockServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range handlers {
		mux.HandleFunc(path, h)
	}
	return httptest.NewServer(mux)
}

// newTestApaczkaProvider builds an ApaczkaProvider pointing at baseURL (mock server).
// It uses http.DefaultClient (no SSRF protection) so the client can reach 127.0.0.1.
func newTestApaczkaProvider(t *testing.T, baseURL string) *ApaczkaProvider {
	t.Helper()
	client := apaczka.NewClient(
		"test-app",
		"test-secret",
		apaczka.WithHTTPClient(http.DefaultClient),
		apaczka.WithBaseURL(baseURL),
	)
	return &ApaczkaProvider{
		client:   client,
		settings: apaczkaSettings{BaseURL: baseURL},
		logger:   slog.Default().With("provider", "apaczka-test"),
	}
}

// ---------------------------------------------------------------------------
// TestApaczkaRegistered — factory registration
// ---------------------------------------------------------------------------

func TestApaczkaRegistered(t *testing.T) {
	creds := json.RawMessage(`{"app_id":"a","app_secret":"b"}`)
	p, err := integration.NewCarrierProvider("apaczka", creds, nil)
	require.NoError(t, err)
	assert.Equal(t, "apaczka", p.ProviderName())
}

// ---------------------------------------------------------------------------
// TestApaczkaGetRates — service structure + valuation → Rate slice
// ---------------------------------------------------------------------------

func TestApaczkaGetRates(t *testing.T) {
	srv := newApaczkaMockServer(t, map[string]http.HandlerFunc{
		"/service_structure/": func(w http.ResponseWriter, _ *http.Request) {
			body := map[string]any{
				"services": []map[string]any{
					{
						"service_id":   "SVC001",
						"name":         "DHL Parcel",
						"supplier":     "DHL",
						"door_to_door": "1",
					},
					{
						"service_id":   "SVC002",
						"name":         "InPost Paczkomat",
						"supplier":     "INPOST",
						"door_to_door": "0",
					},
				},
			}
			apaczkaWriteJSON(w, body)
		},
		"/order_valuation/": func(w http.ResponseWriter, _ *http.Request) {
			body := map[string]any{
				"price_table": map[string]any{
					"SVC001": map[string]string{"price": "900", "price_gross": "1099"},
					"SVC002": map[string]string{"price": "1300", "price_gross": "1599"},
				},
			}
			apaczkaWriteJSON(w, body)
		},
	})
	defer srv.Close()

	// The SDK appends the route to baseURL; ensure baseURL ends with /
	p := newTestApaczkaProvider(t, srv.URL+"/")

	rates, err := p.GetRates(t.Context(), integration.RateRequest{
		FromPostalCode: "00-001",
		FromCountry:    "PL",
		ToPostalCode:   "30-001",
		ToCountry:      "PL",
		Weight:         1.5,
		Length:         30,
		Width:          20,
		Height:         10,
	})
	require.NoError(t, err)
	assert.Len(t, rates, 2)

	// Build a map for order-independent assertions
	byService := make(map[string]integration.Rate)
	for _, r := range rates {
		byService[r.ServiceName] = r
	}

	r1 := byService["SVC001"]
	assert.Equal(t, "DHL Parcel", r1.CarrierName)
	assert.Equal(t, "DHL", r1.CarrierCode)
	assert.Equal(t, "SVC001", r1.ServiceName)
	assert.InDelta(t, 10.99, r1.Price, 0.001)
	assert.Equal(t, "PLN", r1.Currency)
	assert.False(t, r1.PickupPoint, "door_to_door=1 means NOT a pickup-point-only service")

	r2 := byService["SVC002"]
	assert.Equal(t, "InPost Paczkomat", r2.CarrierName)
	assert.Equal(t, "INPOST", r2.CarrierCode)
	assert.Equal(t, "SVC002", r2.ServiceName)
	assert.InDelta(t, 15.99, r2.Price, 0.001)
	assert.True(t, r2.PickupPoint, "door_to_door=0 means pickup-point")
}

// ---------------------------------------------------------------------------
// TestApaczkaCreateShipment — order_send/ → CarrierShipmentResponse
// ---------------------------------------------------------------------------

func TestApaczkaCreateShipment(t *testing.T) {
	srv := newApaczkaMockServer(t, map[string]http.HandlerFunc{
		"/order_send/": func(w http.ResponseWriter, _ *http.Request) {
			body := map[string]any{
				"order": map[string]any{
					"id":             "ORD-999",
					"waybill_number": "WB12345678",
					"status":         "NEW",
				},
			}
			apaczkaWriteJSON(w, body)
		},
	})
	defer srv.Close()

	p := newTestApaczkaProvider(t, srv.URL+"/")

	resp, err := p.CreateShipment(t.Context(), integration.CarrierShipmentRequest{
		ServiceType: "SVC001",
		Receiver: integration.CarrierReceiver{
			Name:       "Jan Kowalski",
			Street:     "Testowa 1",
			City:       "Kraków",
			PostalCode: "30-001",
			Country:    "PL",
			Phone:      "600123456",
			Email:      "jan@example.com",
		},
		Parcel: integration.CarrierParcel{
			WeightKg: 1.5,
			DepthCm:  30,
			WidthCm:  20,
			HeightCm: 10,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "ORD-999", resp.ExternalID)
	assert.Equal(t, "WB12345678", resp.TrackingNumber)
	assert.Equal(t, "NEW", resp.Status)
}

// ---------------------------------------------------------------------------
// TestApaczkaCreateShipmentMissingServiceType — error when ServiceType is empty
// ---------------------------------------------------------------------------

func TestApaczkaCreateShipmentMissingServiceType(t *testing.T) {
	p := newTestApaczkaProvider(t, "http://localhost/")

	_, err := p.CreateShipment(t.Context(), integration.CarrierShipmentRequest{
		// ServiceType intentionally empty
		Receiver: integration.CarrierReceiver{Name: "Test"},
		Parcel:   integration.CarrierParcel{WeightKg: 1},
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "service_type"))
}

// ---------------------------------------------------------------------------
// TestApaczkaGetTracking — tracking/ → []TrackingEvent
// ---------------------------------------------------------------------------

func TestApaczkaGetTracking(t *testing.T) {
	srv := newApaczkaMockServer(t, map[string]http.HandlerFunc{
		"/tracking/WB12345678/": func(w http.ResponseWriter, _ *http.Request) {
			body := map[string]any{
				"tracking": []map[string]any{
					{
						"status":      "IN_TRANSIT",
						"description": "Paczka w transporcie",
						"place":       "Warszawa",
						"updated_at":  "2026-01-15T10:30:00Z",
					},
				},
			}
			apaczkaWriteJSON(w, body)
		},
	})
	defer srv.Close()

	p := newTestApaczkaProvider(t, srv.URL+"/")

	events, err := p.GetTracking(t.Context(), "WB12345678")
	require.NoError(t, err)
	require.Len(t, events, 1)

	ev := events[0]
	assert.Equal(t, "IN_TRANSIT", ev.Status)
	assert.Equal(t, "Warszawa", ev.Location)
	assert.Equal(t, "Paczka w transporcie", ev.Details)
	assert.Equal(t, "2026-01-15T10:30:00Z", ev.Timestamp.UTC().Format("2006-01-02T15:04:05Z"))
}

// ---------------------------------------------------------------------------
// TestApaczkaGetLabel — waybill/ → PDF bytes
// ---------------------------------------------------------------------------

func TestApaczkaGetLabel(t *testing.T) {
	pdfBytes := []byte("%PDF-1.4 test label")
	encoded := base64.StdEncoding.EncodeToString(pdfBytes)

	srv := newApaczkaMockServer(t, map[string]http.HandlerFunc{
		"/waybill/ORD-999/": func(w http.ResponseWriter, _ *http.Request) {
			body := map[string]any{
				"waybill": encoded,
				"type":    "pdf",
			}
			apaczkaWriteJSON(w, body)
		},
	})
	defer srv.Close()

	p := newTestApaczkaProvider(t, srv.URL+"/")

	data, err := p.GetLabel(t.Context(), "ORD-999", "pdf")
	require.NoError(t, err)
	assert.Equal(t, pdfBytes, data)
}

// ---------------------------------------------------------------------------
// TestApaczkaProviderBehaviours — SupportsPickupPoints, MapStatus
// ---------------------------------------------------------------------------

func TestApaczkaProviderBehaviours(t *testing.T) {
	p := newTestApaczkaProvider(t, "http://localhost/")

	assert.True(t, p.SupportsPickupPoints())

	omsStatus, ok := p.MapStatus("DELIVERED")
	assert.True(t, ok)
	assert.Equal(t, "delivered", omsStatus)

	_, ok = p.MapStatus("UNKNOWN")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// TestApaczkaSearchPickupPoints — points/ → []PickupPoint
// ---------------------------------------------------------------------------

func TestApaczkaSearchPickupPoints(t *testing.T) {
	srv := newApaczkaMockServer(t, map[string]http.HandlerFunc{
		// Apaczka point search is carrier-type scoped; the provider calls points/INPOST/.
		"/points/INPOST/": func(w http.ResponseWriter, _ *http.Request) {
			body := map[string]any{
				"points": []map[string]any{
					{
						"id":          "KRA01",
						"name":        "Paczkomat KRA01",
						"line1":       "ul. Test 1",
						"postal_code": "30-001",
						"city":        "Kraków",
						"lat":         50.06,
						"lng":         19.94,
					},
				},
			}
			apaczkaWriteJSON(w, body)
		},
	})
	defer srv.Close()

	p := newTestApaczkaProvider(t, srv.URL+"/")

	points, err := p.SearchPickupPoints(t.Context(), "30-001")
	require.NoError(t, err)
	require.Len(t, points, 1)

	pt := points[0]
	assert.Equal(t, "KRA01", pt.ID)
	assert.Equal(t, "Paczkomat KRA01", pt.Name)
	assert.Equal(t, "ul. Test 1", pt.Street) // mapped from Point.Line1
	assert.Equal(t, "Kraków", pt.City)
	assert.Equal(t, "30-001", pt.PostalCode)
	assert.InDelta(t, 50.06, pt.Latitude, 0.001)
	assert.InDelta(t, 19.94, pt.Longitude, 0.001)
	// The provider does not populate Type for Apaczka points.
	assert.Empty(t, pt.Type)
}

// ---------------------------------------------------------------------------
// TestApaczkaNewProviderErrors — credential validation
// ---------------------------------------------------------------------------

func TestApaczkaNewProviderErrors(t *testing.T) {
	t.Run("empty app_id", func(t *testing.T) {
		creds := json.RawMessage(`{"app_id":"","app_secret":"secret"}`)
		_, err := NewApaczkaProvider(creds, nil)
		require.Error(t, err)
	})

	t.Run("empty app_secret", func(t *testing.T) {
		creds := json.RawMessage(`{"app_id":"id","app_secret":""}`)
		_, err := NewApaczkaProvider(creds, nil)
		require.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := NewApaczkaProvider(json.RawMessage(`not-json`), nil)
		require.Error(t, err)
	})
}
