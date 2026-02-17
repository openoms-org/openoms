package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// StockSyncHandler handles stock synchronization HTTP endpoints.
type StockSyncHandler struct {
	stockSyncService *service.StockSyncService
}

// NewStockSyncHandler creates a new StockSyncHandler.
func NewStockSyncHandler(stockSyncService *service.StockSyncService) *StockSyncHandler {
	return &StockSyncHandler{stockSyncService: stockSyncService}
}

// ListChannels handles GET /v1/stock-sync/channels.
func (h *StockSyncHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.StockSyncChannelListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("enabled"); s != "" {
		enabled := s == "true"
		filter.Enabled = &enabled
	}
	if s := r.URL.Query().Get("channel_type"); s != "" {
		filter.ChannelType = &s
	}

	resp, err := h.stockSyncService.ListChannels(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się pobrać kanałów synchronizacji")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateChannel handles POST /v1/stock-sync/channels.
func (h *StockSyncHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.CreateStockSyncChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe dane żądania")
		return
	}

	ch, err := h.stockSyncService.CreateChannel(r.Context(), tenantID, req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "nie udało się utworzyć kanału synchronizacji")
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

// GetChannel handles GET /v1/stock-sync/channels/{id}.
func (h *StockSyncHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe ID kanału")
		return
	}

	ch, err := h.stockSyncService.GetChannel(r.Context(), tenantID, channelID)
	if err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "kanał synchronizacji nie znaleziony")
			return
		}
		writeError(w, http.StatusInternalServerError, "nie udało się pobrać kanału")
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// UpdateChannel handles PUT /v1/stock-sync/channels/{id}.
func (h *StockSyncHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe ID kanału")
		return
	}

	var req model.UpdateStockSyncChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe dane żądania")
		return
	}

	if err := h.stockSyncService.UpdateChannel(r.Context(), tenantID, channelID, req); err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "kanał synchronizacji nie znaleziony")
			return
		}
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "nie udało się zaktualizować kanału")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteChannel handles DELETE /v1/stock-sync/channels/{id}.
func (h *StockSyncHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe ID kanału")
		return
	}

	if err := h.stockSyncService.DeleteChannel(r.Context(), tenantID, channelID); err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "kanał synchronizacji nie znaleziony")
			return
		}
		writeError(w, http.StatusInternalServerError, "nie udało się usunąć kanału")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PushAll handles POST /v1/stock-sync/push — manual push all products to all channels.
func (h *StockSyncHandler) PushAll(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	synced, err := h.stockSyncService.PushAll(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się uruchomić synchronizacji")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channels_synced": synced,
		"message":         "synchronizacja rozpoczęta",
	})
}

// PushProduct handles POST /v1/stock-sync/push/{product_id} — push single product.
func (h *StockSyncHandler) PushProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe ID produktu")
		return
	}

	if err := h.stockSyncService.PushStockToAllChannels(r.Context(), tenantID, productID); err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się zsynchronizować stanów")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "stany zsynchronizowane",
	})
}

// ReconcileProduct handles POST /v1/stock-sync/reconcile/{product_id}.
func (h *StockSyncHandler) ReconcileProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe ID produktu")
		return
	}

	event, err := h.stockSyncService.ReconcileStock(r.Context(), tenantID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się uzgodnić stanów")
		return
	}
	writeJSON(w, http.StatusOK, event)
}

// ListEvents handles GET /v1/stock-sync/events.
func (h *StockSyncHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.StockSyncEventListFilter{
		PaginationParams: pagination,
	}
	if s := r.URL.Query().Get("product_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "nieprawidłowe product_id")
			return
		}
		filter.ProductID = &id
	}
	if s := r.URL.Query().Get("trigger_type"); s != "" {
		filter.TriggerType = &s
	}

	resp, err := h.stockSyncService.ListEvents(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się pobrać zdarzeń synchronizacji")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetDashboard handles GET /v1/stock-sync/dashboard.
func (h *StockSyncHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	dash, err := h.stockSyncService.GetDashboard(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się pobrać statusu synchronizacji")
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

// GetAllocations handles GET /v1/stock-sync/allocations/{product_id}.
func (h *StockSyncHandler) GetAllocations(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "nieprawidłowe ID produktu")
		return
	}

	allocations, err := h.stockSyncService.CalculateAvailableStock(r.Context(), tenantID, productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nie udało się obliczyć dostępnych stanów")
		return
	}
	writeJSON(w, http.StatusOK, allocations)
}
