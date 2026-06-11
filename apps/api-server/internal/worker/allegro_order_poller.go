// Package worker contains background worker implementations for polling and scheduled tasks.
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

func allegroOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
	order, metadata := newBaseMarketplaceOrder(mo, ti, req)

	// Fallback: if buyer phone is missing, use shipping address phone
	if (order.CustomerPhone == nil || *order.CustomerPhone == "") && mo.ShippingAddress.Phone != "" {
		order.CustomerPhone = &mo.ShippingAddress.Phone
	}

	// Allegro-specific: delivery method and pickup point from RawData
	if mo.RawData != nil {
		if dmName, ok := mo.RawData["delivery_method_name"].(string); ok {
			order.DeliveryMethod = &dmName
		}
		if ppID, ok := mo.RawData["pickup_point_id"].(string); ok {
			order.PickupPointID = &ppID
		}
	}

	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON
	order.Tags = []string{}

	return order
}
