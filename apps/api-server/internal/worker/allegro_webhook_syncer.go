package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// AllegroWebhookSyncer handles on-demand order import/update triggered by Allegro webhooks.
// It iterates active Allegro integrations to resolve which tenant owns the order,
// then fetches full order data from the Allegro API and creates/updates it in OpenOMS.
type AllegroWebhookSyncer struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
	orderRepo     repository.OrderRepo
	shipmentRepo  repository.ShipmentRepo
	auditRepo     repository.AuditRepo
	logger        *slog.Logger
}

// NewAllegroWebhookSyncer creates a new AllegroWebhookSyncer.
func NewAllegroWebhookSyncer(
	pool *pgxpool.Pool,
	encryptionKey []byte,
	orderRepo repository.OrderRepo,
	shipmentRepo repository.ShipmentRepo,
	auditRepo repository.AuditRepo,
	logger *slog.Logger,
) *AllegroWebhookSyncer {
	return &AllegroWebhookSyncer{
		pool:          pool,
		encryptionKey: encryptionKey,
		orderRepo:     orderRepo,
		shipmentRepo:  shipmentRepo,
		auditRepo:     auditRepo,
		logger:        logger.With("component", "allegro_webhook_syncer"),
	}
}

// ImportOrder fetches a single order from Allegro by its external order ID and creates it
// in OpenOMS if it does not already exist. It iterates all active Allegro integrations
// to find the one that owns the order.
func (s *AllegroWebhookSyncer) ImportOrder(ctx context.Context, allegroOrderID string) {
	if allegroOrderID == "" {
		s.logger.Warn("allegro webhook syncer: empty order ID, skipping")
		return
	}

	tis, err := ListActiveIntegrations(ctx, s.pool, "allegro")
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to list integrations", "error", err)
		return
	}

	if len(tis) == 0 {
		s.logger.Warn("allegro webhook syncer: no active allegro integrations found")
		return
	}

	for _, ti := range tis {
		if err := checkWorkerContext(ctx); err != nil {
			return
		}
		imported := s.tryImportOrder(ctx, ti, allegroOrderID)
		if imported {
			return
		}
	}

	s.logger.Warn("allegro webhook syncer: order not found in any active integration",
		"allegro_order_id", allegroOrderID,
	)
}

// ImportOrderForIntegration imports an Allegro order only for the verified tenant integration.
func (s *AllegroWebhookSyncer) ImportOrderForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID, allegroOrderID string) {
	ti, err := s.getActiveIntegration(ctx, tenantID, integrationID)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to load scoped integration",
			"tenant_id", tenantID,
			"integration_id", integrationID,
			"error", err,
		)
		return
	}

	if imported := s.tryImportOrder(ctx, *ti, allegroOrderID); !imported {
		s.logger.Warn("allegro webhook syncer: scoped order import did not complete",
			"tenant_id", tenantID,
			"integration_id", integrationID,
			"allegro_order_id", allegroOrderID,
		)
	}
}

// UpdateOrderStatus finds an existing order by external ID and updates its status
// based on the Allegro order status. If the order is not found, it triggers a full import.
func (s *AllegroWebhookSyncer) UpdateOrderStatus(ctx context.Context, allegroOrderID string) {
	if allegroOrderID == "" {
		s.logger.Warn("allegro webhook syncer: empty order ID for status update, skipping")
		return
	}

	tis, err := ListActiveIntegrations(ctx, s.pool, "allegro")
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to list integrations", "error", err)
		return
	}

	if len(tis) == 0 {
		s.logger.Warn("allegro webhook syncer: no active allegro integrations found")
		return
	}

	for _, ti := range tis {
		if err := checkWorkerContext(ctx); err != nil {
			return
		}
		updated := s.tryUpdateOrderStatus(ctx, ti, allegroOrderID)
		if updated {
			return
		}
	}

	// Order not found in any integration — try a full import (may have been missed by poller)
	s.logger.Info("allegro webhook syncer: order not found for status update, attempting full import",
		"allegro_order_id", allegroOrderID,
	)
	s.ImportOrder(ctx, allegroOrderID)
}

// UpdateOrderStatusForIntegration updates an Allegro order status only for the verified tenant integration.
func (s *AllegroWebhookSyncer) UpdateOrderStatusForIntegration(ctx context.Context, tenantID, integrationID uuid.UUID, allegroOrderID string) {
	ti, err := s.getActiveIntegration(ctx, tenantID, integrationID)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to load scoped integration for status update",
			"tenant_id", tenantID,
			"integration_id", integrationID,
			"error", err,
		)
		return
	}

	if updated := s.tryUpdateOrderStatus(ctx, *ti, allegroOrderID); updated {
		return
	}

	s.logger.Info("allegro webhook syncer: scoped order not found for status update, attempting scoped import",
		"tenant_id", tenantID,
		"integration_id", integrationID,
		"allegro_order_id", allegroOrderID,
	)
	s.tryImportOrder(ctx, *ti, allegroOrderID)
}

