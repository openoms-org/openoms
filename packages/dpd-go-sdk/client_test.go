package dpd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// newTestDPDServer creates a test server that handles DPD auth and API calls.
// The auth endpoint always returns a valid token.
func newTestDPDServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle authentication
		if r.URL.Path == "/auth/login" {
			var req authRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode auth request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(authResponse{Token: "test-session-token"})
			return
		}

		// Verify Bearer token
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-session-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-session-token")
		}

		handler(w, r)
	}))
}

func TestAuthentication(t *testing.T) {
	var authCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			authCalled = true
			var req authRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Login != "testuser" {
				t.Errorf("auth login = %q, want %q", req.Login, "testuser")
			}
			if req.Password != "testpass" {
				t.Errorf("auth password = %q, want %q", req.Password, "testpass")
			}
			if req.MasterFid != "FID001" {
				t.Errorf("auth masterFid = %q, want %q", req.MasterFid, "FID001")
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(authResponse{Token: "session-abc"})
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("testuser", "testpass", "FID001",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.do(context.Background(), "GET", "/test", nil, nil)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
	if !authCalled {
		t.Error("authentication endpoint was not called")
	}
}

func TestCreateParcel(t *testing.T) {
	srv := newTestDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/parcels" {
			t.Errorf("path = %q, want /parcels", r.URL.Path)
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
		json.NewEncoder(w).Encode(CreateParcelResponse{
			ParcelID: "PRC-001",
			Waybill:  "PL1234567890",
			Status:   "NEW",
		})
	})
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.Create(context.Background(), &CreateParcelRequest{
		Sender: Address{
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
	if resp.Status != "NEW" {
		t.Errorf("Status = %q, want %q", resp.Status, "NEW")
	}
}

func TestGetLabel(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake label")
	encoded := base64.StdEncoding.EncodeToString(pdfContent)

	srv := newTestDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/parcels/PRC-001/label" {
			t.Errorf("path = %q, want /parcels/PRC-001/label", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LabelResponse{
			LabelData:   encoded,
			LabelFormat: "PDF",
		})
	})
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	data, err := c.Shipments.GetLabel(context.Background(), "PRC-001")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if string(data) != string(pdfContent) {
		t.Errorf("label data mismatch: got %q, want %q", string(data), string(pdfContent))
	}
}

func TestGetTracking(t *testing.T) {
	srv := newTestDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/tracking/PL1234567890" {
			t.Errorf("path = %q, want /tracking/PL1234567890", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TrackingResponse{
			Events: []TrackingEvent{
				{Status: "SENT", Description: "Paczka nadana", Location: "Warszawa"},
				{Status: "IN_TRANSIT", Description: "W drodze", Location: "Lodz"},
				{Status: "DELIVERED", Description: "Doreczona"},
			},
		})
	})
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	resp, err := c.Shipments.GetTracking(context.Background(), "PL1234567890")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("len(Events) = %d, want 3", len(resp.Events))
	}
	if resp.Events[0].Status != "SENT" {
		t.Errorf("Events[0].Status = %q, want %q", resp.Events[0].Status, "SENT")
	}
	if resp.Events[0].Location != "Warszawa" {
		t.Errorf("Events[0].Location = %q, want %q", resp.Events[0].Location, "Warszawa")
	}
	if resp.Events[2].Status != "DELIVERED" {
		t.Errorf("Events[2].Status = %q, want %q", resp.Events[2].Status, "DELIVERED")
	}
}

func TestCancelParcel(t *testing.T) {
	srv := newTestDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/parcels/PRC-001" {
			t.Errorf("path = %q, want /parcels/PRC-001", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Shipments.Cancel(context.Background(), "PRC-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}
}

func TestCreateParcelError(t *testing.T) {
	srv := newTestDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Nieprawidlowe dane przesylki",
			"code":    "VALIDATION_ERROR",
		})
	})
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
	srv := newTestDPDServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	})
	defer srv.Close()

	c := NewClient("user", "pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuthenticationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"Invalid credentials"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("bad-user", "bad-pass", "FID",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "PL123")
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
		{"NEW", "pending", true},
		{"SENT", "in_transit", true},
		{"IN_TRANSIT", "in_transit", true},
		{"OUT_FOR_DELIVERY", "out_for_delivery", true},
		{"DELIVERED", "delivered", true},
		{"RETURNED", "returned", true},
		{"PICKUP_AT_POINT", "ready_for_pickup", true},
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
