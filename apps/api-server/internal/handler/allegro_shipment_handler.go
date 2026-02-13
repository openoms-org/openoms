package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	allegroprovider "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// AllegroShipmentHandler handles Allegro shipment management ("Wysyłam z Allegro") endpoints.
type AllegroShipmentHandler struct {
	integrationService *service.IntegrationService
	encryptionKey      []byte
}

// NewAllegroShipmentHandler creates a new AllegroShipmentHandler.
func NewAllegroShipmentHandler(integrationService *service.IntegrationService, encryptionKey []byte) *AllegroShipmentHandler {
	return &AllegroShipmentHandler{
		integrationService: integrationService,
		encryptionKey:      encryptionKey,
	}
}

// getProvider creates an Allegro provider for the given tenant.
func (h *AllegroShipmentHandler) getProvider(ctx context.Context, tenantID uuid.UUID) (*allegroprovider.Provider, error) {
	credJSON, _, err := h.integrationService.GetDecryptedCredentialsByProvider(ctx, tenantID, "allegro")
	if err != nil {
		return nil, err
	}
	provider, err := allegroprovider.NewProvider(credJSON, nil)
	if err != nil {
		return nil, err
	}
	return provider, nil
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
		writeError(w, http.StatusBadGateway, "Nie udało się pobrać usług dostawy z Allegro")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"delivery_services": services,
	})
}

// CreateShipment creates a new managed shipment via Allegro.
// POST /v1/integrations/allegro/shipments
func (h *AllegroShipmentHandler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var cmd allegrosdk.CreateShipmentCommand
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

	resp, err := provider.CreateShipment(r.Context(), cmd)
	if err != nil {
		slog.Error("allegro shipment: failed to create shipment", "error", err)
		writeError(w, http.StatusBadGateway, "Nie udało się utworzyć przesyłki w Allegro")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
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
		writeError(w, http.StatusBadGateway, "Nie udało się pobrać etykiety z Allegro")
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
		writeError(w, http.StatusBadGateway, "Nie udało się anulować przesyłki w Allegro")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		writeError(w, http.StatusBadGateway, "Nie udało się pobrać propozycji odbioru z Allegro")
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
		writeError(w, http.StatusBadGateway, "Nie udało się zaplanować odbioru w Allegro")
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
		writeError(w, http.StatusBadGateway, "Nie udało się wygenerować protokołu z Allegro")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"protokol-allegro.pdf\"")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}
