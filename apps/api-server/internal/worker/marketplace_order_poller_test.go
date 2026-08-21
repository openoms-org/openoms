package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
	amazonsdk "github.com/openoms-org/openoms/packages/amazon-sp-sdk"
	ebaysdk "github.com/openoms-org/openoms/packages/ebay-go-sdk"
	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"
)

// ---------------------------------------------------------------------------
// MarketplaceOrderPoller — Name / Interval
// ---------------------------------------------------------------------------

func TestMarketplaceOrderPoller_Name(t *testing.T) {
	p := NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		ProviderName: "allegro",
		Interval:     45 * time.Second,
		Logger:       slog.Default(),
	})
	assert.Equal(t, "allegro_order_poller", p.Name())
}

func TestMarketplaceOrderPoller_NameDifferentProvider(t *testing.T) {
	p := NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		ProviderName: "amazon",
		Interval:     30 * time.Second,
		Logger:       slog.Default(),
	})
	assert.Equal(t, "amazon_order_poller", p.Name())
}

func TestMarketplaceOrderPoller_WithFulfillment(t *testing.T) {
	p := NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		ProviderName: "allegro",
		Interval:     45 * time.Second,
		Logger:       slog.Default(),
	})
	// Default: no fulfillment service wired (OPE-416 routing off).
	assert.Nil(t, p.fulfillment)

	// WithFulfillment wires the service and returns the poller for chaining. A nil
	// service is allowed (the gated service no-ops on its own); the wiring just records it.
	svc := service.NewFulfillmentService(false, nil, nil)
	got := p.WithFulfillment(svc)
	assert.Same(t, p, got, "WithFulfillment returns the poller for chaining")
	assert.Same(t, svc, p.fulfillment)
}

func TestMarketplaceOrderPoller_WithFulfillment_NilReceiverSafe(t *testing.T) {
	var p *MarketplaceOrderPoller // nil
	assert.NotPanics(t, func() {
		assert.Nil(t, p.WithFulfillment(service.NewFulfillmentService(false, nil, nil)))
	})
}

func TestMarketplaceOrderPoller_Interval(t *testing.T) {
	p := NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		ProviderName: "woocommerce",
		Interval:     2 * time.Minute,
		Logger:       slog.Default(),
	})
	assert.Equal(t, 2*time.Minute, p.Interval())
}

func TestMarketplaceOrderPoller_ImplementsWorkerInterface(_ *testing.T) {
	p := NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		ProviderName: "test",
		Interval:     time.Second,
		Logger:       slog.Default(),
	})
	var _ Worker = p
}

func TestIsTerminalOAuthCredentialError(t *testing.T) {
	assert.True(t, isTerminalOAuthCredentialError("olx",
		fmt.Errorf("poll: %w", olxsdk.ErrInvalidGrant)))
	assert.False(t, isTerminalOAuthCredentialError("olx",
		errors.New("temporary network timeout")))
	assert.False(t, isTerminalOAuthCredentialError("allegro",
		fmt.Errorf("poll: %w", olxsdk.ErrInvalidGrant)))

	assert.True(t, isTerminalOAuthCredentialError("allegro", allegrosdk.ErrUnauthorized))
	assert.True(t, isTerminalOAuthCredentialError("allegro", allegrosdk.ErrForbidden))
	assert.True(t, isTerminalOAuthCredentialError("allegro",
		fmt.Errorf("allegro: proactive token refresh failed: %w", allegrosdk.ErrReconnectRequired)))
	assert.True(t, isTerminalOAuthCredentialError("amazon",
		errors.New(`amazon: token request failed (HTTP 400): {"error":"invalid_grant"}`)))
	assert.True(t, isTerminalOAuthCredentialError("amazon",
		&amazonsdk.APIError{StatusCode: 403}))
	assert.True(t, isTerminalOAuthCredentialError("ebay", ebaysdk.ErrUnauthorized))
	assert.True(t, isTerminalOAuthCredentialError("ebay",
		errors.New(`ebay: token refresh failed (HTTP 400): {"error":"invalid_grant"}`)))

	assert.False(t, isTerminalOAuthCredentialError("allegro",
		errors.New("temporary network timeout")))
	assert.False(t, isTerminalOAuthCredentialError("amazon",
		errors.New("temporary network timeout")))
	assert.False(t, isTerminalOAuthCredentialError("ebay",
		errors.New("temporary network timeout")))
}

