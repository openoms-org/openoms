package dpd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("user", "pass", "FID123")

	if c.login != "user" {
		t.Errorf("login = %q, want %q", c.login, "user")
	}
	if c.password != "pass" {
		t.Errorf("password = %q, want %q", c.password, "pass")
	}
	if c.masterFid != "FID123" {
		t.Errorf("masterFid = %q, want %q", c.masterFid, "FID123")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Shipments == nil {
		t.Error("Shipments service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("user", "pass", "FID", WithSandbox())

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("user", "pass", "FID", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestMasterFid(t *testing.T) {
	c := NewClient("user", "pass", "FID123")
	if c.MasterFid() != "FID123" {
		t.Errorf("MasterFid() = %q, want %q", c.MasterFid(), "FID123")
	}
}

func TestAuthentication_UsesBasicAuth(t *testing.T) {
	var gotUser, gotPass, gotFid string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic Auth credentials")
		}
		gotUser = user
		gotPass = pass
		gotFid = r.Header.Get("x-dpd-fid")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","sessionId":1,"packages":[]}`))
	}))
	defer srv.Close()

	c := NewClient("testuser", "testpass", "FID001",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, _ = c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})

	if gotUser != "testuser" {
		t.Errorf("Basic Auth user = %q, want %q", gotUser, "testuser")
	}
	if gotPass != "testpass" {
		t.Errorf("Basic Auth pass = %q, want %q", gotPass, "testpass")
	}
	if gotFid != "FID001" {
		t.Errorf("x-dpd-fid = %q, want %q", gotFid, "FID001")
	}
}

func TestCreateParcel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/public/shipment/v1/generatePackagesNumbers" {
			t.Errorf("path = %q, want /public/shipment/v1/generatePackagesNumbers", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var req CreateParcelRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Receiver.Name != "Jan Kowalski" {
			t.Errorf("Receiver.Name = %q, want %q", req.Receiver.Name, "Jan Kowalski")
		}
		if len(req.Parcels) != 1 {
			t.Fatalf("len(Parcels) = %d, want 1", len(req.Parcels))
		}
		if req.Parcels[0].Weight != 3.5 {
			t.Errorf("Parcels[0].Weight = %f, want 3.5", req.Parcels[0].Weight)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "OK",
			"sessionId": 42,
			"packages": [{
				"statusInfo": {"status": "OK"},
				"parcels": [{"status": "OK", "reference": "PRC-001", "waybill": "PL1234567890"}]
			}]
		}`))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Sender: &Address{
			Name:        "Sklep Online",
			Street:      "Krakowska 10",
			City:        "Krakow",
			PostalCode:  "30-001",
			CountryCode: "PL",
		},
		Receiver: Address{
			Name:        "Jan Kowalski",
			Street:      "Marszalkowska 1",
			City:        "Warszawa",
			PostalCode:  "00-001",
			CountryCode: "PL",
			Phone:       "500100200",
		},
		Parcels: []ParcelSpec{
			{Weight: 3.5, SizeX: 30, SizeY: 20, SizeZ: 15},
		},
		Reference: "ORDER-001",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if resp.ParcelID != "PRC-001" {
		t.Errorf("ParcelID = %q, want %q", resp.ParcelID, "PRC-001")
	}
	if resp.Waybill != "PL1234567890" {
		t.Errorf("Waybill = %q, want %q", resp.Waybill, "PL1234567890")
	}
	if resp.SessionID != 42 {
		t.Errorf("SessionID = %d, want 42", resp.SessionID)
	}
}

func TestGetLabel(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake label")
	encoded := base64.StdEncoding.EncodeToString(pdfContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/public/shipment/v1/generateSpedLabels" {
			t.Errorf("path = %q, want /public/shipment/v1/generateSpedLabels", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(generateLabelResponse{
			Status:       "OK",
			DocumentData: encoded,
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	data, err := c.Shipments.GetLabel(context.Background(), "PL1234567890")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if string(data) != string(pdfContent) {
		t.Errorf("label data mismatch: got %q, want %q", string(data), string(pdfContent))
	}
}

func TestGetTracking_ReturnsError(t *testing.T) {
	// DPD REST API does not have a tracking endpoint.
	c := NewClient("user", "pass", "FID")

	_, err := c.Shipments.GetTracking(context.Background(), "PL1234567890")
	if err == nil {
		t.Fatal("expected error from GetTracking, got nil")
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Errorf("error message should mention 'not available', got: %v", err)
	}
}

func TestCancelParcel_ReturnsError(t *testing.T) {
	// DPD REST API does not support parcel cancellation.
	c := NewClient("user", "pass", "FID")

	err := c.Shipments.Cancel(context.Background(), "PRC-001")
	if err == nil {
		t.Fatal("expected error from Cancel, got nil")
	}
}

func TestCreateParcelError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Nieprawidlowe dane przesylki",
			"code":    "VALIDATION_ERROR",
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuthenticationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Invalid credentials"}`))
	}))
	defer srv.Close()

	c := NewClient("bad-user", "bad-pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Receiver: Address{Name: "Test"},
		Parcels:  []ParcelSpec{{Weight: 1.0}},
	})
	if err == nil {
		t.Fatal("expected authentication error, got nil")
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
			want: "dpd: api error 400: Bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "dpd: api error 500",
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
		dpd    string
		oms    string
		wantOK bool
	}{
		{"NEW", "label_ready", true},
		{"SENT", "picked_up", true},
		{"IN_TRANSIT", "in_transit", true},
		{"OUT_FOR_DELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		{"PICKUP_AT_POINT", "out_for_delivery", true},
		{"CANCELLED", "failed", true},
		{"FAILED", "failed", true},
		{"REFUSED", "failed", true},
		{"LOST", "failed", true},
		{"DESTROYED", "failed", true},
		{"UNKNOWN_STATUS", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.dpd)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.dpd, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.dpd, oms, tc.oms)
		}
	}
}
