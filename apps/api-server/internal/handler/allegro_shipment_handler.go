package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	allegroprovider "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroShipmentHandler handles Allegro shipment management ("Wysyłam z Allegro") endpoints.
type AllegroShipmentHandler struct {
	integrationService *service.IntegrationService
	shipmentService    *service.ShipmentService
	orderRepo          repository.OrderRepo
	shipmentRepo       repository.ShipmentRepo
	pool               *pgxpool.Pool
	encryptionKey      []byte
}

// NewAllegroShipmentHandler creates a new AllegroShipmentHandler.
func NewAllegroShipmentHandler(
	integrationService *service.IntegrationService,
	shipmentService *service.ShipmentService,
	orderRepo repository.OrderRepo,
	shipmentRepo repository.ShipmentRepo,
	pool *pgxpool.Pool,
	encryptionKey []byte,
) *AllegroShipmentHandler {
	return &AllegroShipmentHandler{
		integrationService: integrationService,
		shipmentService:    shipmentService,
		orderRepo:          orderRepo,
		shipmentRepo:       shipmentRepo,
		pool:               pool,
		encryptionKey:      encryptionKey,
	}
}

// getProviderWithIntegration creates an Allegro provider for the given tenant and returns the integration record.
func (h *AllegroShipmentHandler) getProviderWithIntegration(ctx context.Context, tenantID uuid.UUID) (*allegroprovider.Provider, *model.Integration, error) {
	credJSON, integration, err := h.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, nil, err
	}
	provider, err := allegroprovider.NewProvider(credJSON, nil)
	if err != nil {
		return nil, nil, err
	}
	return provider, integration, nil
}

// getProvider creates an Allegro provider for the given tenant (backward-compat helper).
func (h *AllegroShipmentHandler) getProvider(ctx context.Context, tenantID uuid.UUID) (*allegroprovider.Provider, error) {
	p, _, err := h.getProviderWithIntegration(ctx, tenantID)
	return p, err
}

// allegroCreateShipmentRequest wraps the Allegro SDK command with an optional OpenOMS order_id.
type allegroCreateShipmentRequest struct {
	allegrosdk.CreateShipmentCommand
	OrderID *string `json:"order_id,omitempty"`
}

// ListDeliveryServices returns available delivery services for Allegro shipment management.
// GET /v1/integrations/allegro/delivery-services
func (h *AllegroShipmentHandler) ListDeliveryServices(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	services, err := provider.ListDeliveryServices(r.Context())
	if err != nil {
		slog.Error("allegro shipment: failed to list delivery services", "error", err)
		writeAllegroError(w, "Nie udało się pobrać usług dostawy z Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"delivery_services": services,
	})
}

// CreateShipment creates a new managed shipment via Allegro and links it to an OpenOMS shipment record.
// POST /v1/integrations/allegro/shipments
func (h *AllegroShipmentHandler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	// Decode the extended request body (Allegro command + optional order_id).
	var req allegroCreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy format danych")
		return
	}

	provider, integration, err := h.getProviderWithIntegration(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	// Create the shipment on Allegro's side.
	resp, err := provider.CreateShipment(r.Context(), req.CreateShipmentCommand)
	if err != nil {
		slog.Error("allegro shipment: failed to create shipment", "error", err)
		writeAllegroError(w, "Nie udało się utworzyć przesyłki w Allegro", err)
		return
	}

	// Best-effort: create a linked shipment record in OpenOMS.
	h.linkAllegroShipment(r.Context(), tenantID, actorID, integration, resp, req.OrderID, req.CreateShipmentCommand)

	writeJSON(w, http.StatusCreated, resp)
}

// linkAllegroShipment creates an OpenOMS shipment record linked to the Allegro-managed shipment.
// This is best-effort: errors are logged but do not fail the API response.
func (h *AllegroShipmentHandler) linkAllegroShipment(
	ctx context.Context,
	tenantID, actorID uuid.UUID,
	integration *model.Integration,
	resp *allegrosdk.CreateShipmentResponse,
	orderIDStr *string,
	cmd allegrosdk.CreateShipmentCommand,
) {
	if resp == nil || resp.ShipmentID == "" {
		return
	}

	// Resolve the order ID. The caller may provide an OpenOMS order UUID directly,
	// or an Allegro order external ID that we need to look up.
	var orderID uuid.UUID
	if orderIDStr != nil && *orderIDStr != "" {
		// Try parsing as a UUID (OpenOMS order ID).
		parsed, err := uuid.Parse(*orderIDStr)
		if err == nil {
			orderID = parsed
		} else {
			// Treat it as an Allegro external order ID and look up.
			if err := database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
				order, findErr := h.orderRepo.FindByExternalID(ctx, tx, "allegro", *orderIDStr)
				if findErr != nil {
					return findErr
				}
				if order != nil {
					orderID = order.ID
				}
				return nil
			}); err != nil {
				slog.Warn("allegro shipment link: failed to look up order by external ID",
					"error", err, "external_id", *orderIDStr)
			}
		}
	}

	if orderID == uuid.Nil {
		slog.Info("allegro shipment link: no order_id provided or found, skipping OpenOMS shipment creation",
			"allegro_shipment_id", resp.ShipmentID)
		return
	}

	// Determine the provider name for the shipment record.
	providerName := "allegro"
	if cmd.Input.DeliveryMethodID != "" {
		providerName = "allegro:" + cmd.Input.DeliveryMethodID
	}

	// Build carrier_data with Allegro-specific details.
	carrierData, _ := json.Marshal(map[string]any{
		"allegro_command_id":      resp.CommandID,
		"allegro_shipment_id":     resp.ShipmentID,
		"allegro_status":          resp.Status,
		"delivery_method_id":      cmd.Input.DeliveryMethodID,
		"managed_by":              "allegro",
	})

	integrationID := integration.ID

	shipmentReq := model.CreateShipmentRequest{
		OrderID:       orderID,
		Provider:      providerName,
		IntegrationID: &integrationID,
		ExternalID:    &resp.ShipmentID,
		CarrierData:   carrierData,
	}

	_, err := h.shipmentService.Create(ctx, tenantID, shipmentReq, actorID, "allegro-shipment-api")
	if err != nil {
		slog.Error("allegro shipment link: failed to create OpenOMS shipment record",
			"error", err,
			"allegro_shipment_id", resp.ShipmentID,
			"order_id", orderID,
		)
		return
	}
	slog.Info("allegro shipment link: OpenOMS shipment record created",
		"allegro_shipment_id", resp.ShipmentID,
		"order_id", orderID,
	)
}

