package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// AllegroSyncService handles automatic synchronization of order fulfillment
// status and shipment tracking to Allegro when changes occur in OpenOMS.
type AllegroSyncService struct {
	integrationService *IntegrationService
	logger             *slog.Logger
}

// NewAllegroSyncService creates an AllegroSyncService.
func NewAllegroSyncService(integrationService *IntegrationService) *AllegroSyncService {
	return &AllegroSyncService{
		integrationService: integrationService,
		logger:             slog.Default().With("component", "allegro_sync"),
	}
}

// mapStatusToAllegro maps an OpenOMS order status to the corresponding Allegro fulfillment status.
// Returns empty string if the status should not be synced.
func mapStatusToAllegro(status string) string {
	switch status {
	case "shipped":
		return "SENT"
	case "cancelled":
		return "CANCELLED"
	case "processing":
		return "PROCESSING"
	case "ready_to_ship":
		return "READY_FOR_SHIPMENT"
	default:
		return ""
	}
}

// mapCarrierToAllegro maps an OpenOMS carrier provider name to the Allegro carrier ID.
func mapCarrierToAllegro(provider string) string {
	mapping := map[string]string{
		"inpost":        "INPOST",
		"dhl":           "DHL",
		"dpd":           "DPD",
		"gls":           "GLS",
		"ups":           "UPS",
		"fedex":         "FEDEX",
		"poczta_polska": "POCZTA_POLSKA",
		"orlen_paczka":  "ORLEN",
	}
	upper := strings.ToUpper(provider)
	if mapped, ok := mapping[provider]; ok {
		return mapped
	}
	// Fallback: use uppercase provider name
	return upper
}

// SyncFulfillmentStatus sends an order fulfillment status update to Allegro.
// This should be called asynchronously (in a goroutine) so it does not block the main flow.
// It uses context.Background() since the original request context may be cancelled.
func (s *AllegroSyncService) SyncFulfillmentStatus(ctx context.Context, tenantID uuid.UUID, order *model.Order, newStatus string) {
	if order == nil {
		return
	}

	// Only sync orders that originated from Allegro
	if order.Source != "allegro" {
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		return
	}

	allegroStatus := mapStatusToAllegro(newStatus)
	if allegroStatus == "" {
		return
	}

	provider, err := s.buildProvider(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to build Allegro provider for fulfillment sync",
			"tenant_id", tenantID,
			"order_id", order.ID,
			"error", err,
		)
		return
	}
	defer provider.Close()

	if err := provider.UpdateFulfillment(ctx, *order.ExternalID, allegroStatus); err != nil {
		s.logger.Error("failed to sync fulfillment status to Allegro",
			"tenant_id", tenantID,
			"order_id", order.ID,
			"external_id", *order.ExternalID,
			"allegro_status", allegroStatus,
			"error", err,
		)
		return
	}

	s.logger.Info("fulfillment status synced to Allegro",
		"tenant_id", tenantID,
		"order_id", order.ID,
		"external_id", *order.ExternalID,
		"allegro_status", allegroStatus,
	)
}

// SyncTracking sends shipment tracking information to Allegro.
// This should be called asynchronously (in a goroutine) so it does not block the main flow.
func (s *AllegroSyncService) SyncTracking(ctx context.Context, tenantID uuid.UUID, order *model.Order, carrierProvider string, trackingNumber string) {
	if order == nil {
		return
	}

	// Only sync orders that originated from Allegro
	if order.Source != "allegro" {
		return
	}
	if order.ExternalID == nil || *order.ExternalID == "" {
		return
	}
	if trackingNumber == "" {
		return
	}

	allegroCarrier := mapCarrierToAllegro(carrierProvider)

	provider, err := s.buildProvider(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to build Allegro provider for tracking sync",
			"tenant_id", tenantID,
			"order_id", order.ID,
			"error", err,
		)
		return
	}
	defer provider.Close()

	if err := provider.AddTracking(ctx, *order.ExternalID, allegroCarrier, trackingNumber); err != nil {
		s.logger.Error("failed to sync tracking to Allegro",
			"tenant_id", tenantID,
			"order_id", order.ID,
			"external_id", *order.ExternalID,
			"carrier", allegroCarrier,
			"tracking", trackingNumber,
			"error", err,
		)
		return
	}

	s.logger.Info("tracking synced to Allegro",
		"tenant_id", tenantID,
		"order_id", order.ID,
		"external_id", *order.ExternalID,
		"carrier", allegroCarrier,
		"tracking", trackingNumber,
	)
}

// buildProvider creates an Allegro provider using decrypted integration credentials.
func (s *AllegroSyncService) buildProvider(ctx context.Context, tenantID uuid.UUID) (*allegroIntegration.Provider, error) {
	credJSON, _, err := s.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, err
	}
	provider, err := allegroIntegration.NewProvider(json.RawMessage(credJSON), nil)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
