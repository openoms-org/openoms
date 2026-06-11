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

// NewEbayOrderPoller creates an eBay order poller with a 90s interval.
func NewEbayOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "ebay",
		Interval:      90 * time.Second,
		MapOrder:      ebayOrderMapper,
	})
}

func ebayOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
	order, metadata := newBaseMarketplaceOrder(mo, ti, req)

	// Fallback: if buyer phone is missing, use shipping address phone
	if (order.CustomerPhone == nil || *order.CustomerPhone == "") && mo.ShippingAddress.Phone != "" {
		order.CustomerPhone = &mo.ShippingAddress.Phone
	}

	// eBay-specific: store legacy order ID from RawData in metadata
	if mo.RawData != nil {
		if legacyID, ok := mo.RawData["ebay_legacy_order_id"].(string); ok {
			metadata["ebay_legacy_order_id"] = legacyID
		}
	}
	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON
	order.Tags = []string{}

	return order
}
