package olx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"
	"github.com/stretchr/testify/assert"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// newTestProvider creates a Provider backed by the given httptest server URL.
func newTestProvider(t *testing.T, srvURL string) *Provider {
	t.Helper()
	client := olxsdk.NewClient("test_id", "test_secret", "",
		olxsdk.WithBaseURL(srvURL),
		olxsdk.WithAccessToken("test_token"),
	)
	return &Provider{
		client: client,
		logger: slog.Default().With("provider", "olx-test"),
	}
}

// ---------------------------------------------------------------------------
// TestNewProvider — valid credentials
// ---------------------------------------------------------------------------

func TestNewProvider(t *testing.T) {
	creds := `{"client_id":"my_id","client_secret":"my_secret","access_token":"tok"}` //nolint:gosec // test credentials
	p, err := NewProvider(json.RawMessage(creds), nil)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "olx", p.ProviderName())
}

// ---------------------------------------------------------------------------
// TestNewProviderMissingClientID
// ---------------------------------------------------------------------------

func TestNewProviderMissingClientID(t *testing.T) {
	creds := `{"client_secret":"x"}` //nolint:gosec // test credentials
	_, err := NewProvider(json.RawMessage(creds), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client_id")
}

// ---------------------------------------------------------------------------
// TestNewProviderMissingSecret
// ---------------------------------------------------------------------------

func TestNewProviderMissingSecret(t *testing.T) {
	creds := `{"client_id":"x"}` //nolint:gosec // test credentials
	_, err := NewProvider(json.RawMessage(creds), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client_secret")
}

// ---------------------------------------------------------------------------
// TestPollOrders — single transaction returned
// ---------------------------------------------------------------------------

func TestPollOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/transactions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"data": [
				{
					"id": "tx-100",
					"advert_id": 55555,
					"status": "completed",
					"amount": 149.99,
					"currency": "PLN",
					"created_at": "2026-03-05T10:00:00Z",
					"buyer_name": "Jan Kowalski",
					"buyer_email": "jan@test.pl",
					"buyer_phone": "500100200",
					"advert_title": "Test Product",
					"quantity": 1,
					"shipping_address": {
						"name": "Jan Kowalski",
						"street": "Marszalkowska 1",
						"city": "Warszawa",
						"postal_code": "00-001",
						"country": "PL"
					}
				}
			],
			"links": {"self": "/transactions"}
		}`)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	orders, cursor, err := p.PollOrders(context.Background(), "")

	assert.NoError(t, err)
	assert.Len(t, orders, 1)

	o := orders[0]
	assert.Equal(t, "tx-100", o.ExternalID)
	assert.Equal(t, "completed", o.ExternalStatus)
	assert.Equal(t, "Jan Kowalski", o.CustomerName)
	assert.Equal(t, "jan@test.pl", o.CustomerEmail)
	assert.Equal(t, "500100200", o.CustomerPhone)
	assert.Equal(t, 149.99, o.TotalAmount)
	assert.Equal(t, "PLN", o.Currency)
	assert.Equal(t, "paid", o.PaymentStatus)

	// Shipping address
	assert.Equal(t, "Jan Kowalski", o.ShippingAddress.Name)
	assert.Equal(t, "Marszalkowska 1", o.ShippingAddress.Street)
	assert.Equal(t, "Warszawa", o.ShippingAddress.City)
	assert.Equal(t, "00-001", o.ShippingAddress.PostalCode)
	assert.Equal(t, "PL", o.ShippingAddress.Country)

	// Items
	assert.Len(t, o.Items, 1)
	assert.Equal(t, "55555", o.Items[0].ExternalID)
	assert.Equal(t, "Test Product", o.Items[0].Name)
	assert.Equal(t, 1, o.Items[0].Quantity)
	assert.Equal(t, 149.99, o.Items[0].UnitPrice)
	assert.Equal(t, 149.99, o.Items[0].TotalPrice)

	// Cursor updated
	assert.Equal(t, "2026-03-05T10:00:00Z", cursor)
}

// ---------------------------------------------------------------------------
// TestPollOrdersEmpty — no transactions
// ---------------------------------------------------------------------------

func TestPollOrdersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[],"links":{"self":"/transactions"}}`)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	orders, cursor, err := p.PollOrders(context.Background(), "old-cursor")

	assert.NoError(t, err)
	assert.Nil(t, orders)
	assert.Equal(t, "old-cursor", cursor)
}

// ---------------------------------------------------------------------------
// TestGetOrder — find matching transaction
// ---------------------------------------------------------------------------

func TestGetOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"data": [
				{
					"id": "tx-200",
					"advert_id": 11111,
					"status": "pending",
					"amount": 50.0,
					"currency": "PLN",
					"created_at": "2026-03-04T08:00:00Z",
					"buyer_name": "Anna Nowak",
					"buyer_email": "anna@test.pl",
					"advert_title": "Another Product",
					"quantity": 2
				},
				{
					"id": "tx-201",
					"advert_id": 22222,
					"status": "completed",
					"amount": 99.99,
					"currency": "PLN",
					"created_at": "2026-03-04T09:00:00Z",
					"buyer_name": "Piotr Wiśniewski",
					"buyer_email": "piotr@test.pl",
					"advert_title": "Target Product",
					"quantity": 1
				}
			],
			"links": {"self": "/transactions"}
		}`)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	order, err := p.GetOrder(context.Background(), "tx-201")

	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "tx-201", order.ExternalID)
	assert.Equal(t, "Piotr Wiśniewski", order.CustomerName)
	assert.Equal(t, "piotr@test.pl", order.CustomerEmail)
	assert.Equal(t, 99.99, order.TotalAmount)
}

// ---------------------------------------------------------------------------
// TestGetOrderNotFound — target ID not in response
// ---------------------------------------------------------------------------

func TestGetOrderNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"data": [
				{
					"id": "tx-300",
					"advert_id": 33333,
					"status": "completed",
					"amount": 10.0,
					"currency": "PLN",
					"created_at": "2026-03-01T12:00:00Z",
					"buyer_name": "Test User",
					"buyer_email": "test@test.pl",
					"advert_title": "Some Product",
					"quantity": 1
				}
			],
			"links": {"self": "/transactions"}
		}`)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	order, err := p.GetOrder(context.Background(), "tx-999")

	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// TestPushOffer — successful advert creation