// ---------------------------------------------------------------------------
// resolveCarrier
// ---------------------------------------------------------------------------

func TestResolveCarrier_NilSettings(t *testing.T) {
	assert.Equal(t, "", resolveCarrier(nil, "InPost Paczkomaty"))
}

func TestResolveCarrier_EmptySettings(t *testing.T) {
	assert.Equal(t, "", resolveCarrier(json.RawMessage{}, "InPost"))
}

func TestResolveCarrier_AutoCreateDisabled(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": false,
		"default_carrier":      "inpost",
	})
	assert.Equal(t, "", resolveCarrier(settings, "anything"))
}

func TestResolveCarrier_DefaultCarrierNoMapping(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": true,
		"default_carrier":      "dhl",
	})
	assert.Equal(t, "dhl", resolveCarrier(settings, ""))
}

func TestResolveCarrier_DefaultCarrierWhenDeliveryMethodDoesNotMatch(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": true,
		"default_carrier":      "dpd",
		"carrier_mapping": map[string]string{
			"inpost":    "inpost",
			"paczkomat": "inpost",
		},
	})
	assert.Equal(t, "dpd", resolveCarrier(settings, "DHL Express"))
}

func TestResolveCarrier_CarrierMappingExactSubstringMatch(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": true,
		"default_carrier":      "dpd",
		"carrier_mapping": map[string]string{
			"inpost":    "inpost",
			"paczkomat": "inpost",
		},
	})
	assert.Equal(t, "inpost", resolveCarrier(settings, "InPost Paczkomaty 24/7"))
}

func TestResolveCarrier_CarrierMappingCaseInsensitive(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": true,
		"default_carrier":      "dpd",
		"carrier_mapping": map[string]string{
			"INPOST": "inpost",
		},
	})
	// Both key and delivery method are lowercased for comparison.
	assert.Equal(t, "inpost", resolveCarrier(settings, "inpost kurier"))
}

func TestResolveCarrier_EmptyDeliveryMethodUsesDefault(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": true,
		"default_carrier":      "gls",
		"carrier_mapping": map[string]string{
			"inpost": "inpost",
		},
	})
	assert.Equal(t, "gls", resolveCarrier(settings, ""))
}

func TestResolveCarrier_InvalidJSON(t *testing.T) {
	assert.Equal(t, "", resolveCarrier(json.RawMessage(`{invalid`), "test"))
}

func TestResolveCarrier_NoDefaultNoMapping(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": true,
	})
	assert.Equal(t, "", resolveCarrier(settings, "InPost"))
}

func TestResolveCarrier_MappingWithoutAutoCreate(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"auto_create_shipment": false,
		"carrier_mapping": map[string]string{
			"inpost": "inpost",
		},
	})
	assert.Equal(t, "", resolveCarrier(settings, "InPost"))
}

// ---------------------------------------------------------------------------
// shouldAutoLabel
// ---------------------------------------------------------------------------

func TestShouldAutoLabel_NilSettings(t *testing.T) {
	assert.False(t, shouldAutoLabel(nil))
}

func TestShouldAutoLabel_EmptySettings(t *testing.T) {
	assert.False(t, shouldAutoLabel(json.RawMessage{}))
}

func TestShouldAutoLabel_Enabled(t *testing.T) {
	settings := mustJSON(t, map[string]any{"auto_generate_label": true})
	assert.True(t, shouldAutoLabel(settings))
}

func TestShouldAutoLabel_Disabled(t *testing.T) {
	settings := mustJSON(t, map[string]any{"auto_generate_label": false})
	assert.False(t, shouldAutoLabel(settings))
}

func TestShouldAutoLabel_MissingKey(t *testing.T) {
	settings := mustJSON(t, map[string]any{"other_setting": "value"})
	assert.False(t, shouldAutoLabel(settings))
}

func TestShouldAutoLabel_InvalidJSON(t *testing.T) {
	assert.False(t, shouldAutoLabel(json.RawMessage(`not json`)))
}

