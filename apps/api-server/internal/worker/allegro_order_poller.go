// Package worker contains background worker implementations for polling and scheduled tasks.
package worker

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	allegroint "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// NewAllegroOrderPoller creates a MarketplaceOrderPoller configured for the Allegro marketplace.
func NewAllegroOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, labelGen LabelGenerator, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:           pool,
		EncryptionKey:  encryptionKey,
		OrderRepo:      orderRepo,
		ShipmentRepo:   shipmentRepo,
		AuditRepo:      auditRepo,
		LabelGenerator: labelGen,
		Logger:         logger,
		ProviderName:   "allegro",
		Interval:       45 * time.Second,
		MapOrder:       allegroOrderMapper,
	})
}

func allegroOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, _ model.CreateOrderRequest) model.Order {
	return allegroint.ToOMSOrder(mo, ti.TenantID, ti.IntegrationID)
}
