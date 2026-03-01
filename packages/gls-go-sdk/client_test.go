package gls

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("testuser", "testpass")

	if c.username != "testuser" {
		t.Errorf("username = %q, want %q", c.username, "testuser")
	}
	if c.password != "testpass" {
		t.Errorf("password = %q, want %q", c.password, "testpass")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Shipments == nil {
		t.Error("Shipments service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("key", "pass", WithSandbox())

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("key", "pass", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestDoSetsBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth")
		}
		if user != "test-user" {
			t.Errorf("username = %q, want test-user", user)
		}
		if pass != "test-pass" {
			t.Errorf("password = %q, want test-pass", pass)
		}
		if r.Header.Get("Accept") != "application/glsVersion1+json" {
			t.Errorf("Accept = %q, want application/glsVersion1+json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("test-user", "test-pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
}

func TestCreateParcel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/shipments" {
			t.Errorf("path = %q, want /shipments", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/glsVersion1+json" {
			t.Errorf("Content-Type = %q, want application/glsVersion1+json", r.Header.Get("Content-Type"))
		}

		var req CreateParcelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Consignee.Address.Name1 != "Jan Kowalski" {
			t.Errorf("Consignee.Address.Name1 = %q, want %q", req.Consignee.Address.Name1, "Jan Kowalski")
		}
		if len(req.ShippingUnit) != 1 {
			t.Fatalf("len(ShippingUnit) = %d, want 1", len(req.ShippingUnit))
		}
		if req.ShippingUnit[0].Weight != 2.0 {
			t.Errorf("ShippingUnit[0].Weight = %f, want 2.0", req.ShippingUnit[0].Weight)
		}
		if len(req.References) == 0 || req.References[0] != "ORDER-001" {
			t.Errorf("References[0] = %v, want ORDER-001", req.References)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"CreatedShipment":{"ShipmentReference":"ORDER-001","ParcelData":[{"TrackID":"TRK-001"},{"TrackID":"TRK-002"}]}}`))
	}))
	defer srv.Close()

	c := NewClient("api-user", "api-pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Product: "PARCEL",
		Consignee: Consignee{
			Address: ConsigneeAddress{
				Name1:       "Jan Kowalski",
				Street:      "Marszalkowska 1",
				City:        "Warszawa",
				ZIPCode:     "00-001",
				CountryCode: "PL",
				Phone:       "500100200",
			},
		},
		ShippingUnit: []ShipmentUnit{
			{Weight: 2.0, Width: 30, Height: 20, Depth: 40},
		},
		References: []string{"ORDER-001"},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if len(resp.ParcelIDs) != 2 {
		t.Fatalf("len(ParcelIDs) = %d, want 2", len(resp.ParcelIDs))
	}
	if resp.ParcelIDs[0] != "TRK-001" {
		t.Errorf("ParcelIDs[0] = %q, want %q", resp.ParcelIDs[0], "TRK-001")
	}
	if len(resp.TrackIDs) != 2 {
		t.Fatalf("len(TrackIDs) = %d, want 2", len(resp.TrackIDs))
	}
	if resp.TrackIDs[0] != "TRK-001" {
		t.Errorf("TrackIDs[0] = %q, want %q", resp.TrackIDs[0], "TRK-001")
	}
}

func TestGetLabel_ReturnsError(t *testing.T) {
	// GLS labels are embedded in the create shipment response (PrintData).
	// GetLabel as a separate call is not supported.
	c := NewClient("api-user", "api-pass", WithBaseURL("http://unused"))

	_, err := c.Shipments.GetLabel(context.Background(), "GLS-001")
	if err == nil {
		t.Fatal("GetLabel() should return error — GLS does not support separate label retrieval")
	}
}

func TestGetTracking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/shipments/parceldetails" {
			t.Errorf("path = %q, want /shipments/parceldetails", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TrackingResponse{
			Events: []TrackingEvent{
				{Status: "PREADVICE", Location: "Krakow", Details: "Paczka zarejestrowana"},
				{Status: "INTRANSIT", Location: "Lodz", Details: "W drodze"},
				{Status: "DELIVERED", Location: "Warszawa", Details: "Doreczona"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("api-user", "api-pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.GetTracking(context.Background(), "TRK-001")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("len(Events) = %d, want 3", len(resp.Events))
	}
	if resp.Events[0].Status != "PREADVICE" {
		t.Errorf("Events[0].Status = %q, want %q", resp.Events[0].Status, "PREADVICE")
	}
	if resp.Events[0].Location != "Krakow" {
		t.Errorf("Events[0].Location = %q, want %q", resp.Events[0].Location, "Krakow")
	}
	if resp.Events[2].Status != "DELIVERED" {
		t.Errorf("Events[2].Status = %q, want %q", resp.Events[2].Status, "DELIVERED")
	}
	if resp.Events[2].Details != "Doreczona" {
		t.Errorf("Events[2].Details = %q, want %q", resp.Events[2].Details, "Doreczona")
	}
}

func TestCancelParcel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/shipments/cancel/GLS-001" {
			t.Errorf("path = %q, want /shipments/cancel/GLS-001", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("api-user", "api-pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Shipments.Cancel(context.Background(), "GLS-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}
}

func TestCreateParcelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid parcel data",
			"code":    "VALIDATION_ERROR",
		})
	}))
	defer srv.Close()

	c := NewClient("api-user", "api-pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	}))
	defer srv.Close()

	c := NewClient("api-user", "api-pass",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if apiErr.Message != "Internal server error" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Internal server error")
	}
}

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  APIError
		want string
	}{
		{
			name: "with message",
			err:  APIError{StatusCode: 400, Message: "Bad request"},
			want: "gls: api error 400: Bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 503},
			want: "gls: api error 503",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		gls    string
		oms    string
		wantOK bool
	}{
		{"PREADVICE", "pending", true},
		{"INTRANSIT", "in_transit", true},
		{"INWAREHOUSE", "in_transit", true},
		{"INDELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		{"CANCELLED", "cancelled", true},
		{"CANCELLATION_PENDING", "cancelled", true},
		{"SCANNED", "in_transit", true},
		{"NONEXISTENT", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.gls)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.gls, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.gls, oms, tc.oms)
		}
	}
}