// ---------------------------------------------------------------------------

func TestPushOffer(t *testing.T) {
	var receivedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/adverts", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"id":99999,"title":"Test Product","status":"new"}}`)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	sku := "SKU-001"
	product := &model.Product{
		Name:            "Test Product",
		DescriptionLong: "A great product description that is long enough",
		Price:           149.99,
		SKU:             &sku,
	}

	listingData := map[string]any{
		"category_id":   float64(100),
		"city_id":       float64(9000),
		"contact_name":  "Seller Name",
		"contact_phone": "600100200",
	}

	externalID, err := p.PushOffer(context.Background(), product, listingData)

	assert.NoError(t, err)
	assert.Equal(t, "99999", externalID)

	// Verify request body
	assert.Equal(t, "Test Product", receivedBody["title"])
	assert.Equal(t, "A great product description that is long enough", receivedBody["description"])
	assert.Equal(t, float64(100), receivedBody["category_id"])
	assert.Equal(t, "business", receivedBody["advertiser_type"])
	assert.Equal(t, "SKU-001", receivedBody["external_id"])

	price, ok := receivedBody["price"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, 149.99, price["value"])
	assert.Equal(t, "PLN", price["currency"])

	contact, ok := receivedBody["contact"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Seller Name", contact["name"])
	assert.Equal(t, "600100200", contact["phone"])

	location, ok := receivedBody["location"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, float64(9000), location["city_id"])
}

// ---------------------------------------------------------------------------
// TestPushOfferMissingCategory — error when no category_id
// ---------------------------------------------------------------------------

func TestPushOfferMissingCategory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach server when validation fails")
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	product := &model.Product{
		Name:            "Test Product",
		DescriptionLong: "Description",
		Price:           10.0,
	}

	_, err := p.PushOffer(context.Background(), product, map[string]any{
		"city_id":      float64(9000),
		"contact_name": "Seller",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category_id")
}

// ---------------------------------------------------------------------------
// TestPushOfferMissingCity — error when no city_id
// ---------------------------------------------------------------------------

func TestPushOfferMissingCity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("should not reach server when validation fails")
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	product := &model.Product{
		Name:            "Test Product",
		DescriptionLong: "Description",
		Price:           10.0,
	}

	_, err := p.PushOffer(context.Background(), product, map[string]any{
		"category_id":  float64(100),
		"contact_name": "Seller",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "city_id")
}

// ---------------------------------------------------------------------------
// TestUpdatePrice — fetch then update
// ---------------------------------------------------------------------------

func TestUpdatePrice(t *testing.T) {
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/adverts/12345":
			fmt.Fprint(w, `{
				"data": {
					"id": 12345,
					"title": "Existing Product",
					"description": "Desc",
					"status": "active",
					"price": {"value": 100.0, "currency": "PLN", "negotiable": false},
					"category": {"id": 200, "name": "Electronics"},
					"contact": {"name": "Seller"}
				}
			}`)

		case r.Method == http.MethodPut && r.URL.Path == "/adverts/12345":
			body, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(body, &req)

			price, ok := req["price"].(map[string]any)
			assert.True(t, ok, "PUT body must include price")
			assert.Equal(t, 79.99, price["value"])
			assert.Equal(t, "PLN", price["currency"])

			// Verify the rest of the advert fields are preserved
			assert.Equal(t, "Existing Product", req["title"])
			assert.Equal(t, float64(200), req["category_id"])

			fmt.Fprint(w, `{
				"data": {
					"id": 12345,
					"title": "Existing Product",
					"description": "Desc",
					"status": "active",
					"price": {"value": 79.99, "currency": "PLN", "negotiable": false}
				}
			}`)

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	err := p.UpdatePrice(context.Background(), "12345", 79.99)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount, "should make GET then PUT")
}

// ---------------------------------------------------------------------------
// TestActivateOffer — sends activate command
// ---------------------------------------------------------------------------

func TestActivateOffer(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		assert.Equal(t, http.MethodPost, r.Method)

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	err := p.ActivateOffer(context.Background(), "77777")

	assert.NoError(t, err)
	assert.Equal(t, "/adverts/77777/commands", receivedPath)
	assert.Equal(t, "activate", receivedBody["command"])
}

// ---------------------------------------------------------------------------
// TestDeactivateOffer — sends deactivate command
// ---------------------------------------------------------------------------

func TestDeactivateOffer(t *testing.T) {
	var receivedBody map[string]any
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		assert.Equal(t, http.MethodPost, r.Method)

		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	err := p.DeactivateOffer(context.Background(), "88888")

	assert.NoError(t, err)
	assert.Equal(t, "/adverts/88888/commands", receivedPath)
	assert.Equal(t, "deactivate", receivedBody["command"])
}
