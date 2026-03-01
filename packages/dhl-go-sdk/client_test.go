package dhl

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAccountNumber(t *testing.T) {
	c := NewClient("user", "pass", "ACC123")
	if c.AccountNumber() != "ACC123" {
		t.Errorf("AccountNumber() = %q, want %q", c.AccountNumber(), "ACC123")
	}
}

func TestWithSandbox(t *testing.T) {
	// DHL24 has no separate sandbox — WithSandbox is a no-op
	c := NewClient("user", "pass", "ACC", WithSandbox())
	if c.baseURL != productionBaseURL {
		t.Errorf("WithSandbox changed baseURL to %q, expected no change", c.baseURL)
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("user", "pass", "ACC", WithBaseURL("https://custom.test"))
	if c.baseURL != "https://custom.test" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.test")
	}
}

func newSOAPResponse(method, content string) string {
	return `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<` + method + `Response>` + content + `</` + method + `Response>
	</soap:Body>
</soap:Envelope>`
}

func TestCreateShipment_SendsSOAP(t *testing.T) {
	var gotContentType, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(newSOAPResponse("createShipments", "<shipmentId>SHP-001</shipmentId><trackingNumber>TRK123</trackingNumber>")))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	resp, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{
		Receiver:    Receiver{Name: "Jan Kowalski", City: "Warszawa", PostalCode: "00-001", Country: "PL"},
		Piece:       Piece{Weight: 2.5},
		ServiceType: "AH",
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if resp.ShipmentID != "SHP-001" {
		t.Errorf("ShipmentID = %q, want SHP-001", resp.ShipmentID)
	}
	if !strings.Contains(gotContentType, "text/xml") {
		t.Errorf("Content-Type = %q, want text/xml", gotContentType)
	}
	if !strings.Contains(gotBody, "Envelope") {
		t.Error("body should contain SOAP Envelope")
	}
	if !strings.Contains(gotBody, "user") {
		t.Error("body should contain username in AuthData")
	}
}

func TestGetLabel_SendsSOAP(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 test")
	encoded := base64.StdEncoding.EncodeToString(pdfContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(newSOAPResponse("getLabels", "<labelData>"+encoded+"</labelData>")))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	data, err := c.Shipments.GetLabel(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("GetLabel() error: %v", err)
	}
	if string(data) != string(pdfContent) {
		t.Errorf("label mismatch: got %q, want %q", string(data), string(pdfContent))
	}
}

func TestGetTracking_SendsSOAP(t *testing.T) {
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(newSOAPResponse("getTrackAndTraceInfo", "<events></events>")))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	resp, err := c.Shipments.GetTracking(context.Background(), "TRK123")
	if err != nil {
		t.Fatalf("GetTracking() error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !strings.Contains(gotBody, "getTrackAndTraceInfo") {
		t.Error("body should contain SOAP method name")
	}
}

func TestCancel_SendsSOAP(t *testing.T) {
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(newSOAPResponse("deleteShipment", "")))
	}))
	defer srv.Close()

	c := NewClient("user", "pass", "ACC", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	err := c.Shipments.Cancel(context.Background(), "SHP-001")
	if err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}
	if !strings.Contains(gotBody, "deleteShipment") {
		t.Error("body should contain deleteShipment method")
	}
}

func TestSOAPFault_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<soap:Fault>
			<faultcode>soap:Server</faultcode>
			<faultstring>Invalid credentials</faultstring>
		</soap:Fault>
	</soap:Body>
</soap:Envelope>`))
	}))
	defer srv.Close()

	c := NewClient("bad", "bad", "ACC", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	_, err := c.Shipments.Create(context.Background(), &CreateShipmentRequest{})
	if err == nil {
		t.Fatal("expected error for SOAP fault, got nil")
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
			t.Errorf("MapStatus(%q) ok=%v, want %v", tc.dhl, ok, tc.wantOK)
		}
		if ok && oms != tc.oms {
			t.Errorf("MapStatus(%q)=%q, want %q", tc.dhl, oms, tc.oms)
		}
	}
}