func (s *AllegroWebhookSyncer) getActiveIntegration(ctx context.Context, tenantID, integrationID uuid.UUID) (*TenantIntegration, error) {
	var ti TenantIntegration
	var credsJSON json.RawMessage
	err := s.pool.QueryRow(ctx,
		`SELECT tenant_id, id, provider, sync_cursor, credentials, settings
		   FROM integrations
		  WHERE tenant_id = $1 AND id = $2 AND provider = 'allegro' AND status = 'active'`,
		tenantID,
		integrationID,
	).Scan(&ti.TenantID, &ti.IntegrationID, &ti.Provider, &ti.SyncCursor, &credsJSON, &ti.Settings)
	if err != nil {
		return nil, err
	}
	credentials, err := decodeIntegrationCredentialsJSONB(credsJSON)
	if err != nil {
		return nil, err
	}
	ti.Credentials = credentials
	return &ti, nil
}

// tryImportOrder attempts to import an order from a specific integration.
// Returns true if the order was successfully imported or already exists.
func (s *AllegroWebhookSyncer) tryImportOrder(ctx context.Context, ti TenantIntegration, allegroOrderID string) bool {
	// First check if the order already exists for this tenant
	var exists bool
	if err := database.WithTenant(ctx, s.pool, ti.TenantID, func(tx pgx.Tx) error {
		existing, err := s.orderRepo.FindByExternalID(ctx, tx, "allegro", allegroOrderID)
		if err != nil {
			return err
		}
		exists = existing != nil
		return nil
	}); err != nil {
		s.logger.Error("allegro webhook syncer: failed to check existing order",
			"tenant_id", ti.TenantID,
			"allegro_order_id", allegroOrderID,
			"error", err,
		)
		return false
	}

	if exists {
		s.logger.Debug("allegro webhook syncer: order already exists, skipping",
			"tenant_id", ti.TenantID,
			"allegro_order_id", allegroOrderID,
		)
		return true
	}

	// Decrypt credentials and create provider
	credJSON, err := crypto.Decrypt(ti.Credentials, s.encryptionKey)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to decrypt credentials",
			"integration_id", ti.IntegrationID,
			"error", err,
		)
		return false
	}

	provider, err := integration.NewMarketplaceProvider("allegro", credJSON, ti.Settings)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to create provider",
			"integration_id", ti.IntegrationID,
			"error", err,
		)
		return false
	}

	// Fetch the order from Allegro
	mo, err := provider.GetOrder(ctx, allegroOrderID)
	if err != nil {
		// This integration doesn't own the order (404) or API error — try next
		s.logger.Debug("allegro webhook syncer: could not fetch order from this integration",
			"integration_id", ti.IntegrationID,
			"allegro_order_id", allegroOrderID,
			"error", err,
		)
		return false
	}

	// Build and create the order
	req := integration.MarketplaceOrderToCreateRequest(*mo, "allegro", ti.IntegrationID)
	order := allegroOrderMapper(*mo, ti, req)

	created := false
	if err := database.WithTenant(ctx, s.pool, ti.TenantID, func(tx pgx.Tx) error {
		var err error
		created, err = insertMarketplaceOrderIfNotExists(ctx, tx, s.orderRepo, "allegro", mo.ExternalID, &order)
		if err != nil || !created {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   ti.TenantID,
			UserID:     uuid.Nil,
			Action:     "order.created",
			EntityType: "order",
			EntityID:   order.ID,
			Changes:    map[string]string{"source": "allegro", "trigger": "webhook", "external_id": mo.ExternalID},
			IPAddress:  "0.0.0.0",
		})
	}); err != nil {
		s.logger.Error("allegro webhook syncer: failed to create order",
			"integration_id", ti.IntegrationID,
			"allegro_order_id", allegroOrderID,
			"error", err,
		)
		return false
	}
	if !created {
		s.logger.Debug("allegro webhook syncer: duplicate order skipped",
			"tenant_id", ti.TenantID,
			"allegro_order_id", allegroOrderID,
			"integration_id", ti.IntegrationID,
		)
		return true
	}

	s.logger.Info("allegro webhook syncer: order imported via webhook",
		"tenant_id", ti.TenantID,
		"order_id", order.ID,
		"allegro_order_id", allegroOrderID,
		"integration_id", ti.IntegrationID,
	)

	// Auto-create shipment (best effort, same logic as poller)
	if s.shipmentRepo != nil {
		dm := ""
		if order.DeliveryMethod != nil {
			dm = *order.DeliveryMethod
		}
		carrier := resolveCarrier(ti.Settings, dm)
		if carrier != "" {
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

			if err := database.WithTenant(ctx, s.pool, ti.TenantID, func(tx pgx.Tx) error {
				if err := s.shipmentRepo.Create(ctx, tx, shipment); err != nil {
					return err
				}
				if s.auditRepo != nil {
					return s.auditRepo.Log(ctx, tx, model.AuditEntry{
						TenantID:   ti.TenantID,
						UserID:     uuid.Nil,
						Action:     "shipment.created",
						EntityType: "shipment",
						EntityID:   shipment.ID,
						Changes:    map[string]string{"order_id": order.ID.String(), "provider": carrier, "auto": "true", "trigger": "webhook"},
						IPAddress:  "0.0.0.0",
					})
				}
				return nil
			}); err != nil {
				s.logger.Error("allegro webhook syncer: auto-create shipment failed (non-fatal)",
					"order_id", order.ID,
					"carrier", carrier,
					"error", err,
				)
			} else {
				s.logger.Info("allegro webhook syncer: auto-created shipment",
					"order_id", order.ID,
					"carrier", carrier,
				)
			}
		}
	}

	return true
}