// ---------------------------------------------------------------------------
// parseAutoLabelSettings
// ---------------------------------------------------------------------------

func TestParseAutoLabelSettings_NilSettings(t *testing.T) {
	s := parseAutoLabelSettings(nil)
	assert.Equal(t, "", s.DefaultParcelSize)
	assert.Equal(t, "", s.DefaultSendingMethod)
}

func TestParseAutoLabelSettings_EmptySettings(t *testing.T) {
	s := parseAutoLabelSettings(json.RawMessage{})
	assert.Equal(t, "", s.DefaultParcelSize)
	assert.Equal(t, "", s.DefaultSendingMethod)
}

func TestParseAutoLabelSettings_WithValues(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"default_parcel_size":    "medium",
		"default_sending_method": "pop",
	})
	s := parseAutoLabelSettings(settings)
	assert.Equal(t, "medium", s.DefaultParcelSize)
	assert.Equal(t, "pop", s.DefaultSendingMethod)
}

func TestParseAutoLabelSettings_PartialValues(t *testing.T) {
	settings := mustJSON(t, map[string]any{
		"default_parcel_size": "large",
	})
	s := parseAutoLabelSettings(settings)
	assert.Equal(t, "large", s.DefaultParcelSize)
	assert.Equal(t, "", s.DefaultSendingMethod)
}

func TestParseAutoLabelSettings_InvalidJSON(t *testing.T) {
	s := parseAutoLabelSettings(json.RawMessage(`{broken`))
	assert.Equal(t, "", s.DefaultParcelSize)
	assert.Equal(t, "", s.DefaultSendingMethod)
}

// ---------------------------------------------------------------------------
// buildOrder (via MarketplaceOrderPoller)
// ---------------------------------------------------------------------------

func TestBuildOrder_DefaultMapping(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	ti := TenantIntegration{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "allegro",
	}

	mo := integration.MarketplaceOrder{
		ExternalID:    "EXT-001",
		CustomerName:  "Jan Kowalski",
		CustomerEmail: "jan@example.com",
		CustomerPhone: "+48123456789",
		TotalAmount:   199.99,
		Currency:      "PLN",
		Items: []integration.MarketplaceOrderItem{
			{ExternalID: "item-1", Name: "Widget", Quantity: 2, UnitPrice: 99.995},
		},
		ShippingAddress: model.ShippingAddress{
			Name:       "Jan Kowalski",
			Street:     "ul. Testowa 1",
			City:       "Warszawa",
			PostalCode: "00-001",
			Country:    "PL",
		},
		PaymentStatus: "paid",
		PaymentMethod: "card",
		OrderedAt:     time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", integrationID)

	poller := &MarketplaceOrderPoller{
		providerName: "allegro",
		logger:       slog.Default(),
	}

	order := poller.buildOrder(mo, ti, req)

	assert.Equal(t, tenantID, order.TenantID)
	assert.NotEqual(t, uuid.Nil, order.ID)
	assert.Equal(t, "new", order.Status)
	assert.Equal(t, "Jan Kowalski", order.CustomerName)
	assert.Equal(t, "jan@example.com", *order.CustomerEmail)
	assert.Equal(t, "+48123456789", *order.CustomerPhone)
	assert.Equal(t, 199.99, order.TotalAmount)
	assert.Equal(t, "PLN", order.Currency)
	assert.Equal(t, "allegro", order.Source)
	assert.Equal(t, &integrationID, order.IntegrationID)
	assert.Equal(t, "paid", order.PaymentStatus)
	assert.Equal(t, "card", *order.PaymentMethod)
	assert.NotNil(t, order.OrderedAt)
	assert.NotNil(t, order.Tags, "Tags should be initialized (not nil)")
	assert.Empty(t, order.Tags, "Tags should be an empty slice")

	// Verify ExternalID in metadata
	var metadata map[string]any
	require.NoError(t, json.Unmarshal(order.Metadata, &metadata))
	assert.Equal(t, "EXT-001", metadata["external_id"])

	// Verify shipping address is valid JSON
	var addr model.ShippingAddress
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &addr))
	assert.Equal(t, "Jan Kowalski", addr.Name)
	assert.Equal(t, "Warszawa", addr.City)

	// Verify items is valid JSON
	var items []integration.MarketplaceOrderItem
	require.NoError(t, json.Unmarshal(order.Items, &items))
	require.Len(t, items, 1)
	assert.Equal(t, "Widget", items[0].Name)
}

