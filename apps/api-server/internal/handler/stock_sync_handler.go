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
		writeServerError(w, "failed to retrieve sync channels", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateChannel handles POST /v1/stock-sync/channels.
func (h *StockSyncHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var req model.CreateStockSyncChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ch, err := h.stockSyncService.CreateChannel(r.Context(), tenantID, req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeServerError(w, "failed to create sync channel", err)
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

// GetChannel handles GET /v1/stock-sync/channels/{id}.
func (h *StockSyncHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel ID")
		return
	}

	ch, err := h.stockSyncService.GetChannel(r.Context(), tenantID, channelID)
	if err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "sync channel not found")
			return
		}
		writeServerError(w, "failed to retrieve channel", err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// UpdateChannel handles PUT /v1/stock-sync/channels/{id}.
func (h *StockSyncHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel ID")
		return
	}

	var req model.UpdateStockSyncChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.stockSyncService.UpdateChannel(r.Context(), tenantID, channelID, req); err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "sync channel not found")
			return
		}
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeServerError(w, "failed to update channel", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteChannel handles DELETE /v1/stock-sync/channels/{id}.
func (h *StockSyncHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel ID")
		return
	}

	if err := h.stockSyncService.DeleteChannel(r.Context(), tenantID, channelID); err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "sync channel not found")
			return
		}
		writeServerError(w, "failed to delete channel", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PushAll handles POST /v1/stock-sync/push — manual push all products to all channels.
func (h *StockSyncHandler) PushAll(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	synced, err := h.stockSyncService.PushAll(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to start sync", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channels_synced": synced,
		"message":         "sync started",
	})
}

// PushChannel handles POST /v1/stock-sync/push/channel/{channel_id} — push all stock for a specific channel.
func (h *StockSyncHandler) PushChannel(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	channelID, err := uuid.Parse(chi.URLParam(r, "channel_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid channel ID")
		return
	}

	if err := h.stockSyncService.PushStockToChannel(r.Context(), tenantID, channelID); err != nil {
		if errors.Is(err, service.ErrStockSyncChannelNotFound) {
			writeError(w, http.StatusNotFound, "sync channel not found")
			return
		}
		writeServerError(w, "failed to sync channel", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "channel sync started",
	})
}

// PushProduct handles POST /v1/stock-sync/push/{product_id} — push single product.
func (h *StockSyncHandler) PushProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	if err := h.stockSyncService.PushStockToAllChannels(r.Context(), tenantID, productID); err != nil {
		writeServerError(w, "failed to sync stock", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "stock synced",
	})
}

// ReconcileProduct handles POST /v1/stock-sync/reconcile/{product_id}.
func (h *StockSyncHandler) ReconcileProduct(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	event, err := h.stockSyncService.ReconcileStock(r.Context(), tenantID, productID)
	if err != nil {
		writeServerError(w, "failed to reconcile stock", err)
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
			writeError(w, http.StatusBadRequest, "invalid product_id")
			return
		}
		filter.ProductID = &id
	}
	if s := r.URL.Query().Get("trigger_type"); s != "" {
		filter.TriggerType = &s
	}

	resp, err := h.stockSyncService.ListEvents(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to retrieve sync events", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetDashboard handles GET /v1/stock-sync/dashboard.
func (h *StockSyncHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	dash, err := h.stockSyncService.GetDashboard(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to retrieve sync status", err)
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

// PushListing handles POST /v1/stock-sync/push/listing/{listing_id} — force push single listing.
func (h *StockSyncHandler) PushListing(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	listingID, err := uuid.Parse(chi.URLParam(r, "listing_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid listing ID")
		return
	}

	if err := h.stockSyncService.PushSingleListing(r.Context(), tenantID, listingID); err != nil {
		writeServerError(w, "failed to sync listing stock", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "stock synced",
	})
}

// GetAllocations handles GET /v1/stock-sync/allocations/{product_id}.
func (h *StockSyncHandler) GetAllocations(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	allocations, err := h.stockSyncService.CalculateAvailableStock(r.Context(), tenantID, productID)
	if err != nil {
		writeServerError(w, "failed to calculate available stock", err)
		return
	}
	writeJSON(w, http.StatusOK, allocations)
}
