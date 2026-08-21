package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	allegroIntegration "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// AllegroSyncService handles automatic synchronization of order fulfillment
// status and shipment tracking to Allegro when changes occur in OpenOMS.
type AllegroSyncService struct {
	integrationService *IntegrationService
	logger             *slog.Logger
	// fulfillment optionally records the outbound marketplace sync as a provider attempt
	// + fulfillment step (OPE-417 followup). Gated/best-effort via the service itself;
	// nil (or disabled) is a no-op. Wired by WithFulfillment.
	fulfillment *FulfillmentService
}

// NewAllegroSyncService creates an AllegroSyncService.
func NewAllegroSyncService(integrationService *IntegrationService) *AllegroSyncService {
	return &AllegroSyncService{
		integrationService: integrationService,
		logger:             slog.Default().With("component", "allegro_sync"),
	}
}

// WithFulfillment wires the OPE-417 fulfillment recorder so each outbound marketplace
// status/tracking sync is recorded as a provider attempt (and a sync_tracking_to_marketplace
// step). Returns the service for chaining. Safe to omit: recording is gated/best-effort.
func (s *AllegroSyncService) WithFulfillment(f *FulfillmentService) *AllegroSyncService {
	if s == nil {
		return nil
	}
	s.fulfillment = f
	return s
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
		s.recordMarketplaceAttempt(ctx, tenantID, order, model.ProviderOpSyncFulfillmentStatus,
			model.ProviderAttemptFailed, allegroStatus)
		return
	}

	s.logger.Info("fulfillment status synced to Allegro",
		"tenant_id", tenantID,
		"order_id", order.ID,
		"external_id", *order.ExternalID,
		"allegro_status", allegroStatus,
	)
	s.recordMarketplaceAttempt(ctx, tenantID, order, model.ProviderOpSyncFulfillmentStatus,
		model.ProviderAttemptSucceeded, allegroStatus)
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
		s.recordMarketplaceAttempt(ctx, tenantID, order, model.ProviderOpSyncTrackingToMarketplace,
			model.ProviderAttemptFailed, allegroCarrier)
		s.emitTrackingSyncStep(ctx, tenantID, order, model.FulfillmentStatusFailed)
		return
	}

	s.logger.Info("tracking synced to Allegro",
		"tenant_id", tenantID,
		"order_id", order.ID,
		"external_id", *order.ExternalID,
		"carrier", allegroCarrier,
		"tracking", trackingNumber,
	)
	s.recordMarketplaceAttempt(ctx, tenantID, order, model.ProviderOpSyncTrackingToMarketplace,
		model.ProviderAttemptSucceeded, allegroCarrier)
	s.emitTrackingSyncStep(ctx, tenantID, order, model.FulfillmentStatusSucceeded)
}

// recordMarketplaceAttempt records an outbound marketplace sync as a provider attempt
// (OPE-417 followup). BEST-EFFORT + gated: when no fulfillment recorder is wired (or it is
// disabled) it is a complete no-op, and the recorder opens its own tenant transaction and
// swallows errors, so it can never affect the (already best-effort, async) sync itself.
// provider is "allegro"; rawStatus carries the Allegro-side status/carrier for context.
func (s *AllegroSyncService) recordMarketplaceAttempt(ctx context.Context, tenantID uuid.UUID, order *model.Order, operation, status, rawStatus string) {
	if s.fulfillment == nil {
		return
	}
	s.fulfillment.RecordProviderAttempt(ctx, ProviderAttemptInput{
		TenantID:          tenantID,
		OrderID:           order.ID,
		Provider:          "allegro",
		Operation:         operation,
		Status:            status,
		RawProviderStatus: rawStatus,
	})
}

// emitTrackingSyncStep emits the sync_tracking_to_marketplace fulfillment step for the
// order's process (OPE-417 followup). BEST-EFFORT + gated; no-op when no recorder is wired.
func (s *AllegroSyncService) emitTrackingSyncStep(ctx context.Context, tenantID uuid.UUID, order *model.Order, status string) {
	if s.fulfillment == nil {
		return
	}
	s.fulfillment.EmitFulfillmentStep(ctx, tenantID, order.ID, model.StepSyncTrackingToMkt, status, "", nil)
}

// buildProvider creates an Allegro provider using decrypted integration credentials.
func (s *AllegroSyncService) buildProvider(ctx context.Context, tenantID uuid.UUID) (*allegroIntegration.Provider, error) {
	credJSON, _, err := s.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, err
	}
	provider, err := allegroIntegration.NewProvider(json.RawMessage(credJSON), nil, allegroIntegration.WithTokenRefreshPersist(
		allegroIntegration.PersistFn(credJSON, func(newJSON []byte) error {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			return s.integrationService.UpdateCredentialsByProvider(persistCtx, tenantID, "allegro", newJSON)
		}),
	))
	if err != nil {
		return nil, err
	}
	return provider, nil
}
