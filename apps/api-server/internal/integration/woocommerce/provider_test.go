package woocommerce

import (
	"testing"

	woocommercesdk "github.com/openoms-org/openoms/packages/woocommerce-go-sdk"
)

func TestMapWooOrderPaymentStatusOnHoldIsPending(t *testing.T) {
	provider := &Provider{}
	order := provider.mapWooOrder(&woocommercesdk.WooOrder{
		ID:     42,
		Status: "on-hold",
	})

	if order.PaymentStatus != "pending" {
		t.Fatalf("PaymentStatus = %q, want pending", order.PaymentStatus)
	}
}

func TestMapWooOrderPaymentStatusPaidStatuses(t *testing.T) {
	provider := &Provider{}
	for _, status := range []string{"processing", "completed"} {
		t.Run(status, func(t *testing.T) {
			order := provider.mapWooOrder(&woocommercesdk.WooOrder{
				ID:     42,
				Status: status,
			})

			if order.PaymentStatus != "paid" {
				t.Fatalf("PaymentStatus = %q, want paid", order.PaymentStatus)
			}
		})
	}
}
