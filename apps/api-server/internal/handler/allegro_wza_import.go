package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/storage"
)

type wzaImportPlan struct {
	Creates []allegrosdk.ExistingWzAShipment
	Already []model.Shipment
}

func planWzAImport(oms []model.Shipment, found []allegrosdk.ExistingWzAShipment) wzaImportPlan {
	var plan wzaImportPlan
	for _, item := range found {
		if existing := findImportedWzA(oms, item); existing != nil {
			plan.Already = append(plan.Already, *existing)
			continue
		}
		plan.Creates = append(plan.Creates, item)
	}
	return plan
}

func findImportedWzA(oms []model.Shipment, item allegrosdk.ExistingWzAShipment) *model.Shipment {
	waybill := strings.TrimSpace(item.Waybill)
	wzaID := strings.TrimSpace(item.ShipmentID)
	for i := range oms {
		s := &oms[i]
		if !isAllegroWzAProvider(s.Provider) {
			continue
		}
		if wzaID != "" && s.ExternalID != nil && strings.TrimSpace(*s.ExternalID) == wzaID {
			return s
		}
		if waybill != "" && s.TrackingNumber != nil && strings.TrimSpace(*s.TrackingNumber) == waybill {
			return s
		}
	}
	return nil
}

func isAllegroWzAProvider(provider string) bool {
	p := strings.ToLower(strings.TrimSpace(provider))
	return p == "allegro" || strings.HasPrefix(p, "allegro:")
}

func wzaImportProvider(_ allegrosdk.ExistingWzAShipment) string {
	return "allegro"
}

func wzaImportLabelFilename(item allegrosdk.ExistingWzAShipment) string {
	if id := strings.TrimSpace(item.ShipmentID); id != "" {
		return "wza-" + id + ".pdf"
	}
	return "wza-" + strings.TrimSpace(item.Waybill) + ".pdf"
}

func wzaExternalID(item allegrosdk.ExistingWzAShipment) string {
	if id := strings.TrimSpace(item.ShipmentID); id != "" {
		return id
	}
	return strings.TrimSpace(item.Waybill)
}

// SetLabelStore wires object storage used to persist imported WzA PDFs so
// GET /v1/shipments/{id}/label can serve them through the logged-in API.
func (h *AllegroShipmentHandler) SetLabelStore(store storage.ObjectStorage, baseURL string) {
	h.objectStorage = store
	h.baseURL = baseURL
}

// ImportExistingShipments copies an already-created WzA shipment into OMS.
// POST /v1/integrations/allegro/orders/{orderId}/wza-shipments/import
// Read-only against Allegro create APIs: never POST create-commands.
func (h *AllegroShipmentHandler) ImportExistingShipments(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	orderIDParam := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if orderIDParam == "" {
		writeError(w, http.StatusBadRequest, "Missing order ID")
		return
	}

	provider, integration, err := h.getProviderWithIntegration(r.Context(), tenantID)
	if err != nil {
		slog.Error("allegro wza import: failed to get provider", "error", err)
		writeError(w, http.StatusBadRequest, "Failed to connect to Allegro. Check integration configuration.")
		return
	}
	defer provider.Close()

	checkoutFormID := h.resolveCheckoutFormID(r.Context(), tenantID, &orderIDParam)
	if checkoutFormID == "" {
		writeError(w, http.StatusBadRequest, "Missing Allegro checkout form ID")
		return
	}

	omsOrderID := h.resolveOMSOrderID(r.Context(), tenantID, orderIDParam, checkoutFormID)
	if omsOrderID == uuid.Nil {
		writeError(w, http.StatusNotFound, "Order not found")
		return
	}

	found, err := provider.FindExistingWzA(r.Context(), checkoutFormID)
	if err != nil {
		if errors.Is(err, allegrosdk.ErrWzANoExistingShipment) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		slog.Error("allegro wza import: failed to list existing shipments", "error", err)
		writeAllegroError(w, "Failed to read existing Wysyłam z Allegro shipments", err)
		return
	}

	omsShipments, err := h.shipmentService.ListByOrder(r.Context(), tenantID, omsOrderID)
	if err != nil {
		slog.Error("allegro wza import: failed to list OMS shipments", "error", err)
		writeError(w, http.StatusInternalServerError, "Failed to list order shipments")
		return
	}

	plan := planWzAImport(omsShipments, found)
	var results []wzaImportResult
	for _, existing := range plan.Already {
		results = append(results, wzaImportResultFromOMS(existing, false))
	}
	for _, item := range plan.Creates {
		created, createErr := h.persistImportedWzA(r.Context(), tenantID, actorID, integration, omsOrderID, item, provider)
		if createErr != nil {
			slog.Error("allegro wza import: failed to persist OMS shipment",
				"error", createErr,
				"waybill", item.Waybill,
				"allegro_shipment_id", item.ShipmentID,
			)
			writeError(w, http.StatusInternalServerError, "Failed to store imported shipment")
			return
		}
		results = append(results, wzaImportResultFromOMS(*created, true))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"shipments": results,
	})
}

