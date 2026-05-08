package woocommerce

import (
	"strings"
	"testing"

	woocommercesdk "github.com/openoms-org/openoms/packages/woocommerce-go-sdk"
)

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