// tryUpdateOrderStatus attempts to update an order's status from a specific integration.
// Returns true if the order was found and updated (or no update was needed).
func (s *AllegroWebhookSyncer) tryUpdateOrderStatus(ctx context.Context, ti TenantIntegration, allegroOrderID string) bool {
	// Check if the order exists for this tenant
	var existingOrder *model.Order
	if err := database.WithTenant(ctx, s.pool, ti.TenantID, func(tx pgx.Tx) error {
		var err error
		existingOrder, err = s.orderRepo.FindByExternalID(ctx, tx, "allegro", allegroOrderID)
		return err
	}); err != nil {
		s.logger.Error("allegro webhook syncer: failed to find order for status update",
			"tenant_id", ti.TenantID,
			"allegro_order_id", allegroOrderID,
			"error", err,
		)
		return false
	}

	if existingOrder == nil {
		return false // order not in this tenant
	}

	// Decrypt credentials and create provider
	credJSON, err := crypto.Decrypt(ti.Credentials, s.encryptionKey)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to decrypt credentials",
			"integration_id", ti.IntegrationID,
			"error", err,
		)
		return true // order exists in this tenant, just can't update — don't try other tenants
	}

	provider, err := integration.NewMarketplaceProvider("allegro", credJSON, ti.Settings)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to create provider",
			"integration_id", ti.IntegrationID,
			"error", err,
		)
		return true
	}

	// Fetch current order status from Allegro
	mo, err := provider.GetOrder(ctx, allegroOrderID)
	if err != nil {
		s.logger.Error("allegro webhook syncer: failed to fetch order for status update",
			"integration_id", ti.IntegrationID,
			"allegro_order_id", allegroOrderID,
			"error", err,
		)
		return true
	}

	// Map Allegro status to OpenOMS status
	newStatus := mapAllegroStatus(mo.ExternalStatus)
	if newStatus == "" || newStatus == existingOrder.Status {
		s.logger.Debug("allegro webhook syncer: no status change needed",
			"allegro_order_id", allegroOrderID,
			"current_status", existingOrder.Status,
			"allegro_status", mo.ExternalStatus,
		)
		return true
	}

	// Update the order status
	if err := database.WithTenant(ctx, s.pool, ti.TenantID, func(tx pgx.Tx) error {
		if err := s.orderRepo.UpdateStatus(ctx, tx, existingOrder.ID, newStatus, nil, nil); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   ti.TenantID,
			UserID:     uuid.Nil,
			Action:     "order.status_changed",
			EntityType: "order",
			EntityID:   existingOrder.ID,
			Changes: map[string]string{
				"from":             existingOrder.Status,
				"to":               newStatus,
				"allegro_status":   mo.ExternalStatus,
				"trigger":          "webhook",
				"allegro_order_id": allegroOrderID,
			},
			IPAddress: "0.0.0.0",
		})
	}); err != nil {
		s.logger.Error("allegro webhook syncer: failed to update order status",
			"order_id", existingOrder.ID,
			"allegro_order_id", allegroOrderID,
			"from", existingOrder.Status,
			"to", newStatus,
			"error", err,
		)
		return true
	}

	s.logger.Info("allegro webhook syncer: order status updated via webhook",
		"tenant_id", ti.TenantID,
		"order_id", existingOrder.ID,
		"allegro_order_id", allegroOrderID,
		"from", existingOrder.Status,
		"to", newStatus,
		"allegro_status", mo.ExternalStatus,
	)

	return true
}

// mapAllegroStatus maps Allegro order statuses to OpenOMS order statuses.
func mapAllegroStatus(allegroStatus string) string {
	switch allegroStatus {
	case "READY_FOR_PROCESSING":
		return "new"
	case "PROCESSING":
		return "processing"
	case "SENT":
		return "shipped"
	case "DELIVERED":
		return "delivered"
	case "CANCELLED":
		return "cancelled"
	case "RETURNED":
		return "returned"
	default:
		return ""
	}
}
