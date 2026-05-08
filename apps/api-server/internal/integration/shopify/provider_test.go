package shopify

import (
	"testing"

	shopifysdk "github.com/openoms-org/openoms/packages/shopify-go-sdk"
)

func TestMapShopifyOrderItemExternalIDPrefersVariantID(t *testing.T) {
	productID := int64(111)
	variantID := int64(222)
	provider := &Provider{}

	order := provider.mapShopifyOrder(&shopifysdk.Order{
		ID: 42,
		LineItems: []shopifysdk.LineItem{
			{
				ProductID: &productID,
				VariantID: &variantID,
				Name:      "Koszulka / XL",
				Quantity:  1,
				Price:     "49.99",
			},
		},
	})

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

	order := provider.mapShopifyOrder(&shopifysdk.Order{
		ID: 42,
		LineItems: []shopifysdk.LineItem{
			{
				ProductID: &productID,
				Name:      "Produkt bez wariantu",
				Quantity:  1,
				Price:     "19.99",
			},
		},
	})

	if len(order.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(order.Items))
	}
	if order.Items[0].ExternalID != "111" {
		t.Fatalf("ExternalID = %q, want product ID 111", order.Items[0].ExternalID)
	}
}
