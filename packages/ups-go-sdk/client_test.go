package ups

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// newAuthenticatedServer creates a test server that handles both OAuth token
// requests and regular API requests. It auto-issues tokens on /security/v1/oauth/token.
func newAuthenticatedServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/security/v1/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tokenResponse{ //nolint:gosec // G117: test response
				AccessToken: "test-access-token",
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			})
			return
		}
		handler(w, r)
	}))
}

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient("test-client-id", "test-client-secret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("cid", "csecret")

	if c.clientID != "cid" {
		t.Errorf("clientID = %q, want %q", c.clientID, "cid")
	}
	if c.clientSecret != "csecret" {
		t.Errorf("clientSecret = %q, want %q", c.clientSecret, "csecret")
	}
	if c.baseURL != productionBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, productionBaseURL)
	}
	if c.Shipments == nil {
		t.Error("Shipments service is nil")
	}
}

func TestWithSandbox(t *testing.T) {
	c := NewClient("cid", "csecret", WithSandbox())

	if c.baseURL != sandboxBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, sandboxBaseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("cid", "csecret", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("cid", "csecret", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set by WithHTTPClient")
	}
}

func TestAuthenticate(t *testing.T) {
	var authCalled atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/security/v1/oauth/token" {
			authCalled.Add(1)
			user, pass, ok := r.BasicAuth()
			if !ok {
				t.Error("expected Basic Auth on token endpoint")
			}
			if user != "test-cid" {
				t.Errorf("auth user = %q, want %q", user, "test-cid")
			}
			if pass != "test-csecret" {
				t.Errorf("auth pass = %q, want %q", pass, "test-csecret")
			}
			if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tokenResponse{ //nolint:gosec // G117: test response
				AccessToken: "fresh-token",
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			})
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer fresh-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer fresh-token")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{}})
	}))
	defer srv.Close()

	c := NewClient("test-cid", "test-csecret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "1Z999AA10123456784")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}

	if authCalled.Load() != 1 {
		t.Errorf("auth called %d times, want 1", authCalled.Load())
	}
}

func TestAuthenticateReusesToken(t *testing.T) {
	var authCalled atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/security/v1/oauth/token" {
			authCalled.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tokenResponse{ //nolint:gosec // G117: test response
				AccessToken: "cached-token",
				ExpiresIn:   3600,
				TokenType:   "Bearer",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{}})
	}))
	defer srv.Close()

	c := NewClient("cid", "csecret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	// Make two requests; auth should only be called once
	_, _ = c.Shipments.GetTracking(context.Background(), "1Z1")
	_, _ = c.Shipments.GetTracking(context.Background(), "1Z2")

	if authCalled.Load() != 1 {
		t.Errorf("auth called %d times, want 1 (should reuse cached token)", authCalled.Load())
	}
}

func TestAuthenticateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message": "Invalid credentials"}`))
	}))
	defer srv.Close()

	c := NewClient("bad-cid", "bad-csecret",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Shipments.GetTracking(context.Background(), "1Z999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateShipment(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/shipments/v2409/ship" {
			t.Errorf("path = %q, want /shipments/v2409/ship", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var req ShipmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.ShipTo.Name != "Jan Kowalski" {
			t.Errorf("ShipTo.Name = %q, want %q", req.ShipTo.Name, "Jan Kowalski")
		}
		if req.Service.Code != "03" {
			t.Errorf("Service.Code = %q, want %q", req.Service.Code, "03")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ShipmentResponse{
			ShipmentID:     "SHP-001",
			TrackingNumber: "1Z999AA10123456784",
			LabelImage:     base64.StdEncoding.EncodeToString([]byte("%PDF-1.4 fake")),
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	resp, err := c.Shipments.Create(context.Background(), &ShipmentRequest{
		Shipper: Party{
			Name: "Sklep Online",
			Address: Address{
				AddressLine: []string{"Krakowska 10"},
				City:        "Krakow",
				PostalCode:  "30-001",
				CountryCode: "PL",
			},
		},
		ShipTo: Party{
			Name: "Jan Kowalski",
			Address: Address{
				AddressLine: []string{"Marszalkowska 1"},
				City:        "Warszawa",
				PostalCode:  "00-001",
				CountryCode: "PL",
			},
		},
		Service: ServiceCode{Code: "03", Description: "UPS Ground"},
		Package: []PackageSpec{
			{
				PackagingType: Code{Code: "02"},
				PackageWeight: PkgWeight{
					UnitOfMeasurement: Code{Code: "KGS"},
					Weight:            "2.5",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if resp.ShipmentID != "SHP-001" {
		t.Errorf("ShipmentID = %q, want %q", resp.ShipmentID, "SHP-001")
	}
	if resp.TrackingNumber != "1Z999AA10123456784" {
		t.Errorf("TrackingNumber = %q, want %q", resp.TrackingNumber, "1Z999AA10123456784")
	}
}

func TestGetLabel(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake UPS label")
	encoded := base64.StdEncoding.EncodeToString(pdfContent)

	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/shipments/v2409/SHP-001/label" {
			t.Errorf("path = %q, want /shipments/v2409/SHP-001/label", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"labelImage": encoded,
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	data, err := c.Shipments.GetLabel(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if string(data) != string(pdfContent) {
		t.Errorf("label data mismatch: got %q, want %q", string(data), string(pdfContent))
	}
}

func TestGetTracking(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/track/v1/details/1Z999AA10123456784" {
			t.Errorf("path = %q, want /track/v1/details/1Z999AA10123456784", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TrackingResponse{
			Events: []TrackingEvent{
				{Status: "P", Location: "Krakow", Description: "Pickup"},
				{Status: "I", Location: "Lodz", Description: "In Transit"},
				{Status: "D", Location: "Warszawa", Description: "Delivered"},
			},
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	resp, err := c.Shipments.GetTracking(context.Background(), "1Z999AA10123456784")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("len(Events) = %d, want 3", len(resp.Events))
	}
	if resp.Events[0].Status != "P" {
		t.Errorf("Events[0].Status = %q, want %q", resp.Events[0].Status, "P")
	}
	if resp.Events[0].Location != "Krakow" {
		t.Errorf("Events[0].Location = %q, want %q", resp.Events[0].Location, "Krakow")
	}
	if resp.Events[2].Status != "D" {
		t.Errorf("Events[2].Status = %q, want %q", resp.Events[2].Status, "D")
	}
	if resp.Events[2].Description != "Delivered" {
		t.Errorf("Events[2].Description = %q, want %q", resp.Events[2].Description, "Delivered")
	}
}

func TestCancelShipment(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/shipments/v2409/SHP-001" {
			t.Errorf("path = %q, want /shipments/v2409/SHP-001", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	err := c.Shipments.Cancel(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}
}

func TestCreateShipmentError(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid shipment data",
			"code":    "VALIDATION_ERROR",
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Shipments.Create(context.Background(), &ShipmentRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServerError(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal server error",
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Shipments.GetTracking(context.Background(), "INVALID")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if apiErr.Message != "Internal server error" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Internal server error")
	}
}

func TestNotFoundError(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Shipment not found",
		})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Shipments.GetLabel(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
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
			want: "ups: api error 400: Bad request",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 503},
			want: "ups: api error 503",
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
		ups    string
		oms    string
		wantOK bool
	}{
		{"M", "label_ready", true},
		{"P", "picked_up", true},
		{"I", "in_transit", true},
		{"O", "out_for_delivery", true},
		{"D", "delivered", true},
		{"DO", "delivered", true},
		{"W", "in_transit", true},
		{"X", "failed", true},
		{"RS", "returned", true},
		{"NONEXISTENT", "", false},
	}

	for _, tc := range tests {
		oms, ok := MapStatus(tc.ups)
		if ok != tc.wantOK {
			t.Errorf("MapStatus(%q) ok = %v, want %v", tc.ups, ok, tc.wantOK)
		}
		if oms != tc.oms {
			t.Errorf("MapStatus(%q) = %q, want %q", tc.ups, oms, tc.oms)
		}
	}
}

func TestDoSetsBearerToken(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-access-token")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{}})
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Shipments.GetTracking(context.Background(), "1Z1")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
}

func TestEmptyErrorBody(t *testing.T) {
	srv := newAuthenticatedServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Shipments.GetTracking(context.Background(), "1Z1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
	}
}
