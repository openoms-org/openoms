package shopify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	shopifysdk "github.com/openoms-org/openoms/packages/shopify-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func newShopifyProviderForTest(t *testing.T, handler http.HandlerFunc) (*Provider, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := shopifysdk.NewClient("test-shop", "shpat_test-token",
		shopifysdk.WithBaseURL(server.URL),
		shopifysdk.WithHTTPClient(server.Client()),
	)

	return &Provider{client: client}, server.Close
}

func TestPushOfferCreatesShopifyProductAndReturnsVariantID(t *testing.T) {
	sku := "SKU-123"
	ean := "5901234123457"

	provider, closeServer := newShopifyProviderForTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/products.json" {
			t.Fatalf("request = %s %s, want POST /products.json", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Shopify-Access-Token") != "shpat_test-token" {
			t.Fatalf("missing Shopify access token header")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload struct {
			Product struct {
				Title    string           `json:"title"`
				BodyHTML string           `json:"body_html"`
				Variants []map[string]any `json:"variants"`
			} `json:"product"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if payload.Product.Title != "Test Shopify Product" {
			t.Fatalf("title = %q, want product name", payload.Product.Title)
		}
		if payload.Product.BodyHTML != "Long description" {
			t.Fatalf("body_html = %q, want long description", payload.Product.BodyHTML)
		}
		if len(payload.Product.Variants) != 1 {
			t.Fatalf("len(variants) = %d, want 1", len(payload.Product.Variants))
		}
		variant := payload.Product.Variants[0]
		if variant["sku"] != sku {
			t.Fatalf("variant sku = %#v, want %q", variant["sku"], sku)
		}
		if variant["barcode"] != ean {
			t.Fatalf("variant barcode = %#v, want %q", variant["barcode"], ean)
		}
		if variant["price"] != "123.45" {
			t.Fatalf("variant price = %#v, want 123.45", variant["price"])
		}
		if variant["inventory_management"] != "shopify" {
			t.Fatalf("variant inventory_management = %#v, want shopify", variant["inventory_management"])
		}
		if variant["inventory_quantity"] != float64(9) {
			t.Fatalf("variant inventory_quantity = %#v, want 9", variant["inventory_quantity"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"product": {
				"id": 111,
				"title": "Test Shopify Product",
				"variants": [
					{"id": 222, "product_id": 111, "inventory_item_id": 333}
				]
			}
		}`))
	})
	defer closeServer()

	externalID, err := provider.PushOffer(context.Background(), &model.Product{
		ID:              uuid.New(),
		Name:            "Test Shopify Product",
		SKU:             &sku,
		EAN:             &ean,
		Price:           123.45,
		StockQuantity:   9,
		DescriptionLong: "Long description",
	}, nil)
	if err != nil {
		t.Fatalf("PushOffer returned error: %v", err)
	}
	if externalID != "222" {
		t.Fatalf("externalID = %q, want real Shopify variant ID 222", externalID)
	}
	if strings.HasPrefix(externalID, "shopify-") {
		t.Fatalf("externalID = %q, must not be a synthetic local ID", externalID)
	}
}