func TestBuildOrder_DefaultPaymentStatusWhenNil(t *testing.T) {
	ti := TenantIntegration{
		TenantID:      uuid.New(),
		IntegrationID: uuid.New(),
		Provider:      "woocommerce",
	}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-002",
		CustomerName: "Anna Nowak",
		TotalAmount:  50.0,
		Currency:     "EUR",
		// PaymentStatus is empty string
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "woocommerce", ti.IntegrationID)
	// When PaymentStatus is empty, MarketplaceOrderToCreateRequest does NOT set it,
	// so req.PaymentStatus will be nil, and buildOrder should default to "pending".

	poller := &MarketplaceOrderPoller{
		providerName: "woocommerce",
		logger:       slog.Default(),
	}

	order := poller.buildOrder(mo, ti, req)
	assert.Equal(t, "pending", order.PaymentStatus)
}

func TestBuildOrder_WithDeliveryMethodFromRawData(t *testing.T) {
	ti := TenantIntegration{
		TenantID:      uuid.New(),
		IntegrationID: uuid.New(),
		Provider:      "allegro",
	}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-003",
		CustomerName: "Piotr Zielinski",
		TotalAmount:  30.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"delivery_method_name": "InPost Paczkomaty 24/7",
			"pickup_point_id":      "KRA04A",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	poller := &MarketplaceOrderPoller{
		providerName: "allegro",
		logger:       slog.Default(),
	}

	order := poller.buildOrder(mo, ti, req)
	require.NotNil(t, order.DeliveryMethod)
	assert.Equal(t, "InPost Paczkomaty 24/7", *order.DeliveryMethod)
	require.NotNil(t, order.PickupPointID)
	assert.Equal(t, "KRA04A", *order.PickupPointID)
}

func TestBuildOrder_NoRawData(t *testing.T) {
	ti := TenantIntegration{
		TenantID:      uuid.New(),
		IntegrationID: uuid.New(),
		Provider:      "amazon",
	}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-004",
		CustomerName: "Maria Kaminska",
		TotalAmount:  100.0,
		Currency:     "PLN",
		RawData:      nil,
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "amazon", ti.IntegrationID)
	poller := &MarketplaceOrderPoller{
		providerName: "amazon",
		logger:       slog.Default(),
	}

	order := poller.buildOrder(mo, ti, req)
	assert.Nil(t, order.DeliveryMethod)
	assert.Nil(t, order.PickupPointID)
}

func TestBuildOrder_RawDataWithoutDeliveryFields(t *testing.T) {
	ti := TenantIntegration{
		TenantID:      uuid.New(),
		IntegrationID: uuid.New(),
		Provider:      "allegro",
	}
	mo := integration.MarketplaceOrder{
		ExternalID:   "EXT-005",
		CustomerName: "Test User",
		TotalAmount:  10.0,
		Currency:     "PLN",
		RawData: map[string]any{
			"some_other_field": "value",
		},
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)
	poller := &MarketplaceOrderPoller{
		providerName: "allegro",
		logger:       slog.Default(),
	}

	order := poller.buildOrder(mo, ti, req)
	assert.Nil(t, order.DeliveryMethod)
	assert.Nil(t, order.PickupPointID)
}

func TestBuildOrder_CustomMapper(t *testing.T) {
	customOrder := model.Order{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Status:       "confirmed",
		CustomerName: "Custom Mapper",
	}

	poller := &MarketplaceOrderPoller{
		providerName: "custom",
		logger:       slog.Default(),
		mapOrder: func(_ integration.MarketplaceOrder, _ TenantIntegration, _ model.CreateOrderRequest) model.Order {
			return customOrder
		},
	}

	order := poller.buildOrder(integration.MarketplaceOrder{}, TenantIntegration{}, model.CreateOrderRequest{})
	assert.Equal(t, customOrder.ID, order.ID)
	assert.Equal(t, "confirmed", order.Status)
	assert.Equal(t, "Custom Mapper", order.CustomerName)
}

