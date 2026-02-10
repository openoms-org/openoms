package inpost

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLabelGenerate(t *testing.T) {
	pdfData := []byte("%PDF-1.4 fake label content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/shipments/labels" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req generateLabelsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if len(req.ShipmentIDs) != 2 {
			t.Fatalf("expected 2 shipment IDs, got %d", len(req.ShipmentIDs))
		}
		if req.Format != "Pdf" {
			t.Fatalf("expected format Pdf, got %s", req.Format)
		}
		if req.Type != "normal" {
			t.Fatalf("expected type normal, got %s", req.Type)
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write(pdfData)
	}))
	defer srv.Close()

	c := NewClient("tok", "org1", WithBaseURL(srv.URL))
	data, err := c.Labels.Generate(context.Background(), []int64{1, 2}, LabelPDF, LabelPageNormal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(data, pdfData) {
		t.Fatalf("expected PDF data %q, got %q", pdfData, data)
	}
}

func TestLabelGet(t *testing.T) {
	zplData := []byte("^XA^FO50,50^FDHello^FS^XZ")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/shipments/42/label" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("format") != "Zpl" {
			t.Fatalf("expected format=Zpl, got %s", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(zplData)
	}))
	defer srv.Close()

	c := NewClient("tok", "org1", WithBaseURL(srv.URL))
	data, err := c.Labels.Get(context.Background(), 42, LabelZPL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(data, zplData) {
		t.Fatalf("expected ZPL data %q, got %q", zplData, data)
	}
}

func TestLabelGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"shipment not found"}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org1", WithBaseURL(srv.URL))
	_, err := c.Labels.Get(context.Background(), 99999, LabelPDF)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestLabelGenerateServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	c := NewClient("tok", "org1", WithBaseURL(srv.URL))
	_, err := c.Labels.Generate(context.Background(), []int64{1}, LabelPDF, LabelPageNormal)
	if err == nil {
		t.Fatal("expected error for 500")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Fatalf("expected status 500, got %d", apiErr.StatusCode)
	}
}
