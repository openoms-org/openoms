package ksef

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(EnvironmentTest)

	if c.env != EnvironmentTest {
		t.Errorf("env = %q, want %q", c.env, EnvironmentTest)
	}
	if c.baseURL != "https://ksef-test.mf.gov.pl/api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://ksef-test.mf.gov.pl/api")
	}
	if c.Session == nil {
		t.Error("Session service is nil")
	}
	if c.Invoice == nil {
		t.Error("Invoice service is nil")
	}
}

func TestNewClientProduction(t *testing.T) {
	c := NewClient(EnvironmentProduction)

	if c.baseURL != "https://ksef.mf.gov.pl/api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://ksef.mf.gov.pl/api")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient(EnvironmentTest, WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestEnvironmentBaseURL(t *testing.T) {
	tests := []struct {
		env  Environment
		want string
	}{
		{EnvironmentTest, "https://ksef-test.mf.gov.pl/api"},
		{EnvironmentProduction, "https://ksef.mf.gov.pl/api"},
		{Environment("unknown"), "https://ksef-test.mf.gov.pl/api"},
	}

	for _, tc := range tests {
		got := tc.env.BaseURL()
		if got != tc.want {
			t.Errorf("Environment(%q).BaseURL() = %q, want %q", tc.env, got, tc.want)
		}
	}
}

func TestAuthorisationChallenge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/online/Session/AuthorisationChallenge" {
			t.Errorf("path = %q, want /online/Session/AuthorisationChallenge", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}

		var req AuthorisationChallengeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.ContextIdentifier.Identifier != "1234567890" {
			t.Errorf("NIP = %q, want %q", req.ContextIdentifier.Identifier, "1234567890")
		}
		if req.ContextIdentifier.Type != "onip" {
			t.Errorf("Type = %q, want %q", req.ContextIdentifier.Type, "onip")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AuthorisationChallengeResponse{
			Timestamp: "2024-01-15T10:00:00.000Z",
			Challenge: "challenge-token-abc123",
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	resp, err := c.Session.AuthorisationChallenge(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("AuthorisationChallenge() error: %v", err)
	}
	if resp.Challenge != "challenge-token-abc123" {
		t.Errorf("Challenge = %q, want %q", resp.Challenge, "challenge-token-abc123")
	}
	if resp.Timestamp != "2024-01-15T10:00:00.000Z" {
		t.Errorf("Timestamp = %q, want %q", resp.Timestamp, "2024-01-15T10:00:00.000Z")
	}
}

func TestSessionStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/online/Session/Status") {
			t.Errorf("path = %q, want prefix /online/Session/Status", r.URL.Path)
		}
		if r.Header.Get("SessionToken") != "session-token-xyz" {
			t.Errorf("SessionToken = %q, want %q", r.Header.Get("SessionToken"), "session-token-xyz")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SessionStatusResponse{
			Timestamp:             "2024-01-15T10:05:00.000Z",
			ReferenceNumber:       "REF-001",
			NumberOfElements:      2,
			ProcessingCode:        200,
			ProcessingDescription: "Przetwarzanie zakonczone",
			InvoiceStatusList: []InvoiceStatus{
				{
					InvoiceNumber:       "FV/2024/001",
					KsefReferenceNumber: "KSEF-12345",
					ProcessingCode:      200,
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	resp, err := c.Session.Status(context.Background(), "session-token-xyz", 10, 0)
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if resp.ReferenceNumber != "REF-001" {
		t.Errorf("ReferenceNumber = %q, want %q", resp.ReferenceNumber, "REF-001")
	}
	if resp.NumberOfElements != 2 {
		t.Errorf("NumberOfElements = %d, want 2", resp.NumberOfElements)
	}
	if len(resp.InvoiceStatusList) != 1 {
		t.Fatalf("len(InvoiceStatusList) = %d, want 1", len(resp.InvoiceStatusList))
	}
	if resp.InvoiceStatusList[0].KsefReferenceNumber != "KSEF-12345" {
		t.Errorf("KsefReferenceNumber = %q, want %q", resp.InvoiceStatusList[0].KsefReferenceNumber, "KSEF-12345")
	}
}

func TestSessionTerminate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/online/Session/Terminate" {
			t.Errorf("path = %q, want /online/Session/Terminate", r.URL.Path)
		}
		if r.Header.Get("SessionToken") != "session-token-xyz" {
			t.Errorf("SessionToken header missing or wrong")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TerminateSessionResponse{
			Timestamp:       "2024-01-15T10:10:00.000Z",
			ReferenceNumber: "REF-001",
			ProcessingCode:  200,
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	resp, err := c.Session.Terminate(context.Background(), "session-token-xyz")
	if err != nil {
		t.Fatalf("Terminate() error: %v", err)
	}
	if resp.ProcessingCode != 200 {
		t.Errorf("ProcessingCode = %d, want 200", resp.ProcessingCode)
	}
}

func TestInvoiceSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/online/Invoice/Send" {
			t.Errorf("path = %q, want /online/Invoice/Send", r.URL.Path)
		}
		if r.Header.Get("SessionToken") != "session-token-xyz" {
			t.Errorf("SessionToken = %q, want %q", r.Header.Get("SessionToken"), "session-token-xyz")
		}
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("Content-Type = %q, want application/octet-stream", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected XML body, got empty")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SendInvoiceResponse{
			ElementReferenceNumber: "ELEM-001",
			ReferenceNumber:        "REF-001",
			ProcessingCode:         200,
			ProcessingDescription:  "Przyjeto do przetwarzania",
			Timestamp:              "2024-01-15T10:15:00.000Z",
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	invoiceXML := []byte(`<?xml version="1.0"?><Faktura/>`)
	resp, err := c.Invoice.Send(context.Background(), "session-token-xyz", invoiceXML)
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if resp.ElementReferenceNumber != "ELEM-001" {
		t.Errorf("ElementReferenceNumber = %q, want %q", resp.ElementReferenceNumber, "ELEM-001")
	}
	if resp.ProcessingCode != 200 {
		t.Errorf("ProcessingCode = %d, want 200", resp.ProcessingCode)
	}
}

func TestInvoiceGetStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/online/Invoice/Status/ELEM-001" {
			t.Errorf("path = %q, want /online/Invoice/Status/ELEM-001", r.URL.Path)
		}
		if r.Header.Get("SessionToken") != "session-token-xyz" {
			t.Errorf("SessionToken header missing or wrong")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InvoiceStatus{
			InvoiceNumber:       "FV/2024/001",
			KsefReferenceNumber: "KSEF-12345",
			ProcessingCode:      200,
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	status, err := c.Invoice.GetStatus(context.Background(), "session-token-xyz", "ELEM-001")
	if err != nil {
		t.Fatalf("GetStatus() error: %v", err)
	}
	if status.KsefReferenceNumber != "KSEF-12345" {
		t.Errorf("KsefReferenceNumber = %q, want %q", status.KsefReferenceNumber, "KSEF-12345")
	}
	if status.InvoiceNumber != "FV/2024/001" {
		t.Errorf("InvoiceNumber = %q, want %q", status.InvoiceNumber, "FV/2024/001")
	}
}

func TestInvoiceGetUPO(t *testing.T) {
	upoData := "UPO-document-content"
	encoded := base64.StdEncoding.EncodeToString([]byte(upoData))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/common/Status/REF-001" {
			t.Errorf("path = %q, want /common/Status/REF-001", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UPOResponse{
			Timestamp:       "2024-01-15T10:20:00.000Z",
			ReferenceNumber: "REF-001",
			ProcessingCode:  200,
			UPO:             encoded,
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	data, err := c.Invoice.GetUPOBytes(context.Background(), "REF-001")
	if err != nil {
		t.Fatalf("GetUPOBytes() error: %v", err)
	}
	if string(data) != upoData {
		t.Errorf("UPO data = %q, want %q", string(data), upoData)
	}
}

func TestInvoiceGetUPONotReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UPOResponse{
			Timestamp:       "2024-01-15T10:20:00.000Z",
			ReferenceNumber: "REF-001",
			ProcessingCode:  100,
			UPO:             "", // UPO not yet available
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	_, err := c.Invoice.GetUPOBytes(context.Background(), "REF-001")
	if err == nil {
		t.Fatal("expected error for empty UPO, got nil")
	}
	if !strings.Contains(err.Error(), "not yet available") {
		t.Errorf("error = %q, want to contain 'not yet available'", err.Error())
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Brak autoryzacji",
			"code":    "UNAUTHORIZED",
		})
	}))
	defer srv.Close()

	c := NewClient(EnvironmentTest, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))

	_, err := c.Session.AuthorisationChallenge(context.Background(), "1234567890")
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
			err:  APIError{StatusCode: 401, Message: "Unauthorized"},
			want: "ksef: Unauthorized",
		},
		{
			name: "with description only",
			err:  APIError{StatusCode: 400, Description: "Invalid NIP"},
			want: "ksef: Invalid NIP",
		},
		{
			name: "empty",
			err:  APIError{StatusCode: 500},
			want: "ksef: API error",
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

func TestBuildInvoiceXML(t *testing.T) {
	data := InvoiceData{
		InvoiceDate:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		InvoiceNumber: "FV/2024/001",
		Currency:      "PLN",
		SellerNIP:     "1234567890",
		SellerName:    "Sprzedawca Sp. z o.o.",
		SellerStreet:  "Krakowska 10",
		SellerCity:    "Krakow",
		SellerPostal:  "30-001",
		BuyerNIP:      "9876543210",
		BuyerName:     "Kupujacy S.A.",
		BuyerStreet:   "Marszalkowska 1",
		BuyerCity:     "Warszawa",
		BuyerPostal:   "00-001",
		Items: []InvoiceLineItem{
			{
				LineNumber:  1,
				Name:        "Produkt testowy",
				Quantity:    2,
				Unit:        "szt.",
				NetPrice:    100.00,
				NetAmount:   200.00,
				VATRate:     "23",
				VATAmount:   46.00,
				GrossAmount: 246.00,
			},
		},
		TotalNet:    200.00,
		TotalVAT:    46.00,
		TotalGross:  246.00,
		PaymentDate: time.Date(2024, 1, 29, 0, 0, 0, 0, time.UTC),
		PaymentType: "przelew",
	}

	xmlBytes, err := BuildInvoiceXML(data)
	if err != nil {
		t.Fatalf("BuildInvoiceXML() error: %v", err)
	}

	xmlStr := string(xmlBytes)

	if !strings.Contains(xmlStr, "<NIP>1234567890</NIP>") {
		t.Error("XML should contain seller NIP")
	}
	if !strings.Contains(xmlStr, "<NIP>9876543210</NIP>") {
		t.Error("XML should contain buyer NIP")
	}
	if !strings.Contains(xmlStr, "<P_2>FV/2024/001</P_2>") {
		t.Error("XML should contain invoice number")
	}
	if !strings.Contains(xmlStr, "<P_7>Produkt testowy</P_7>") {
		t.Error("XML should contain item name")
	}
	if !strings.Contains(xmlStr, "<KodWaluty>PLN</KodWaluty>") {
		t.Error("XML should contain currency")
	}
	if !strings.Contains(xmlStr, "Faktura") {
		t.Error("XML should contain Faktura root element")
	}
}

func TestBuildInvoiceXMLValidation(t *testing.T) {
	// Missing seller NIP
	_, err := BuildInvoiceXML(InvoiceData{
		Items: []InvoiceLineItem{{Name: "test"}},
	})
	if err == nil {
		t.Fatal("expected error for missing seller NIP")
	}

	// Missing items
	_, err = BuildInvoiceXML(InvoiceData{
		SellerNIP: "1234567890",
	})
	if err == nil {
		t.Fatal("expected error for missing items")
	}
}

func TestBuildInvoiceXMLDefaults(t *testing.T) {
	data := InvoiceData{
		SellerNIP: "1234567890",
		Items: []InvoiceLineItem{
			{Name: "Test", Quantity: 1, NetPrice: 100, NetAmount: 100, VATRate: "23", VATAmount: 23, GrossAmount: 123},
		},
	}

	xmlBytes, err := BuildInvoiceXML(data)
	if err != nil {
		t.Fatalf("BuildInvoiceXML() error: %v", err)
	}

	xmlStr := string(xmlBytes)
	if !strings.Contains(xmlStr, "<KodWaluty>PLN</KodWaluty>") {
		t.Error("default currency should be PLN")
	}
	if !strings.Contains(xmlStr, "<KodKraju>PL</KodKraju>") {
		t.Error("default country should be PL")
	}
}

func TestMapPaymentType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"przelew", "6"},
		{"transfer", "6"},
		{"gotowka", "1"},
		{"cash", "1"},
		{"karta", "2"},
		{"card", "2"},
		{"unknown", "6"},
	}

	for _, tc := range tests {
		got := mapPaymentType(tc.input)
		if got != tc.want {
			t.Errorf("mapPaymentType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"a longer string", 5, "a lon..."},
		{"exact", 5, "exact"},
	}

	for _, tc := range tests {
		got := truncate([]byte(tc.input), tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
		}
	}
}

func TestEncryptTokenWithAES(t *testing.T) {
	token := "test-authorization-token"

	encrypted, aesKey, iv, err := EncryptTokenWithAES(token)
	if err != nil {
		t.Fatalf("EncryptTokenWithAES() error: %v", err)
	}

	if encrypted == "" {
		t.Error("encrypted token should not be empty")
	}
	if len(aesKey) != 32 {
		t.Errorf("AES key length = %d, want 32", len(aesKey))
	}
	if len(iv) != 16 {
		t.Errorf("IV length = %d, want 16", len(iv))
	}

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Errorf("encrypted token is not valid base64: %v", err)
	}
}
