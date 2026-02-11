package worker

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

func NewWooCommerceOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		Logger:        logger,
		ProviderName:  "woocommerce",
		Interval:      60 * time.Second,
		MapOrder:      woocommerceOrderMapper,
	})
}

func woocommerceOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
	order := model.Order{
		ID:            uuid.New(),
		TenantID:      ti.TenantID,
		ExternalID:    req.ExternalID,
		Source:        req.Source,
		IntegrationID: req.IntegrationID,
		Status:        "new",
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		CustomerPhone: req.CustomerPhone,
		TotalAmount:   req.TotalAmount,
		Currency:      req.Currency,
		OrderedAt:     req.OrderedAt,
		PaymentMethod: req.PaymentMethod,
	}

	if req.PaymentStatus != nil {
		order.PaymentStatus = *req.PaymentStatus
	} else {
		order.PaymentStatus = "pending"
	}

	addrJSON, err := json.Marshal(mo.ShippingAddress)
	if err == nil {
		order.ShippingAddress = addrJSON
	}

	itemsJSON, err := json.Marshal(mo.Items)
	if err == nil {
		order.Items = itemsJSON
	}

	// WooCommerce-specific: customer note from RawData
	if mo.RawData != nil {
		if note, ok := mo.RawData["customer_note"].(string); ok && note != "" {
			order.Notes = &note
		}
	}

	metadata := map[string]any{"external_id": mo.ExternalID}
	if mo.RawData != nil {
		if wcID, ok := mo.RawData["woocommerce_order_id"]; ok {
			metadata["woocommerce_order_id"] = wcID
		}
	}
	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON
	order.Tags = []string{}

	return order
}
