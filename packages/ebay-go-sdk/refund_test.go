package ebay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueRefund(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/sell/fulfillment/v1/order/04-12345-67890/issue_refund" {
			t.Errorf("path = %q, want /sell/fulfillment/v1/order/04-12345-67890/issue_refund", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_token" {
			t.Errorf("Authorization = %q, want Bearer test_token", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var req IssueRefundRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ReasonForRefund != "BUYER_CANCEL" {
			t.Errorf("reasonForRefund = %q, want BUYER_CANCEL", req.ReasonForRefund)
		}
		if len(req.RefundItems) != 1 {
			t.Fatalf("len(refundItems) = %d, want 1", len(req.RefundItems))
		}
		if req.RefundItems[0].LineItemID != "LI-001" {
			t.Errorf("lineItemId = %q, want LI-001", req.RefundItems[0].LineItemID)
		}
		if req.RefundItems[0].RefundAmount.Value != "25.00" {
			t.Errorf("refundAmount.value = %q, want 25.00", req.RefundItems[0].RefundAmount.Value)
		}

		resp := IssueRefundResponse{
			RefundID:     "REF-12345",
			RefundStatus: "PENDING",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	result, err := c.Fulfillment.IssueRefund(context.Background(), "04-12345-67890", IssueRefundRequest{
		ReasonForRefund: "BUYER_CANCEL",
		RefundItems: []RefundItem{
			{
				LineItemID:   "LI-001",
				RefundAmount: Amount{Value: "25.00", Currency: "PLN"},
			},
		},
	})
	if err != nil {
		t.Fatalf("IssueRefund error: %v", err)
	}
	if result.RefundID != "REF-12345" {
		t.Errorf("RefundID = %q, want REF-12345", result.RefundID)
	}
	if result.RefundStatus != "PENDING" {
		t.Errorf("RefundStatus = %q, want PENDING", result.RefundStatus)
	}
}

func TestIssueRefund_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"errorId":35000,"domain":"API_FULFILLMENT","category":"REQUEST","message":"Invalid refund reason"}]}`))
	}))
	defer srv.Close()

	c := NewClient("app", "cert", "dev", "tok",
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithAccessToken("test_token"),
	)

	_, err := c.Fulfillment.IssueRefund(context.Background(), "04-99999-00000", IssueRefundRequest{
		ReasonForRefund: "INVALID",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	got := err.Error()
	want := "ebay: issue refund for order 04-99999-00000: ebay: HTTP 400: Invalid refund reason"
	if got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}
