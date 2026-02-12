package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// OrderMapper converts a MarketplaceOrder into a model.Order for a specific provider.
// The default implementation is used if nil.
type OrderMapper func(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order

// MarketplaceOrderPoller is a generic order poller for any marketplace provider.
type MarketplaceOrderPoller struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	orderRepo     repository.OrderRepo
	shipmentRepo  repository.ShipmentRepo
	auditRepo     repository.AuditRepo
	logger        *slog.Logger
	providerName  string
	interval      time.Duration
	// mapOrder allows provider-specific customization of the order mapping.
	// If nil, a default mapping is used.
	mapOrder OrderMapper
}

// MarketplaceOrderPollerConfig configures a MarketplaceOrderPoller.
type MarketplaceOrderPollerConfig struct {
	Pool          *pgxpool.Pool
	EncryptionKey []byte
	OrderRepo     repository.OrderRepo
	ShipmentRepo  repository.ShipmentRepo
	AuditRepo     repository.AuditRepo
	Logger        *slog.Logger
	ProviderName  string
	Interval      time.Duration
	MapOrder      OrderMapper
}

func NewMarketplaceOrderPoller(cfg MarketplaceOrderPollerConfig) *MarketplaceOrderPoller {
	return &MarketplaceOrderPoller{
		pool:          cfg.Pool,
		encryptionKey: cfg.EncryptionKey,
		orderRepo:     cfg.OrderRepo,
		shipmentRepo:  cfg.ShipmentRepo,
		auditRepo:     cfg.AuditRepo,
		logger:        cfg.Logger,
		providerName:  cfg.ProviderName,
		interval:      cfg.Interval,
		mapOrder:      cfg.MapOrder,
	}
}

func (p *MarketplaceOrderPoller) Name() string {
	return p.providerName + "_order_poller"
}

func (p *MarketplaceOrderPoller) Interval() time.Duration {
	return p.interval
}