type wzaImportResult struct {
	ID                string `json:"id"`
	Waybill           string `json:"waybill,omitempty"`
	AllegroShipmentID string `json:"allegro_shipment_id,omitempty"`
	Provider          string `json:"provider"`
	LabelReady        bool   `json:"label_ready"`
	Created           bool   `json:"created"`
}

func wzaImportResultFromOMS(s model.Shipment, created bool) wzaImportResult {
	res := wzaImportResult{
		ID:         s.ID.String(),
		Provider:   s.Provider,
		LabelReady: s.LabelURL != nil && *s.LabelURL != "",
		Created:    created,
	}
	if s.TrackingNumber != nil {
		res.Waybill = *s.TrackingNumber
	}
	if s.ExternalID != nil {
		res.AllegroShipmentID = *s.ExternalID
	}
	return res
}

func (h *AllegroShipmentHandler) resolveOMSOrderID(ctx context.Context, tenantID uuid.UUID, orderIDParam, checkoutFormID string) uuid.UUID {
	if parsed, err := uuid.Parse(strings.TrimSpace(orderIDParam)); err == nil {
		if h.pool != nil && h.orderRepo != nil {
			var found uuid.UUID
			_ = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
				order, findErr := h.orderRepo.FindByID(ctx, tx, parsed)
				if findErr != nil || order == nil {
					return findErr
				}
				found = order.ID
				return nil
			})
			if found != uuid.Nil {
				return found
			}
		} else {
			return parsed
		}
	}
	if h.pool == nil || h.orderRepo == nil {
		return uuid.Nil
	}
	var found uuid.UUID
	_ = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		order, findErr := h.orderRepo.FindByExternalID(ctx, tx, "allegro", checkoutFormID)
		if findErr != nil || order == nil {
			return findErr
		}
		found = order.ID
		return nil
	})
	return found
}

func (h *AllegroShipmentHandler) persistImportedWzA(
	ctx context.Context,
	tenantID, actorID uuid.UUID,
	integration *model.Integration,
	orderID uuid.UUID,
	item allegrosdk.ExistingWzAShipment,
	provider interface {
		GetLabel(context.Context, []string) ([]byte, error)
	},
) (*model.Shipment, error) {
	waybill := strings.TrimSpace(item.Waybill)
	externalID := wzaExternalID(item)
	carrierData, _ := json.Marshal(map[string]any{
		"allegro_shipment_id": item.ShipmentID,
		"waybill":             waybill,
		"carrier_id":          item.CarrierID,
		"carrier":             item.Carrier,
		"delivery_method_id":  item.DeliveryMethodID,
		"managed_by":          "allegro",
		"billing":             "allegro_wza",
		"imported":            true,
	})

	var labelURL *string
	if item.ShipmentID != "" && provider != nil && h.objectStorage != nil {
		pdf, labelErr := provider.GetLabel(ctx, []string{item.ShipmentID})
		if labelErr != nil {
			slog.Warn("allegro wza import: existing label fetch failed",
				"error", labelErr,
				"allegro_shipment_id", item.ShipmentID,
			)
		} else if len(pdf) > 0 {
			stored, storeErr := service.PersistShipmentLabel(
				ctx, h.objectStorage, h.baseURL, tenantID,
				wzaImportLabelFilename(item), "application/pdf", pdf,
			)
			if storeErr != nil {
				slog.Warn("allegro wza import: failed to store label PDF", "error", storeErr)
			} else {
				labelURL = &stored
			}
		}
	}

	var integrationID *uuid.UUID
	if integration != nil {
		id := integration.ID
		integrationID = &id
	}

	req := model.CreateShipmentRequest{
		OrderID:        orderID,
		Provider:       wzaImportProvider(item),
		IntegrationID:  integrationID,
		ExternalID:     &externalID,
		TrackingNumber: &waybill,
		LabelURL:       labelURL,
		CarrierData:    carrierData,
	}

	created, err := h.shipmentService.Create(ctx, tenantID, req, actorID, "allegro-wza-import")
	if err != nil {
		return nil, err
	}
	if labelURL != nil && h.shipmentRepo != nil && h.pool != nil {
		_ = database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
			return h.shipmentRepo.UpdateStatus(ctx, tx, created.ID, "label_ready")
		})
		created.Status = "label_ready"
		created.LabelURL = labelURL
	}
	return created, nil
}
