package worker

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// NewWooCommerceOrderPoller creates a MarketplaceOrderPoller configured for WooCommerce.
func NewWooCommerceOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "woocommerce",
		Interval:      60 * time.Second,
		MapOrder:      woocommerceOrderMapper,
	})
}

func woocommerceOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
	order, metadata := newBaseMarketplaceOrder(mo, ti, req)

	// WooCommerce-specific: customer note from RawData
	if mo.RawData != nil {
		if note, ok := mo.RawData["customer_note"].(string); ok && note != "" {
			order.Notes = &note
		}
	}

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
