package erli

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	erlisdk "github.com/openoms-org/openoms/packages/erli-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

func newErliProviderForTest(t *testing.T, handler http.HandlerFunc) (*Provider, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := erlisdk.NewClient("test-token",
		erlisdk.WithBaseURL(server.URL),
		erlisdk.WithHTTPClient(server.Client()),
	)

	return &Provider{client: client, logger: slog.Default()}, server.Close
}

// pushedErliStock runs PushOffer against a stub Erli API and returns the stock it sent.
func pushedErliStock(t *testing.T, product *model.Product, listingData map[string]any) int {
	t.Helper()

	var sent struct {
		Stock int `json:"stock"`
	}
	provider, closeServer := newErliProviderForTest(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ERLI-1"}`))
	})
	defer closeServer()

	if _, err := provider.PushOffer(context.Background(), product, listingData); err != nil {
		t.Fatalf("PushOffer returned error: %v", err)
	}
	return sent.Stock
}

func TestPushOfferUsesCanonicalAvailableStock(t *testing.T) {
	sku := "SKU-1"
	product := &model.Product{
		Name:           "Widget",
		SKU:            &sku,
		Price:          19.99,
		StockQuantity:  42, // legacy, not decremented on shipment
		AvailableStock: 7,  // canonical, warehouse-backed
	}

	assertStock(t, pushedErliStock(t, product, nil), 7)
}

func TestPushOfferListingDataStockWinsOverAvailableStock(t *testing.T) {
	sku := "SKU-1"
	product := &model.Product{
		Name:           "Widget",
		SKU:            &sku,
		Price:          19.99,
		StockQuantity:  42,
		AvailableStock: 7,
	}

	assertStock(t, pushedErliStock(t, product, map[string]any{"stock": 3}), 3)
}

func assertStock(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("pushed stock = %d, want %d", got, want)
	}
}
