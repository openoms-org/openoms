package woocommerce

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	woocommercesdk "github.com/openoms-org/openoms/packages/woocommerce-go-sdk"
)

func TestPollOrdersUsesGMTModifiedCursorAndModifiedOrdering(t *testing.T) {
	tests := []struct {
		name              string
		cursor            string
		wantDatesAreGMT   string
		wantModifiedAfter string
	}{
		{
			name:              "UTC cursor",
			cursor:            "2026-03-29T00:30:00Z",
			wantDatesAreGMT:   "true",
			wantModifiedAfter: "2026-03-29T00:30:00Z",
		},
		{
			name:              "legacy local-time cursor",
			cursor:            "2026-03-29T00:30:00",
			wantDatesAreGMT:   "",
			wantModifiedAfter: "2026-03-29T00:30:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[
					{
						"id": 101,
						"status": "processing",
						"currency": "PLN",
						"total": "10.00",
						"date_created": "2026-03-29T01:15:00",
						"date_modified": "2026-03-30T01:30:00",
						"date_modified_gmt": "2026-03-29T23:30:00"
					},
					{
						"id": 102,
						"status": "processing",
						"currency": "PLN",
						"total": "20.00",
						"date_created": "2026-03-29T01:20:00",
						"date_modified": "2026-03-30T03:15:00",
						"date_modified_gmt": "2026-03-30T01:15:00"
					}
				]`))
			}))
			defer srv.Close()

			provider := &Provider{
				client: woocommercesdk.NewClient("https://shop.example.com", "ck", "cs",
					woocommercesdk.WithBaseURL(srv.URL),
					woocommercesdk.WithHTTPClient(srv.Client()),
				),
			}

			orders, cursor, err := provider.PollOrders(context.Background(), tt.cursor)
			if err != nil {
				t.Fatalf("PollOrders returned error: %v", err)
			}
			if len(orders) != 2 {
				t.Fatalf("len(orders) = %d, want 2", len(orders))
			}
			if got := gotQuery.Get("modified_after"); got != tt.wantModifiedAfter {
				t.Fatalf("modified_after = %q, want %q", got, tt.wantModifiedAfter)
			}
			if got := gotQuery.Get("dates_are_gmt"); got != tt.wantDatesAreGMT {
				t.Fatalf("dates_are_gmt = %q, want %q", got, tt.wantDatesAreGMT)
			}
			if got := gotQuery.Get("orderby"); got != "modified" {
				t.Fatalf("orderby = %q, want modified", got)
			}
			if cursor != "2026-03-30T01:15:00Z" {
				t.Fatalf("cursor = %q, want 2026-03-30T01:15:00Z", cursor)
			}
		})
	}
}

func TestMapWooOrderPaymentStatusOnHoldIsPending(t *testing.T) {
	provider := &Provider{}
	order, err := provider.mapWooOrder(&woocommercesdk.WooOrder{
		ID:     42,
		Status: "on-hold",
		Total:  "0.00",
	})
	if err != nil {
		t.Fatalf("mapWooOrder returned error: %v", err)
	}

	if order.PaymentStatus != "pending" {
		t.Fatalf("PaymentStatus = %q, want pending", order.PaymentStatus)
	}
}

func TestMapWooOrderPaymentStatusPaidStatuses(t *testing.T) {
	provider := &Provider{}
	for _, status := range []string{"processing", "completed"} {
		t.Run(status, func(t *testing.T) {
			order, err := provider.mapWooOrder(&woocommercesdk.WooOrder{
				ID:     42,
				Status: status,
				Total:  "0.00",
			})
			if err != nil {
				t.Fatalf("mapWooOrder returned error: %v", err)
			}

			if order.PaymentStatus != "paid" {
				t.Fatalf("PaymentStatus = %q, want paid", order.PaymentStatus)
			}
		})
	}
}

func TestMapWooOrderRejectsInvalidTotal(t *testing.T) {
	provider := &Provider{}

	_, err := provider.mapWooOrder(&woocommercesdk.WooOrder{
		ID:    42,
		Total: "not-money",
	})
	if err == nil {
		t.Fatal("expected invalid total error")
	}
	if !strings.Contains(err.Error(), "total") {
		t.Fatalf("error = %q, want total context", err.Error())
	}
}

func TestMapWooOrderRejectsInvalidLineItemTotal(t *testing.T) {
	provider := &Provider{}

	_, err := provider.mapWooOrder(&woocommercesdk.WooOrder{
		ID:    42,
		Total: "19.99",
		LineItems: []woocommercesdk.WooLineItem{
			{
				ID:       7,
				Name:     "Produkt",
				Quantity: 1,
				Total:    "bad-total",
				Price:    19.99,
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid line item total error")
	}
	if !strings.Contains(err.Error(), "line_items[0].total") {
		t.Fatalf("error = %q, want line item total context", err.Error())
	}
}