func TestPushOfferRejectsCreateResponseWithoutVariantID(t *testing.T) {
	provider, closeServer := newShopifyProviderForTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/products.json" {
			t.Fatalf("request = %s %s, want POST /products.json", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"product":{"id":111,"title":"Missing variant","variants":[]}}`))
	})
	defer closeServer()

	externalID, err := provider.PushOffer(context.Background(), &model.Product{
		ID:    uuid.New(),
		Name:  "Missing variant",
		Price: 10,
	}, nil)
	if err == nil {
		t.Fatal("expected missing variant error")
	}
	if externalID != "" {
		t.Fatalf("externalID = %q, want empty on failure", externalID)
	}
	if !strings.Contains(err.Error(), "variant") {
		t.Fatalf("error = %q, want variant context", err.Error())
	}
}

func TestUpdateStockResolvesVariantIDToInventoryItem(t *testing.T) {
	var sawSetLevel bool

	provider, closeServer := newShopifyProviderForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/variants/222.json":
			_, _ = w.Write([]byte(`{"variant":{"id":222,"product_id":111,"inventory_item_id":333}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/inventory_levels.json":
			if got := r.URL.Query().Get("inventory_item_ids"); got != "333" {
				t.Fatalf("inventory_item_ids = %q, want 333", got)
			}
			_, _ = w.Write([]byte(`{"inventory_levels":[{"inventory_item_id":333,"location_id":444,"available":3}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/inventory_levels/set.json":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode set level body: %v", err)
			}
			if payload["inventory_item_id"] != float64(333) {
				t.Fatalf("inventory_item_id = %#v, want 333", payload["inventory_item_id"])
			}
			if payload["location_id"] != float64(444) {
				t.Fatalf("location_id = %#v, want 444", payload["location_id"])
			}
			if payload["available"] != float64(7) {
				t.Fatalf("available = %#v, want 7", payload["available"])
			}
			sawSetLevel = true
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request = %s %s", r.Method, r.URL.String())
		}
	})
	defer closeServer()

	if err := provider.UpdateStock(context.Background(), "222", 7); err != nil {
		t.Fatalf("UpdateStock returned error: %v", err)
	}
	if !sawSetLevel {
		t.Fatal("expected inventory level update request")
	}
}

func TestUpdateStockSelectsInventoryLocationDeterministically(t *testing.T) {
	var sawSetLevel bool

	provider, closeServer := newShopifyProviderForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/variants/222.json":
			_, _ = w.Write([]byte(`{"variant":{"id":222,"product_id":111,"inventory_item_id":333}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/inventory_levels.json":
			if got := r.URL.Query().Get("inventory_item_ids"); got != "333" {
				t.Fatalf("inventory_item_ids = %q, want 333", got)
			}
			_, _ = w.Write([]byte(`{
				"inventory_levels": [
					{"inventory_item_id":333,"location_id":555,"available":3},
					{"inventory_item_id":333,"location_id":222,"available":4},
					{"inventory_item_id":333,"location_id":444,"available":5}
				]
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/inventory_levels/set.json":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode set level body: %v", err)
			}
			if payload["location_id"] != float64(222) {
				t.Fatalf("location_id = %#v, want deterministic lowest location 222", payload["location_id"])
			}
			if payload["available"] != float64(7) {
				t.Fatalf("available = %#v, want 7", payload["available"])
			}
			sawSetLevel = true
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request = %s %s", r.Method, r.URL.String())
		}
	})
	defer closeServer()

	if err := provider.UpdateStock(context.Background(), "222", 7); err != nil {
		t.Fatalf("UpdateStock returned error: %v", err)
	}
	if !sawSetLevel {
		t.Fatal("expected inventory level update request")
	}
}

func TestUpdateStockRejectsLegacySyntheticExternalID(t *testing.T) {
	provider := &Provider{}

	err := provider.UpdateStock(context.Background(), "shopify-"+uuid.NewString(), 7)
	if err == nil {
		t.Fatal("expected legacy synthetic external ID error")
	}
	if !strings.Contains(err.Error(), "legacy synthetic external ID") {
		t.Fatalf("error = %q, want legacy synthetic external ID context", err.Error())
	}
	if !strings.Contains(err.Error(), "recreate the Shopify listing") {
		t.Fatalf("error = %q, want recreate guidance", err.Error())
	}
}

func TestUpdatePriceRejectsLegacySyntheticExternalID(t *testing.T) {
	provider := &Provider{}

	err := provider.UpdatePrice(context.Background(), "shopify-"+uuid.NewString(), 12.34)
	if err == nil {
		t.Fatal("expected legacy synthetic external ID error")
	}
	if !strings.Contains(err.Error(), "legacy synthetic external ID") {
		t.Fatalf("error = %q, want legacy synthetic external ID context", err.Error())
	}
	if !strings.Contains(err.Error(), "recreate the Shopify listing") {
		t.Fatalf("error = %q, want recreate guidance", err.Error())
	}
}

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
