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

// NewAmazonOrderPoller creates a MarketplaceOrderPoller configured for the Amazon marketplace.
func NewAmazonOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "amazon",
		Interval:      2 * time.Minute,
		MapOrder:      amazonOrderMapper,
	})
}

func amazonOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
	order, metadata := newBaseMarketplaceOrder(mo, ti, req)

	// Amazon-specific: fulfillment channel from RawData
	if mo.RawData != nil {
		if dmName, ok := mo.RawData["fulfillment_channel"].(string); ok {
			order.DeliveryMethod = &dmName
		}
	}

	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON
	order.Tags = []string{}

	return order
}