// GetLabel generates and returns a shipping label PDF for a given shipment.
// GET /v1/integrations/allegro/shipments/{shipmentId}/label
func (h *AllegroShipmentHandler) GetLabel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	shipmentID := chi.URLParam(r, "shipmentId")
	if shipmentID == "" {
		writeError(w, http.StatusBadRequest, "Brak ID przesyłki")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	pdfBytes, err := provider.GetLabel(r.Context(), []string{shipmentID})
	if err != nil {
		slog.Error("allegro shipment: failed to get label", "error", err, "shipment_id", shipmentID)
		writeAllegroError(w, "Nie udało się pobrać etykiety z Allegro", err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"etykieta-"+shipmentID+".pdf\"")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

// CancelShipment cancels a managed shipment.
// DELETE /v1/integrations/allegro/shipments/{shipmentId}
func (h *AllegroShipmentHandler) CancelShipment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	shipmentID := chi.URLParam(r, "shipmentId")
	if shipmentID == "" {
		writeError(w, http.StatusBadRequest, "Brak ID przesyłki")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	if err := provider.CancelShipment(r.Context(), []string{shipmentID}); err != nil {
		slog.Error("allegro shipment: failed to cancel shipment", "error", err, "shipment_id", shipmentID)
		writeAllegroError(w, "Nie udało się anulować przesyłki w Allegro", err)
		return
	}

	// Best-effort: update the linked OpenOMS shipment status to "cancelled".
	h.cancelLinkedShipment(r.Context(), tenantID, shipmentID)

	w.WriteHeader(http.StatusNoContent)
}

// cancelLinkedShipment finds the OpenOMS shipment record by external_id and marks it as cancelled.
func (h *AllegroShipmentHandler) cancelLinkedShipment(ctx context.Context, tenantID uuid.UUID, allegroShipmentID string) {
	err := database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		shipment, err := h.shipmentRepo.FindByExternalID(ctx, tx, allegroShipmentID)
		if err != nil {
			return err
		}
		if shipment == nil {
			slog.Debug("allegro shipment link: no OpenOMS shipment found for cancel",
				"allegro_shipment_id", allegroShipmentID)
			return nil
		}
		if err := h.shipmentRepo.UpdateStatus(ctx, tx, shipment.ID, "cancelled"); err != nil {
			return err
		}
		slog.Info("allegro shipment link: OpenOMS shipment status set to cancelled",
			"shipment_id", shipment.ID,
			"allegro_shipment_id", allegroShipmentID,
		)
		return nil
	})
	if err != nil {
		slog.Error("allegro shipment link: failed to cancel OpenOMS shipment",
			"error", err,
			"allegro_shipment_id", allegroShipmentID,
		)
	}
}

// GetPickupProposals retrieves available pickup proposals.
// POST /v1/integrations/allegro/pickup-proposals
func (h *AllegroShipmentHandler) GetPickupProposals(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req allegrosdk.PickupProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy format danych")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	proposals, err := provider.GetPickupProposals(r.Context(), req)
	if err != nil {
		slog.Error("allegro shipment: failed to get pickup proposals", "error", err)
		writeAllegroError(w, "Nie udało się pobrać propozycji odbioru z Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"proposals": proposals,
	})
}

// SchedulePickup schedules a courier pickup.
// POST /v1/integrations/allegro/pickups
func (h *AllegroShipmentHandler) SchedulePickup(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cmd allegrosdk.SchedulePickupCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy format danych")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	if err := provider.SchedulePickup(r.Context(), cmd); err != nil {
		slog.Error("allegro shipment: failed to schedule pickup", "error", err)
		writeAllegroError(w, "Nie udało się zaplanować odbioru w Allegro", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateProtocol generates a dispatch protocol PDF.
// POST /v1/integrations/allegro/protocol
func (h *AllegroShipmentHandler) GenerateProtocol(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var body struct {
		ShipmentIDs []string `json:"shipment_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy format danych")
		return
	}
	if len(body.ShipmentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "Wymagane jest co najmniej jedno ID przesyłki")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Nie można połączyć z Allegro. Sprawdź konfigurację integracji.")
		return
	}
	defer provider.Close()

	pdfBytes, err := provider.GenerateProtocol(r.Context(), body.ShipmentIDs)
	if err != nil {
		slog.Error("allegro shipment: failed to generate protocol", "error", err)
		writeAllegroError(w, "Nie udało się wygenerować protokołu z Allegro", err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"protokol-allegro.pdf\"")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}
