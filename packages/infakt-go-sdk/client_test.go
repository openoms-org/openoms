package infakt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("api-key-123")

	if c.apiKey != "api-key-123" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "api-key-123")
	}
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
	if c.Invoices == nil {
		t.Error("Invoices service is nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	c := NewClient("key", WithBaseURL("https://custom.api"))

	if c.baseURL != "https://custom.api" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api")
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	c := NewClient("key", WithHTTPClient(custom))

	if c.httpClient != custom {
		t.Error("httpClient was not set to custom client")
	}
}

func TestDoSetsAPIKeyHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-inFakt-ApiKey")
		if apiKey != "test-api-key" {
			t.Errorf("X-inFakt-ApiKey = %q, want %q", apiKey, "test-api-key")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("test-api-key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	var result map[string]any
	err := c.do(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("do() error: %v", err)
	}
}

func TestInvoiceCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/invoices.json" {
			t.Errorf("path = %q, want /invoices.json", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if _, ok := body["invoice"]; !ok {
			t.Error("expected 'invoice' key in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceResponse{
			Invoice: Invoice{
				ID:         42,
				Kind:       "vat",
				Number:     "FV/2024/001",
				GrossPrice: "123.00",
				Currency:   "PLN",
				Status:     "created",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	inv, err := c.Invoices.Create(context.Background(), CreateInvoiceRequest{
		Kind:          "vat",
		InvoiceDate:   "2024-01-15",
		SaleDate:      "2024-01-15",
		PaymentDate:   "2024-01-29",
		PaymentMethod: "transfer",
		Currency:      "PLN",
		Client: InvoiceClient{
			Name:       "Firma ABC",
			NIP:        "1234567890",
			Street:     "Marszalkowska 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		Services: []InvoiceLineItem{
			{
				Name:         "Usługa programistyczna",
				Unit:         "szt",
				Quantity:     1,
				UnitNetPrice: 100.00,
				TaxSymbol:    "23",
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if inv.ID != 42 {
		t.Errorf("ID = %d, want 42", inv.ID)
	}
	if inv.Kind != "vat" {
		t.Errorf("Kind = %q, want %q", inv.Kind, "vat")
	}
	if inv.Number != "FV/2024/001" {
		t.Errorf("Number = %q, want %q", inv.Number, "FV/2024/001")
	}
	if inv.GrossPrice != "123.00" {
		t.Errorf("GrossPrice = %q, want %q", inv.GrossPrice, "123.00")
	}
}

func TestInvoiceGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/invoices/42.json" {
			t.Errorf("path = %q, want /invoices/42.json", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceResponse{
			Invoice: Invoice{
				ID:     42,
				Kind:   "vat",
				Number: "FV/2024/001",
				Status: "sent",
			},
		})
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	inv, err := c.Invoices.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if inv.ID != 42 {
		t.Errorf("ID = %d, want 42", inv.ID)
	}
	if inv.Status != "sent" {
		t.Errorf("Status = %q, want %q", inv.Status, "sent")
	}
}

func TestInvoiceList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %q, want 1", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("per_page") != "25" {
			t.Errorf("per_page = %q, want 25", r.URL.Query().Get("per_page"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceListResponse{
			Entities: []Invoice{
				{ID: 1, Number: "FV/2024/001"},
				{ID: 2, Number: "FV/2024/002"},
			},
			Total: 2,
		})
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	invoices, err := c.Invoices.List(context.Background(), ListInvoicesParams{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(invoices) != 2 {
		t.Fatalf("len(invoices) = %d, want 2", len(invoices))
	}
	if invoices[0].Number != "FV/2024/001" {
		t.Errorf("invoices[0].Number = %q, want %q", invoices[0].Number, "FV/2024/001")
	}
	if invoices[1].Number != "FV/2024/002" {
		t.Errorf("invoices[1].Number = %q, want %q", invoices[1].Number, "FV/2024/002")
	}
}

func TestInvoiceListWithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "3" {
			t.Errorf("page = %q, want 3", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("per_page") != "10" {
			t.Errorf("per_page = %q, want 10", r.URL.Query().Get("per_page"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invoiceListResponse{Entities: []Invoice{}})
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	invoices, err := c.Invoices.List(context.Background(), ListInvoicesParams{Page: 3, Limit: 10})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(invoices) != 0 {
		t.Errorf("len(invoices) = %d, want 0", len(invoices))
	}
}

func TestInvoiceDownloadPDF(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake invoice content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/invoices/42.pdf" {
			t.Errorf("path = %q, want /invoices/42.pdf", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfContent)
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	data, err := c.Invoices.DownloadPDF(context.Background(), 42)
	if err != nil {
		t.Fatalf("DownloadPDF() error: %v", err)
	}
	if string(data) != string(pdfContent) {
		t.Errorf("PDF data mismatch: got %q, want %q", string(data), string(pdfContent))
	}
}

func TestInvoiceSendByEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/invoices/42/deliver_via_email.json" {
			t.Errorf("path = %q, want /invoices/42/deliver_via_email.json", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["recipient"] != "client@example.com" {
			t.Errorf("recipient = %v, want client@example.com", body["recipient"])
		}
		if body["print_type"] != "original" {
			t.Errorf("print_type = %v, want original", body["print_type"])
		}
		if body["locale"] != "pl" {
			t.Errorf("locale = %v, want pl", body["locale"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	err := c.Invoices.SendByEmail(context.Background(), 42, "client@example.com")
	if err != nil {
		t.Fatalf("SendByEmail() error: %v", err)
	}
}

func TestInvoiceCreateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid invoice data",
			"code":    "VALIDATION_ERROR",
		})
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Invoices.Create(context.Background(), CreateInvoiceRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
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

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Invoices.Get(context.Background(), 999)
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
}

func TestNotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invoice not found",
		})
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Invoices.Get(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json}`))
	}))
	defer srv.Close()

	c := NewClient("key",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)

	_, err := c.Invoices.Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
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
			err:  APIError{StatusCode: 422, Message: "Unprocessable entity"},
			want: "infakt: Unprocessable entity",
		},
		{
			name: "without message",
			err:  APIError{StatusCode: 500},
			want: "infakt: unexpected status 500",
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

func TestParseExternalID(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"42", 42, false},
		{"1", 1, false},
		{"0", 0, false},
		{"abc", 0, true},
		{"", 0, true},
		{"12.5", 0, true},
	}

	for _, tc := range tests {
		got, err := ParseExternalID(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseExternalID(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseExternalID(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	dt := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	got := FormatDate(dt)
	want := "2024-03-15"
	if got != want {
		t.Errorf("FormatDate() = %q, want %q", got, want)
	}
}
