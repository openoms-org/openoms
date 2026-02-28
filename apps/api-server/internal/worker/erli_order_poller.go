package worker

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	erlisdk "github.com/openoms-org/openoms/packages/erli-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// NewErliOrderPoller creates a worker that polls orders from the Erli marketplace.
func NewErliOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, shipmentRepo repository.ShipmentRepo, auditRepo repository.AuditRepo, logger *slog.Logger) *MarketplaceOrderPoller {
	return NewMarketplaceOrderPoller(MarketplaceOrderPollerConfig{
		Pool:          pool,
		EncryptionKey: encryptionKey,
		OrderRepo:     orderRepo,
		ShipmentRepo:  shipmentRepo,
		AuditRepo:     auditRepo,
		Logger:        logger,
		ProviderName:  "erli",
		Interval:      45 * time.Second,
		MapOrder:      erliOrderMapper,
	})
}

func erliOrderMapper(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
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

	// Build metadata: always include external_id plus Erli-specific status fields.
	metadata := map[string]any{"external_id": mo.ExternalID}

	// Prefer pre-computed statuses from RawData (set by the provider's mapErliOrder).
	if mo.RawData != nil {
		if erliStatus, ok := mo.RawData["erli_status"].(string); ok {
			metadata["erli_status"] = erliStatus
		}
		if omsStatus, ok := mo.RawData["oms_status"].(string); ok {
			metadata["oms_status"] = omsStatus
		}
	}
	// Fallback: derive from ExternalStatus when RawData didn't supply erli_status.
	if _, hasErli := metadata["erli_status"]; !hasErli && mo.ExternalStatus != "" {
		if omsStatus, ok := erlisdk.MapStatus(mo.ExternalStatus); ok {
			metadata["erli_status"] = mo.ExternalStatus
			metadata["oms_status"] = omsStatus
		}
	}

	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON
	order.Tags = []string{}

	return order
}