func (p *MarketplaceOrderPoller) Run(ctx context.Context) error {
	tis, err := ListActiveIntegrations(ctx, p.pool, p.providerName)
	if err != nil {
		return err
	}

	totalOrders := 0

	for _, ti := range tis {
		credJSON, err := crypto.Decrypt(ti.Credentials, p.encryptionKey)
		if err != nil {
			p.logger.Error("failed to decrypt credentials", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		provider, err := integration.NewMarketplaceProvider(p.providerName, credJSON, ti.Settings)
		if err != nil {
			p.logger.Error("failed to create provider", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		cursor := ""
		if ti.SyncCursor != nil {
			cursor = *ti.SyncCursor
		}

		orders, newCursor, err := provider.PollOrders(ctx, cursor)
		if err != nil {
			p.logger.Error("failed to poll orders", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		for _, mo := range orders {
			req := integration.MarketplaceOrderToCreateRequest(mo, p.providerName, ti.IntegrationID)
			order := p.buildOrder(mo, ti, req)

			if err := database.WithTenant(ctx, p.pool, ti.TenantID, func(tx pgx.Tx) error {
				existing, err := p.orderRepo.FindByExternalID(ctx, tx, p.providerName, mo.ExternalID)
				if err != nil {
					return err
				}
				if existing != nil {
					return nil // duplicate, skip
				}

				if err := p.orderRepo.Create(ctx, tx, &order); err != nil {
					return err
				}

				p.logger.Info("worker: order created",
					"operation", "order.create",
					"tenant_id", ti.TenantID,
					"entity_id", order.ID,
					"external_id", mo.ExternalID,
					"provider", p.providerName,
					"integration_id", ti.IntegrationID,
				)
				return nil
			}); err != nil {
				p.logger.Error("failed to create order", "integration_id", ti.IntegrationID, "external_id", mo.ExternalID, "error", err)
				continue
			}
			totalOrders++

			// Auto-create shipment based on integration carrier mapping (best effort)
			if p.shipmentRepo != nil {
				dm := ""
				if order.DeliveryMethod != nil {
					dm = *order.DeliveryMethod
				}
				carrier := resolveCarrier(ti.Settings, dm)
				if carrier != "" {
					if err := p.autoCreateShipment(ctx, ti, order, carrier); err != nil {
						p.logger.Error("auto-create shipment failed (non-fatal)",
							"order_id", order.ID,
							"carrier", carrier,
							"error", err,
						)
					} else {
						p.logger.Info("auto-created shipment for marketplace order",
							"order_id", order.ID,
							"carrier", carrier,
						)
					}
				}
			}
		}

		// Update sync cursor (bypasses RLS)
		if newCursor != cursor {
			if _, err := p.pool.Exec(ctx,
				"UPDATE integrations SET sync_cursor = $1, last_sync_at = NOW() WHERE id = $2",
				newCursor, ti.IntegrationID,
			); err != nil {
				p.logger.Error("failed to update sync cursor",
					"operation", "integration.update_cursor",
					"tenant_id", ti.TenantID,
					"entity_id", ti.IntegrationID,
					"error", err,
				)
			} else {
				p.logger.Info("worker: sync cursor updated",
					"operation", "integration.update_cursor",
					"tenant_id", ti.TenantID,
					"entity_id", ti.IntegrationID,
					"new_cursor", newCursor,
				)
			}
		}
	}

	p.logger.Info(p.providerName+" order poller completed", "tenants", len(tis), "orders", totalOrders)
	return nil
}

func (p *MarketplaceOrderPoller) buildOrder(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
	if p.mapOrder != nil {
		return p.mapOrder(mo, ti, req)
	}

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

	// Shipping address
	addrJSON, err := json.Marshal(mo.ShippingAddress)
	if err == nil {
		order.ShippingAddress = addrJSON
	}

	// Items
	itemsJSON, err := json.Marshal(mo.Items)
	if err == nil {
		order.Items = itemsJSON
	}

	// Metadata with external_id for duplicate detection
	metadata := map[string]any{"external_id": mo.ExternalID}
	metadataJSON, _ := json.Marshal(metadata)
	order.Metadata = metadataJSON

	order.Tags = []string{}

	// Default: extract delivery method from RawData
	if mo.RawData != nil {
		if dmName, ok := mo.RawData["delivery_method_name"].(string); ok {
			order.DeliveryMethod = &dmName
		}
		if ppID, ok := mo.RawData["pickup_point_id"].(string); ok {
			order.PickupPointID = &ppID
		}
	}

	return order
}

// resolveCarrier looks up the delivery method in carrier_mapping, falling back to default_carrier.
func resolveCarrier(settings json.RawMessage, deliveryMethod string) string {
	if len(settings) == 0 {
		return ""
	}
	var s struct {
		AutoCreateShipment bool              `json:"auto_create_shipment"`
		DefaultCarrier     string            `json:"default_carrier"`
		CarrierMapping     map[string]string `json:"carrier_mapping"`
	}
	if err := json.Unmarshal(settings, &s); err != nil || !s.AutoCreateShipment {
		return ""
	}
	// Substring matching in carrier_mapping
	if deliveryMethod != "" && len(s.CarrierMapping) > 0 {
		deliveryLower := strings.ToLower(deliveryMethod)
		for key, provider := range s.CarrierMapping {
			if strings.Contains(deliveryLower, strings.ToLower(key)) {
				return provider
			}
		}
	}
	return s.DefaultCarrier
}

func (p *MarketplaceOrderPoller) autoCreateShipment(ctx context.Context, ti TenantIntegration, order model.Order, carrier string) error {
	shipment := &model.Shipment{
		ID:       uuid.New(),
		TenantID: ti.TenantID,
		OrderID:  order.ID,
		Provider: carrier,
		Status:   "created",
	}
	carrierData := map[string]any{}
	if order.PickupPointID != nil && *order.PickupPointID != "" {
		carrierData["target_point"] = *order.PickupPointID
	}
	if len(carrierData) > 0 {
		cd, _ := json.Marshal(carrierData)
		shipment.CarrierData = cd
	} else {
		shipment.CarrierData = json.RawMessage("{}")
	}

	return database.WithTenant(ctx, p.pool, ti.TenantID, func(tx pgx.Tx) error {
		if err := p.shipmentRepo.Create(ctx, tx, shipment); err != nil {
			return err
		}
		if p.auditRepo != nil {
			return p.auditRepo.Log(ctx, tx, model.AuditEntry{
				TenantID:   ti.TenantID,
				UserID:     uuid.Nil,
				Action:     "shipment.created",
				EntityType: "shipment",
				EntityID:   shipment.ID,
				Changes:    map[string]string{"order_id": order.ID.String(), "provider": carrier, "auto": "true"},
				IPAddress:  "worker",
			})
		}
		return nil
	})
}
