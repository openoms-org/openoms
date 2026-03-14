package btp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvoicesGet(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/InvoicesGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req InvoicesRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Documents) != 1 {
			t.Fatalf("expected 1 document, got %d", len(req.Documents))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InvoicesResponse{
			Invoices: []InvoiceDocument{
				{
					Header: InvoiceDocumentHeader{
						InvoiceNumber: "FV/2026/001",
						Currency:      "PLN",
					},
					Lines: []InvoiceLine{
						{LineNumber: 1, ItemName: "Product A", Quantity: 5, UnitNetPrice: 10.00},
					},
					Summary: &InvoiceSummary{
						TotalNetAmount:   50.00,
						TotalTaxAmount:   11.50,
						TotalGrossAmount: 61.50,
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.Invoices.Get(context.Background(), &InvoicesRequest{
		Documents: []InvoicesRequestDocument{
			{DocumentID: "FV/2026/001", DocumentType: "INV"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(resp.Invoices))
	}
	if resp.Invoices[0].Header.InvoiceNumber != "FV/2026/001" {
		t.Fatalf("expected FV/2026/001, got %s", resp.Invoices[0].Header.InvoiceNumber)
	}
	if resp.Invoices[0].Summary.TotalGrossAmount != 61.50 {
		t.Fatalf("expected 61.50, got %f", resp.Invoices[0].Summary.TotalGrossAmount)
	}
}

func TestInvoiceGetImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/InvoiceImageGet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("documentId") != "FV001" {
			t.Fatalf("expected documentId=FV001, got %s", q.Get("documentId"))
		}
		if q.Get("documentType") != "INV" {
			t.Fatalf("expected documentType=INV, got %s", q.Get("documentType"))
		}
		if q.Get("imageFormat") != "PDF" {
			t.Fatalf("expected imageFormat=PDF, got %s", q.Get("imageFormat"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InvoiceImageResponse{
			ImageBase64: "JVBERi0xLjQ=",
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.Invoices.GetImage(context.Background(), &InvoiceImageRequest{
		DocumentID:   "FV001",
		DocumentType: "INV",
		ImageFormat:  "PDF",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ImageBase64 != "JVBERi0xLjQ=" {
		t.Fatalf("unexpected base64: %s", resp.ImageBase64)
	}
}

func TestInvoiceConfigureNotify(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/InvoiceNotifyConfigure" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var cfg DocumentNotifyConfig
		_ = json.NewDecoder(r.Body).Decode(&cfg)
		if cfg.EndpointURL != "https://example.com/webhook" {
			t.Fatalf("expected endpoint URL, got %s", cfg.EndpointURL)
		}
		if cfg.Format != "JSON" {
			t.Fatalf("expected JSON format, got %s", cfg.Format)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(APIResult{ResultType: "INFORMATION"})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	result, err := c.Invoices.ConfigureNotify(context.Background(), &DocumentNotifyConfig{
		EndpointURL: "https://example.com/webhook",
		HTTPMethod:  "POST",
		Mode:        "WholeBody",
		Format:      "JSON",
		Authentication: DocumentNotifyAuth{
			AuthType: "Bearer",
			AuthKey1: "secret-token",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResultType != "INFORMATION" {
		t.Fatalf("expected INFORMATION, got %s", result.ResultType)
	}
}
