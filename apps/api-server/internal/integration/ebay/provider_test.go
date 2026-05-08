package ebay

import (
	"strings"
	"testing"

	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"
)

func TestMapEbayOrderSuccessfulMapping(t *testing.T) {
	provider := &Provider{}

	order, err := provider.mapEbayOrder(&ebaysdk.Order{
		OrderID:         "EBAY-42",
		OrderFulfStatus: "NOT_STARTED",
		PricingSummary: ebaysdk.PricingSummary{
			Total: ebaysdk.Amount{Value: "49.99", Currency: "PLN"},
		},
		LineItems: []ebaysdk.LineItem{
			{
				LineItemID:   "line-1",
				Title:        "Produkt",
				Quantity:     1,
				LineItemCost: ebaysdk.Amount{Value: "49.99", Currency: "PLN"},
				Total:        ebaysdk.Amount{Value: "49.99", Currency: "PLN"},
			},
		},
	})
	if err != nil {
		t.Fatalf("mapEbayOrder returned error: %v", err)
	}
	if order.ExternalID != "EBAY-42" {
		t.Fatalf("ExternalID = %q, want EBAY-42", order.ExternalID)
	}
	if order.TotalAmount != 49.99 {
		t.Fatalf("TotalAmount = %v, want 49.99", order.TotalAmount)
	}
	if len(order.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(order.Items))
	}
}

func TestMapEbayOrderRejectsInvalidTotal(t *testing.T) {
	provider := &Provider{}

	_, err := provider.mapEbayOrder(&ebaysdk.Order{
		OrderID: "EBAY-1",
		PricingSummary: ebaysdk.PricingSummary{
			Total: ebaysdk.Amount{Value: "not-money", Currency: "PLN"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid total error")
	}
	if !strings.Contains(err.Error(), "pricingSummary.total.value") {
		t.Fatalf("error = %q, want pricingSummary total context", err.Error())
	}
}

func TestMapEbayOrderRejectsInvalidLineItemCost(t *testing.T) {
	provider := &Provider{}

	_, err := provider.mapEbayOrder(&ebaysdk.Order{
		OrderID: "EBAY-1",
		PricingSummary: ebaysdk.PricingSummary{
			Total: ebaysdk.Amount{Value: "19.99", Currency: "PLN"},
		},
		LineItems: []ebaysdk.LineItem{
			{
				LineItemID:   "line-1",
				Title:        "Produkt",
				Quantity:     1,
				LineItemCost: ebaysdk.Amount{Value: "bad-cost", Currency: "PLN"},
				Total:        ebaysdk.Amount{Value: "19.99", Currency: "PLN"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid line item cost error")
	}
	if !strings.Contains(err.Error(), "lineItems[0].lineItemCost.value") {
		t.Fatalf("error = %q, want line item cost context", err.Error())
	}
}

func TestMapEbayOrderRejectsInvalidLineItemTotal(t *testing.T) {
	provider := &Provider{}

	_, err := provider.mapEbayOrder(&ebaysdk.Order{
		OrderID: "EBAY-1",
		PricingSummary: ebaysdk.PricingSummary{
			Total: ebaysdk.Amount{Value: "19.99", Currency: "PLN"},
		},
		LineItems: []ebaysdk.LineItem{
			{
				LineItemID:   "line-1",
				Title:        "Produkt",
				Quantity:     1,
				LineItemCost: ebaysdk.Amount{Value: "19.99", Currency: "PLN"},
				Total:        ebaysdk.Amount{Value: "bad-total", Currency: "PLN"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid line item total error")
	}
	if !strings.Contains(err.Error(), "lineItems[0].total.value") {
		t.Fatalf("error = %q, want line item total context", err.Error())
	}
}
