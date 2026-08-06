package ebay

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
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

// pushedEbayQuantities runs PushOffer against a stub eBay API and returns the
// quantity sent on the inventory item and on the offer.
func pushedEbayQuantities(t *testing.T, product *model.Product, listingData map[string]any) (itemQty, offerQty int) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		switch {
		case strings.HasPrefix(r.URL.Path, "/sell/inventory/v1/inventory_item/"):
			var item struct {
				Availability struct {
					ShipToLocationAvailability struct {
						Quantity int `json:"quantity"`
					} `json:"shipToLocationAvailability"`
				} `json:"availability"`
			}
			if err := json.Unmarshal(body, &item); err != nil {
				t.Fatalf("decode inventory item: %v", err)
			}
			itemQty = item.Availability.ShipToLocationAvailability.Quantity
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(r.URL.Path, "/publish"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"listingId":"LISTING-1"}`))
		case r.URL.Path == "/sell/inventory/v1/offer":
			var offer struct {
				AvailableQuantity int `json:"availableQuantity"`
			}
			if err := json.Unmarshal(body, &offer); err != nil {
				t.Fatalf("decode offer: %v", err)
			}
			offerQty = offer.AvailableQuantity
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"offerId":"OFFER-1"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := ebaysdk.NewClient("app", "cert", "dev", "tok",
		ebaysdk.WithBaseURL(server.URL),
		ebaysdk.WithHTTPClient(server.Client()),
		ebaysdk.WithAccessToken("test_token"),
	)
	provider := &Provider{client: client, logger: slog.Default(), currency: "PLN"}

	if _, err := provider.PushOffer(context.Background(), product, listingData); err != nil {
		t.Fatalf("PushOffer returned error: %v", err)
	}
	return itemQty, offerQty
}

func ebayTestListingData() map[string]any {
	return map[string]any{
		"category_id":           "1234",
		"fulfillment_policy_id": "FP-1",
		"return_policy_id":      "RP-1",
	}
}

func TestPushOfferUsesCanonicalAvailableStock(t *testing.T) {
	sku := "SKU-CANON"
	// Legacy stock_quantity is not decremented on shipment, so it can overstate
	// what is really on the shelf. The listing must carry the canonical value.
	product := &model.Product{
		Name:           "Widget",
		SKU:            &sku,
		Price:          19.99,
		StockQuantity:  42,
		AvailableStock: 7,
	}

	itemQty, offerQty := pushedEbayQuantities(t, product, ebayTestListingData())

	if itemQty != 7 {
		t.Fatalf("inventory item quantity = %d, want 7 (canonical available stock)", itemQty)
	}
	if offerQty != 7 {
		t.Fatalf("offer availableQuantity = %d, want 7 (canonical available stock)", offerQty)
	}
}

func TestPushOfferStockOverrideWinsOverAvailableStock(t *testing.T) {
	sku := "SKU-CANON"
	product := &model.Product{
		Name:           "Widget",
		SKU:            &sku,
		Price:          19.99,
		StockQuantity:  42,
		AvailableStock: 7,
	}
	listingData := ebayTestListingData()
	listingData["stock_override"] = 3

	itemQty, offerQty := pushedEbayQuantities(t, product, listingData)

	if itemQty != 3 {
		t.Fatalf("inventory item quantity = %d, want 3 (stock_override)", itemQty)
	}
	if offerQty != 3 {
		t.Fatalf("offer availableQuantity = %d, want 3 (stock_override)", offerQty)
	}
}
