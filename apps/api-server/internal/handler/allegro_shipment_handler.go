package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	allegroprovider "github.com/openoms-org/openoms/apps/api-server/internal/integration/allegro"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

// AllegroShipmentHandler handles Allegro shipment management ("Wysyłam z Allegro") endpoints.
type AllegroShipmentHandler struct {
	integrationService *service.IntegrationService
	shipmentService    *service.ShipmentService
	orderRepo          repository.OrderRepo
	shipmentRepo       repository.ShipmentRepo
	pool               *pgxpool.Pool
	encryptionKey      []byte
	objectStorage      storage.ObjectStorage
	baseURL            string
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
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	services, err := provider.ListDeliveryServices(r.Context())
	if err != nil {
		slog.Error("allegro shipment: failed to list delivery services", "error", err)
		writeAllegroError(w, "Failed to fetch delivery services from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"delivery_services": services,
	})
}

// GetDeliveryProposals returns the official prefilled create-commands body for a checkout-form.
// GET /v1/integrations/allegro/delivery-proposals/{orderId}
func (h *AllegroShipmentHandler) GetDeliveryProposals(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	orderID := chi.URLParam(r, "orderId")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "Missing Allegro order ID")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	proposals, err := provider.GetDeliveryProposals(r.Context(), orderID)
	if err != nil {
		slog.Error("allegro shipment: failed to get delivery proposals", "error", err)
		writeAllegroError(w, "Failed to fetch delivery proposals from Allegro", err)
		return
	}

	writeJSON(w, http.StatusOK, proposals)
}

// CreateShipment creates a new managed shipment via Allegro and links it to an OpenOMS shipment record.
// POST /v1/integrations/allegro/shipments
func (h *AllegroShipmentHandler) CreateShipment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	// Decode the extended request body (Allegro command + optional order_id).
	var req allegroCreateShipmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	provider, integration, err := h.getProviderWithIntegration(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	omsOrder := h.loadOMSOrder(r.Context(), tenantID, req.OrderID)
	checkoutFormID := ""
	if omsOrder != nil && omsOrder.ExternalID != nil {
		checkoutFormID = strings.TrimSpace(*omsOrder.ExternalID)
	}
	if checkoutFormID == "" {
		checkoutFormID = h.resolveCheckoutFormID(r.Context(), tenantID, req.OrderID)
	}
	if checkoutFormID != "" {
		proposals, propErr := provider.GetDeliveryProposals(r.Context(), checkoutFormID)
		if propErr != nil {
			slog.Error("allegro shipment: failed to get delivery proposals before create", "error", propErr)
			writeAllegroError(w, "Failed to fetch delivery proposals from Allegro", propErr)
			return
		}
		checkoutID, checkoutName := checkoutWzAMethodFromOrder(omsOrder)
		var catalogIDs []string
		if services, listErr := provider.ListDeliveryServices(r.Context()); listErr != nil {
			slog.Warn("allegro shipment: delivery-services unavailable for WzA method check", "error", listErr)
		} else {
			for _, svc := range services {
				catalogIDs = append(catalogIDs, svc.ID)
			}
		}
		if valErr := allegrosdk.ValidateWzACreateCommand(allegrosdk.WzACreateMethodInput{
			ProposedDeliveryMethodID:  proposals.SuggestedInput.DeliveryMethodID,
			RequestedDeliveryMethodID: req.Input.DeliveryMethodID,
			CheckoutMethodID:          checkoutID,
			CheckoutMethodName:        checkoutName,
			CatalogServiceIDs:         catalogIDs,
		}); valErr != nil {
			writeError(w, http.StatusConflict, valErr.Error())
			return
		}
	}

	// Create the shipment on Allegro's side.
	resp, err := provider.CreateShipment(r.Context(), req.CreateShipmentCommand)
	if err != nil {
		slog.Error("allegro shipment: failed to create shipment", "error", err)
		writeAllegroError(w, "Failed to create shipment on Allegro", err)
		return
	}

	// Best-effort: create a linked shipment record in OpenOMS.
	h.linkAllegroShipment(r.Context(), tenantID, actorID, integration, resp, req.OrderID, req.CreateShipmentCommand)

	writeJSON(w, http.StatusCreated, resp)
}

