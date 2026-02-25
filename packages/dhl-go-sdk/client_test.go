package dhl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("user", "pass", "ACC123")

	if c.username != "user" {
		t.Errorf("username = %q, want %q", c.username, "user")
	}
	if c.password != "pass" {
		t.Errorf("password = %q, want %q", c.password, "pass")
	}
	if c.accountNum != "ACC123" {
		t.Errorf("accountNum = %q, want %q", c.accountNum, "ACC123")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Shipments == nil {
		t.Error("Shipments service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("user", "pass", "ACC123", WithSandbox())

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("user", "pass", "ACC123", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestAccountNumber(t *testing.T) {
	c := NewClient("user", "pass", "ACC123")
	if c.AccountNumber() != "ACC123" {
		t.Errorf("AccountNumber() = %q, want %q", c.AccountNumber(), "ACC123")
	}
}

func TestDoSetsBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth header")
		}
		if user != "testuser" {
			t.Errorf("username = %q, want %q", user, "testuser")
		}
		if pass != "testpass" {
			t.Errorf("password = %q, want %q", pass, "testpass")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("testuser", "testpass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() returned error: %v", err)
	}
}

func TestCreateShipment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/v1/shipments" {
			t.Errorf("path = %q, want /v1/shipments", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var req CreateShipmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Receiver.Name != "Jan Kowalski" {
			t.Errorf("Receiver.Name = %q, want %q", req.Receiver.Name, "Jan Kowalski")
		}
		if req.Piece.Weight != 2.5 {
			t.Errorf("Piece.Weight = %f, want 2.5", req.Piece.Weight)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ShipmentResponse{
			ShipmentID:     "SHP-001",
			TrackingNumber: "1234567890",
			Status:         "CREATED",
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		ShipperAccount: "ACC",
		Receiver: Receiver{
			Name:       "Jan Kowalski",
			Street:     "Marszalkowska 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		Shipper: Shipper{
			Name:       "Sklep Online",
			Street:     "Krakowska 10",
			City:       "Krakow",
			PostalCode: "30-001",
			Country:    "PL",
		},
		Piece: Piece{
			Weight: 2.5,
			Width:  30,
			Height: 20,
			Length: 40,
		},
		ServiceType: "AH",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if resp.ShipmentID != "SHP-001" {
		t.Errorf("ShipmentID = %q, want %q", resp.ShipmentID, "SHP-001")
	}
	if resp.TrackingNumber != "1234567890" {
		t.Errorf("TrackingNumber = %q, want %q", resp.TrackingNumber, "1234567890")
	}
	if resp.Status != "CREATED" {
		t.Errorf("Status = %q, want %q", resp.Status, "CREATED")
	}
}

func TestGetLabel(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake label content")
	encoded := base64.StdEncoding.EncodeToString(pdfContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/v1/shipments/SHP-001/label" {
			t.Errorf("path = %q, want /v1/shipments/SHP-001/label", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LabelResponse{
			LabelData:   encoded,
			LabelFormat: "PDF",
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	data, err := c.Shipments.GetLabel(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if string(data) != string(pdfContent) {
		t.Errorf("label data mismatch: got %q, want %q", string(data), string(pdfContent))
	}
}

func TestGetTracking(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/v1/tracking" {
			t.Errorf("path = %q, want /v1/tracking", r.URL.Path)
		}
		if r.URL.Query().Get("trackingNumber") != "1234567890" {
			t.Errorf("trackingNumber = %q, want 1234567890", r.URL.Query().Get("trackingNumber"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TrackingResponse{
			ShipmentID:     "SHP-001",
			TrackingNumber: "1234567890",
			Events: []TrackingEvent{
				{Status: "PICKED_UP", Location: "Warszawa"},
				{Status: "IN_TRANSIT", Location: "Lodz"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.GetTracking(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if resp.TrackingNumber != "1234567890" {
		t.Errorf("TrackingNumber = %q, want %q", resp.TrackingNumber, "1234567890")
	}
	if len(resp.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(resp.Events))
	}
	if resp.Events[0].Status != "PICKED_UP" {
		t.Errorf("Events[0].Status = %q, want %q", resp.Events[0].Status, "PICKED_UP")
	}
	if resp.Events[1].Location != "Lodz" {
		t.Errorf("Events[1].Location = %q, want %q", resp.Events[1].Location, "Lodz")
	}
}

func TestCancelShipment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/v1/shipments/SHP-001" {
			t.Errorf("path = %q, want /v1/shipments/SHP-001", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Shipments.Cancel(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}
}

func TestCreateShipmentError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid shipment data",
			"code":    "VALIDATION_ERROR",
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		// The error is wrapped, so unwrap it
		t.Logf("error type: %T, value: %v", err, err)
	} else {
		if apiErr.StatusCode != 400 {
			t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
		}
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

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "INVALID")
	if err == nil {
		t.Fatal("expected error, got nil")
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
			want: "dhl: api error 400: Bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "dhl: api error 500",
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

func TestDoWithRequestBody(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"shipmentId":"123"}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	body := map[string]string{"key": "value"}
	var result map[string]any
	err := c.do(context.Background(), "POST", "/test", body, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if len(gotBody) == 0 {
		t.Error("expected request body, got empty")
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		dhl    string
		oms    string
		wantOK bool
	}{
		{"CREATED", "created", true},
		{"PICKED_UP", "picked_up", true},
		{"IN_TRANSIT", "in_transit", true},
		{"OUT_FOR_DELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		{"FAILED", "failed", true},
		{"LABEL_CREATED", "label_ready", true},
		{"CUSTOMS", "in_transit", true},
		{"NONEXISTENT", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.dhl)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.dhl, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.dhl, oms, tc.oms)
		}
	}
}
