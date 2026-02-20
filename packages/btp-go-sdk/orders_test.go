package btp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrderCreate(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/OrderCreate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var req OrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if len(req.Lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(req.Lines))
		}
		if req.Lines[0].BarCode != "5901234567890" {
			t.Fatalf("expected barcode 5901234567890, got %s", req.Lines[0].BarCode)
		}
		if req.ClientOrderNumber != "ORD-12345" {
			t.Fatalf("expected client order number ORD-12345, got %s", req.ClientOrderNumber)
		}
		if !req.IsTesting {
			t.Fatal("expected isTesting=true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrderResponse{
			Result:      APIResult{ResultType: "INFORMATION"},
			OrderID:     "ord-abc-123",
			OrderNumber: "ZAM/2026/001",
			OrderDate:   "2026-02-19T10:00:00Z",
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.Orders.Create(context.Background(), &OrderRequest{
		Lines: []OrderRequestLine{
			{BarCode: "5901234567890", Quantity: 2},
			{ItemID: "ITEM002", Quantity: 1},
		},
		ClientOrderNumber: "ORD-12345",
		IsTesting:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OrderID != "ord-abc-123" {
		t.Fatalf("expected order ID ord-abc-123, got %s", resp.OrderID)
	}
	if resp.OrderNumber != "ZAM/2026/001" {
		t.Fatalf("expected order number ZAM/2026/001, got %s", resp.OrderNumber)
	}
}

func TestOrderCreateValidationError(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResult{
			ResultType: "ERROR",
			Messages: []APIResultMessage{
				{MessageType: "ERROR", Message: "Lines collection is empty"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	_, err := c.Orders.Create(context.Background(), &OrderRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Fatalf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestOrderSetAttachments(t *testing.T) {
	srv := httptest.NewServer(withCSRF(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Gateway/ClientApi/OrderAttachmentsSet" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var req OrderAttachmentsRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.OrderID != "ord-123" {
			t.Fatalf("expected order ID ord-123, got %s", req.OrderID)
		}
		if len(req.AttachmentFileIDs) != 2 {
			t.Fatalf("expected 2 file IDs, got %d", len(req.AttachmentFileIDs))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OrderAttachmentsResponse{
			Result:          APIResult{ResultType: "INFORMATION"},
			OrderID:         "ord-123",
			AttachmentCount: 2,
		})
	}))
	defer srv.Close()

	c := NewClient("u", "p", WithBaseURL(srv.URL))
	resp, err := c.Orders.SetAttachments(context.Background(), &OrderAttachmentsRequest{
		OrderID:           "ord-123",
		AttachmentFileIDs: []string{"file-1", "file-2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AttachmentCount != 2 {
		t.Fatalf("expected 2 attachments, got %d", resp.AttachmentCount)
	}
}
