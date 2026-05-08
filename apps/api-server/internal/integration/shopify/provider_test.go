package shopify

import (
	"strings"
	"testing"

	shopifysdk "github.com/openoms-org/openoms/packages/shopify-go-sdk"
)

func TestMapShopifyOrderItemExternalIDPrefersVariantID(t *testing.T) {
	productID := int64(111)
	variantID := int64(222)
	provider := &Provider{}

	order, err := provider.mapShopifyOrder(&shopifysdk.Order{
		ID:         42,
		TotalPrice: "49.99",
		LineItems: []shopifysdk.LineItem{
			{
				ProductID:     &productID,
				VariantID:     &variantID,
				Name:          "Koszulka / XL",
				Quantity:      1,
				Price:         "49.99",
				TotalDiscount: "0.00",
			},
		},
	})
	if err != nil {
		t.Fatalf("mapShopifyOrder returned error: %v", err)
	}

	if len(order.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(order.Items))
	}
	if order.Items[0].ExternalID != "222" {
		t.Fatalf("ExternalID = %q, want variant ID 222", order.Items[0].ExternalID)
	}
}

func TestMapShopifyOrderItemExternalIDFallsBackToProductID(t *testing.T) {
	productID := int64(111)
	provider := &Provider{}

	order, err := provider.mapShopifyOrder(&shopifysdk.Order{
		ID:         42,
		TotalPrice: "19.99",
		LineItems: []shopifysdk.LineItem{
			{
				ProductID:     &productID,
				Name:          "Produkt bez wariantu",
				Quantity:      1,
				Price:         "19.99",
				TotalDiscount: "0.00",
			},
		},
	})
	if err != nil {
		t.Fatalf("mapShopifyOrder returned error: %v", err)
	}

	if len(order.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(order.Items))
	}
	if order.Items[0].ExternalID != "111" {
		t.Fatalf("ExternalID = %q, want product ID 111", order.Items[0].ExternalID)
	}
}

func TestMapShopifyOrderRejectsInvalidTotalPrice(t *testing.T) {
	provider := &Provider{}

	_, err := provider.mapShopifyOrder(&shopifysdk.Order{
		ID:         42,
		TotalPrice: "not-money",
	})
	if err == nil {
		t.Fatal("expected invalid total price error")
	}
	if !strings.Contains(err.Error(), "total_price") {
		t.Fatalf("error = %q, want total_price context", err.Error())
	}
}

func TestMapShopifyOrderRejectsInvalidLineItemMoney(t *testing.T) {
	provider := &Provider{}

	tests := []struct {
		name      string
		lineItem  shopifysdk.LineItem
		wantField string
	}{
		{
			name: "price",
			lineItem: shopifysdk.LineItem{
				ID:            7,
				Name:          "Produkt",
				Quantity:      1,
				Price:         "bad-price",
				TotalDiscount: "0.00",
			},
			wantField: "line_items[0].price",
		},
		{
			name: "total discount",
			lineItem: shopifysdk.LineItem{
				ID:            7,
				Name:          "Produkt",
				Quantity:      1,
				Price:         "19.99",
				TotalDiscount: "bad-discount",
			},
			wantField: "line_items[0].total_discount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.mapShopifyOrder(&shopifysdk.Order{
				ID:         42,
				TotalPrice: "19.99",
				LineItems:  []shopifysdk.LineItem{tt.lineItem},
			})
			if err == nil {
				t.Fatal("expected invalid line item money error")
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Fatalf("error = %q, want %s context", err.Error(), tt.wantField)
			}
		})
	}
}
