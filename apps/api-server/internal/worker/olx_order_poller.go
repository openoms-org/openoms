package worker

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	olxsdk "github.com/openoms-org/openoms/packages/olx-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// NewOLXOrderPoller creates a worker that polls orders (transactions) from the OLX marketplace.
func NewOLXOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "olx",
		Interval:      45 * time.Second,
		MapOrder:      olxOrderMapper,
	})
}

func olxOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
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

	// OLX-specific metadata from RawData
	metadata := map[string]any{"external_id": mo.ExternalID}
	if mo.RawData != nil {
		if v, ok := mo.RawData["olx_status"].(string); ok {
			metadata["olx_status"] = v
		}
		if v, ok := mo.RawData["oms_status"].(string); ok {
			metadata["oms_status"] = v
		}
		if v, ok := mo.RawData["olx_transaction_id"].(string); ok {
			metadata["olx_transaction_id"] = v
		}
		if v, ok := mo.RawData["olx_advert_id"]; ok {
			metadata["olx_advert_id"] = v
		}
	}
	// Fallback: derive oms_status from ExternalStatus if not in RawData
	if _, has := metadata["oms_status"]; !has && mo.ExternalStatus != "" {
		if omsStatus, ok := olxsdk.MapTransactionStatus(mo.ExternalStatus); ok {
			metadata["olx_status"] = mo.ExternalStatus
			metadata["oms_status"] = omsStatus
		}
	}

	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON
	order.Tags = []string{}

	return order
}
