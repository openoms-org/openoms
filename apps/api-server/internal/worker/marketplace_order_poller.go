package worker

import (
	"context"
	"encoding/json"
	"log/slog"
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
			if err := database.WithTenant(ctx, p.pool, ti.TenantID, func(tx pgx.Tx) error {
				existing, err := p.orderRepo.FindByExternalID(ctx, tx, p.providerName, mo.ExternalID)
				if err != nil {
					return err
				}
				if existing != nil {
					return nil // duplicate, skip
				}

				req := integration.MarketplaceOrderToCreateRequest(mo, p.providerName, ti.IntegrationID)

				order := p.buildOrder(mo, ti, req)

				return p.orderRepo.Create(ctx, tx, &order)
			}); err != nil {
				p.logger.Error("failed to create order", "integration_id", ti.IntegrationID, "external_id", mo.ExternalID, "error", err)
				continue
			}
			totalOrders++
		}

		// Update sync cursor (bypasses RLS)
		if newCursor != cursor {
			if _, err := p.pool.Exec(ctx,
				"UPDATE integrations SET sync_cursor = $1, last_sync_at = NOW() WHERE id = $2",
				newCursor, ti.IntegrationID,
			); err != nil {
				p.logger.Error("failed to update sync cursor", "integration_id", ti.IntegrationID, "error", err)
			}
		}
	}

	p.logger.Info(p.providerName+" order poller completed", "tenants", len(tis), "orders", totalOrders)
	return nil
}

func (p *MarketplaceOrderPoller) buildOrder(mo integration.MarketplaceOrder, ti TenantIntegration, req model.CreateOrderRequest) model.Order {
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

	// Allow provider-specific customization
	if p.mapOrder != nil {
		order = p.mapOrder(mo, ti, req)
	} else {
		// Default: extract delivery method from RawData
		if mo.RawData != nil {
			if dmName, ok := mo.RawData["delivery_method_name"].(string); ok {
				order.DeliveryMethod = &dmName
			}
			if ppID, ok := mo.RawData["pickup_point_id"].(string); ok {
				order.PickupPointID = &ppID
			}
		}
	}

	return order
}