// resolveCheckoutFormID returns the Allegro checkout-form id used by delivery-proposals.
// order_id may be an OMS order UUID or the checkout-form id itself (also a UUID).
func (h *AllegroShipmentHandler) resolveCheckoutFormID(ctx context.Context, tenantID uuid.UUID, orderIDStr *string) string {
	if orderIDStr == nil {
		return ""
	}
	id := strings.TrimSpace(*orderIDStr)
	if id == "" {
		return ""
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return id
	}
	if h.pool != nil && h.orderRepo != nil {
		var externalID string
		_ = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
			order, findErr := h.orderRepo.FindByID(ctx, tx, parsed)
			if findErr != nil {
				return findErr
			}
			if order != nil && order.ExternalID != nil && *order.ExternalID != "" {
				externalID = *order.ExternalID
			}
			return nil
		})
		if externalID != "" {
			return externalID
		}
	}
	return id
}

func (h *AllegroShipmentHandler) loadOMSOrder(ctx context.Context, tenantID uuid.UUID, orderIDStr *string) *model.Order {
	if orderIDStr == nil || h.pool == nil || h.orderRepo == nil {
		return nil
	}
	id := strings.TrimSpace(*orderIDStr)
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil
	}
	var found *model.Order
	_ = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		order, findErr := h.orderRepo.FindByID(ctx, tx, parsed)
		if findErr != nil {
			return findErr
		}
		found = order
		return nil
	})
	return found
}

func checkoutWzAMethodFromOrder(order *model.Order) (id, name string) {
	if order == nil {
		return "", ""
	}
	if order.DeliveryMethod != nil {
		name = strings.TrimSpace(*order.DeliveryMethod)
	}
	if len(order.Metadata) == 0 {
		return "", name
	}
	var metadata map[string]any
	if err := json.Unmarshal(order.Metadata, &metadata); err != nil {
		return "", name
	}
	if v, ok := metadata["delivery_method_id"].(string); ok {
		id = strings.TrimSpace(v)
	}
	if v, ok := metadata["delivery_method_name"].(string); ok && strings.TrimSpace(v) != "" {
		name = strings.TrimSpace(v)
	}
	return id, name
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
		"allegro_command_id":  resp.CommandID,
		"allegro_shipment_id": resp.ShipmentID,
		"allegro_status":      resp.Status,
		"delivery_method_id":  cmd.Input.DeliveryMethodID,
		"managed_by":          "allegro",
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
		writeError(w, http.StatusBadRequest, "Missing shipment ID")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	pdfBytes, err := provider.GetLabel(r.Context(), []string{shipmentID})
	if err != nil {
		slog.Error("allegro shipment: failed to get label", "error", err, "shipment_id", shipmentID) //nolint:gosec
		writeAllegroError(w, "Failed to fetch label from Allegro", err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"label-"+shipmentID+".pdf\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes) // #nosec G705
}

// CancelShipment cancels a managed shipment.
// DELETE /v1/integrations/allegro/shipments/{shipmentId}
func (h *AllegroShipmentHandler) CancelShipment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	shipmentID := chi.URLParam(r, "shipmentId")
	if shipmentID == "" {
		writeError(w, http.StatusBadRequest, "Missing shipment ID")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	if err := provider.CancelShipment(r.Context(), []string{shipmentID}); err != nil {
		slog.Error("allegro shipment: failed to cancel shipment", "error", err, "shipment_id", shipmentID) //nolint:gosec
		writeAllegroError(w, "Failed to cancel shipment on Allegro", err)
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
			slog.Debug("allegro shipment link: no OpenOMS shipment found for cancel", "allegro_shipment_id", allegroShipmentID) //nolint:gosec
			return nil
		}
		if err := h.shipmentRepo.UpdateStatus(ctx, tx, shipment.ID, "cancelled"); err != nil {
			return err
		}
		slog.Info("allegro shipment link: OpenOMS shipment status set to cancelled", "shipment_id", shipment.ID, "allegro_shipment_id", allegroShipmentID) //nolint:gosec
		return nil
	})
	if err != nil {
		slog.Error("allegro shipment link: failed to cancel OpenOMS shipment", //nolint:gosec
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
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	proposals, err := provider.GetPickupProposals(r.Context(), req)
	if err != nil {
		slog.Error("allegro shipment: failed to get pickup proposals", "error", err)
		writeAllegroError(w, "Failed to fetch pickup proposals from Allegro", err)
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
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	if err := provider.SchedulePickup(r.Context(), cmd); err != nil {
		slog.Error("allegro shipment: failed to schedule pickup", "error", err)
		writeAllegroError(w, "Failed to schedule pickup on Allegro", err)
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
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(body.ShipmentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "At least one shipment ID is required")
		return
	}

	provider, err := h.getProvider(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro shipment: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	pdfBytes, err := provider.GenerateProtocol(r.Context(), body.ShipmentIDs)
	if err != nil {
		slog.Error("allegro shipment: failed to generate protocol", "error", err)
		writeAllegroError(w, "Failed to generate protocol from Allegro", err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"protocol-allegro.pdf\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}
