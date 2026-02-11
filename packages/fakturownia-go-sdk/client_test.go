package fakturownia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvoiceService_Create(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/invoices.json" {
			t.Errorf("expected /invoices.json, got %s", r.URL.Path)
		}

		var req CreateInvoiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.APIToken != "test-token" {
			t.Errorf("expected api_token test-token, got %s", req.APIToken)
		}
		if req.Invoice.BuyerName != "Jan Kowalski" {
			t.Errorf("expected buyer_name Jan Kowalski, got %s", req.Invoice.BuyerName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvoiceResponse{
			ID:         123,
			Number:     "FV/2024/01/001",
			Status:     "issued",
			PriceNet:   "100.00",
			PriceGross: "123.00",
			Currency:   "PLN",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test", "test-token", WithBaseURL(server.URL))

	result, err := client.Invoices.Create(context.Background(), InvoiceRequestData{
		Kind:      "vat",
		SellDate:  "2024-01-15",
		IssueDate: "2024-01-15",
		PaymentTo: "2024-01-29",
		BuyerName: "Jan Kowalski",
		Positions: []InvoicePosition{
			{Name: "Produkt 1", Quantity: 2, NetPrice: 50.00, Tax: 23, Unit: "szt."},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
	if result.Number != "FV/2024/01/001" {
		t.Errorf("expected number FV/2024/01/001, got %s", result.Number)
	}
	if result.PriceGross != "123.00" {
		t.Errorf("expected price_gross 123.00, got %s", result.PriceGross)
	}
}

func TestInvoiceService_Get(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/invoices/123.json" {
			t.Errorf("expected /invoices/123.json, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("api_token") != "test-token" {
			t.Errorf("expected api_token test-token, got %s", r.URL.Query().Get("api_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InvoiceResponse{
			ID:     123,
			Number: "FV/2024/01/001",
			Status: "issued",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test", "test-token", WithBaseURL(server.URL))

	result, err := client.Invoices.Get(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
}

func TestInvoiceService_GetPDF(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 fake pdf content")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/invoices/123.pdf" {
			t.Errorf("expected /invoices/123.pdf, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Write(pdfContent)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test", "test-token", WithBaseURL(server.URL))

	data, err := client.Invoices.GetPDF(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != string(pdfContent) {
		t.Errorf("expected pdf content, got %s", string(data))
	}
}

func TestInvoiceService_Cancel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/invoices/123/cancel.json" {
			t.Errorf("expected /invoices/123/cancel.json, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test", "test-token", WithBaseURL(server.URL))

	err := client.Invoices.Cancel(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"code":    "unauthorized",
			"message": "Invalid API token",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient("test", "bad-token", WithBaseURL(server.URL))

	_, err := client.Invoices.Get(context.Background(), 123)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestParseExternalID(t *testing.T) {
	id, err := ParseExternalID("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Errorf("expected 123, got %d", id)
	}

	_, err = ParseExternalID("abc")
	if err == nil {
		t.Fatal("expected error for invalid ID, got nil")
	}
}