func TestBuildOrder_ExternalIDInRequestPassedThrough(t *testing.T) {
	ti := TenantIntegration{
		TenantID:      uuid.New(),
		IntegrationID: uuid.New(),
		Provider:      "ebay",
	}
	externalID := "EBAY-99999"
	mo := integration.MarketplaceOrder{
		ExternalID:   externalID,
		CustomerName: "Test",
		TotalAmount:  1.0,
		Currency:     "USD",
	}

	req := integration.MarketplaceOrderToCreateRequest(mo, "ebay", ti.IntegrationID)
	poller := &MarketplaceOrderPoller{
		providerName: "ebay",
		logger:       slog.Default(),
	}

	order := poller.buildOrder(mo, ti, req)
	require.NotNil(t, order.ExternalID)
	assert.Equal(t, externalID, *order.ExternalID)
}

// ---------------------------------------------------------------------------
// insertMarketplaceOrderIfNotExists
// ---------------------------------------------------------------------------

func TestInsertMarketplaceOrderIfNotExists_SkipsExistingOrder(t *testing.T) {
	repo := &marketplaceInsertOrderRepo{
		existing:        &model.Order{ID: uuid.New()},
		createIfCreated: true,
	}

	created, err := insertMarketplaceOrderIfNotExists(
		context.Background(), nil, repo, "allegro", "A-100", &model.Order{ID: uuid.New()},
	)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, 1, repo.findCalls)
	assert.Equal(t, 0, repo.createIfCalls)
}

func TestInsertMarketplaceOrderIfNotExists_SkipsConcurrentDuplicate(t *testing.T) {
	repo := &marketplaceInsertOrderRepo{createIfCreated: false}

	created, err := insertMarketplaceOrderIfNotExists(
		context.Background(), nil, repo, "allegro", "A-100", &model.Order{ID: uuid.New()},
	)

	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, 1, repo.findCalls)
	assert.Equal(t, 1, repo.createIfCalls)
}

func TestInsertMarketplaceOrderIfNotExists_CreatesNewOrder(t *testing.T) {
	order := &model.Order{ID: uuid.New()}
	repo := &marketplaceInsertOrderRepo{createIfCreated: true}

	created, err := insertMarketplaceOrderIfNotExists(
		context.Background(), nil, repo, "amazon", "AMZ-1", order,
	)

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, 1, repo.findCalls)
	assert.Equal(t, 1, repo.createIfCalls)
	assert.Equal(t, order, repo.createdOrder)
}

type marketplaceInsertOrderRepo struct {
	repository.OrderRepo
	existing        *model.Order
	createIfCreated bool
	createdOrder    *model.Order
	findCalls       int
	createIfCalls   int
}

func (r *marketplaceInsertOrderRepo) FindByExternalID(_ context.Context, _ pgx.Tx, _, _ string) (*model.Order, error) {
	r.findCalls++
	return r.existing, nil
}

func (r *marketplaceInsertOrderRepo) CreateIfExternalIDNotExists(_ context.Context, _ pgx.Tx, order *model.Order) (bool, error) {
	r.createIfCalls++
	if !r.createIfCreated {
		return false, nil
	}
	r.createdOrder = order
	return true, nil
}

// ---------------------------------------------------------------------------
// NewMarketplaceOrderPoller
// ---------------------------------------------------------------------------

func TestNewMarketplaceOrderPoller_SetsAllFields(t *testing.T) {
	logger := slog.Default()
	key := []byte("encryption-key")
	mapper := func(_ integration.MarketplaceOrder, _ TenantIntegration, _ model.CreateOrderRequest) model.Order {
		return model.Order{}
	}

	p := NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		EncryptionKey: key,
		Logger:        logger,
		ProviderName:  "test-provider",
		Interval:      99 * time.Second,
		MapOrder:      mapper,
	})

	assert.Equal(t, "test-provider", p.providerName)
	assert.Equal(t, 99*time.Second, p.interval)
	assert.Equal(t, key, p.encryptionKey)
	assert.Equal(t, logger, p.logger)
	assert.NotNil(t, p.mapOrder)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}
