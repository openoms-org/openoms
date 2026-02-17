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

// NewShoperOrderPoller creates a background worker that polls Shoper for new orders.
func NewShoperOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "shoper",
		Interval:      60 * time.Second,
		MapOrder:      storeOrderMapper("shoper"),
	})
}

// NewPrestaShopOrderPoller creates a background worker that polls PrestaShop for new orders.
func NewPrestaShopOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "prestashop",
		Interval:      90 * time.Second,
		MapOrder:      storeOrderMapper("prestashop"),
	})
}

// NewShopifyOrderPoller creates a background worker that polls Shopify for new orders.
func NewShopifyOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "shopify",
		Interval:      60 * time.Second,
		MapOrder:      storeOrderMapper("shopify"),
	})
}

// storeOrderMapper returns an OrderMapper function for a store platform provider.
// All three store platforms (Shoper, PrestaShop, Shopify) share the same mapping logic.
func storeOrderMapper(providerName string) OrderMapper {
	return func(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
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

		// Store-specific: customer note from RawData
		if mo.RawData != nil {
			if note, ok := mo.RawData["customer_note"].(string); ok && note != "" {
				order.Notes = &note
			}
		}

		metadata := map[string]any{
			"external_id": mo.ExternalID,
			providerName + "_order_id": mo.RawData[providerName+"_order_id"],
		}
		metadataJSON, _ := json.Marshal(metadata)
		order.Metadata = metadataJSON
		order.Tags = []string{}

		return order
	}
}
