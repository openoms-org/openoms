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

type AllegroOrderPoller struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	orderRepo     repository.OrderRepo
	logger        *slog.Logger
}

func NewAllegroOrderPoller(pool *pgxpool.Pool, encryptionKey []byte, orderRepo repository.OrderRepo, logger *slog.Logger) *AllegroOrderPoller {
	return &AllegroOrderPoller{
		pool:          pool,
		encryptionKey: encryptionKey,
		orderRepo:     orderRepo,
		logger:        logger,
	}
}

func (a *AllegroOrderPoller) Name() string {
	return "allegro_order_poller"
}

func (a *AllegroOrderPoller) Interval() time.Duration {
	return 45 * time.Second
}

func (a *AllegroOrderPoller) Run(ctx context.Context) error {
	tis, err := ListActiveIntegrations(ctx, a.pool, "allegro")
	if err != nil {
		return err
	}

	totalOrders := 0

	for _, ti := range tis {
		credJSON, err := crypto.Decrypt(ti.Credentials, a.encryptionKey)
		if err != nil {
			a.logger.Error("failed to decrypt credentials", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		provider, err := integration.NewMarketplaceProvider("allegro", credJSON, ti.Settings)
		if err != nil {
			a.logger.Error("failed to create provider", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		cursor := ""
		if ti.SyncCursor != nil {
			cursor = *ti.SyncCursor
		}

		orders, newCursor, err := provider.PollOrders(ctx, cursor)
		if err != nil {
			a.logger.Error("failed to poll orders", "integration_id", ti.IntegrationID, "error", err)
			continue
		}

		for _, mo := range orders {
			if err := database.WithTenant(ctx, a.pool, ti.TenantID, func(tx pgx.Tx) error {
				existing, err := a.orderRepo.FindByExternalID(ctx, tx, "allegro", mo.ExternalID)
				if err != nil {
					return err
				}
				if existing != nil {
					return nil // duplicate, skip
				}

				req := integration.MarketplaceOrderToCreateRequest(mo, "allegro", ti.IntegrationID)

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

				// Delivery method and pickup point from RawData
				if mo.RawData != nil {
					if dmName, ok := mo.RawData["delivery_method_name"].(string); ok {
						order.DeliveryMethod = &dmName
					}
					if ppID, ok := mo.RawData["pickup_point_id"].(string); ok {
						order.PickupPointID = &ppID
					}
				}

				// Metadata with external_id for duplicate detection
				metadata := map[string]any{"external_id": mo.ExternalID}
				metadataJSON, _ := json.Marshal(metadata)
				order.Metadata = metadataJSON

				order.Tags = []string{}

				return a.orderRepo.Create(ctx, tx, &order)
			}); err != nil {
				a.logger.Error("failed to create order", "integration_id", ti.IntegrationID, "external_id", mo.ExternalID, "error", err)
				continue
			}
			totalOrders++
		}

		// Update sync cursor (bypasses RLS)
		if newCursor != cursor {
			if _, err := a.pool.Exec(ctx,
				"UPDATE integrations SET sync_cursor = $1, last_sync_at = NOW() WHERE id = $2",
				newCursor, ti.IntegrationID,
			); err != nil {
				a.logger.Error("failed to update sync cursor", "integration_id", ti.IntegrationID, "error", err)
			}
		}
	}

	a.logger.Info("allegro order poller completed", "tenants", len(tis), "orders", totalOrders)
	return nil
}
