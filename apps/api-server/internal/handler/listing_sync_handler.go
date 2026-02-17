package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ListingSyncHandler handles REST endpoints for listing sync configuration and operations.
type ListingSyncHandler struct {
	syncService *service.ListingSyncService
}

// NewListingSyncHandler creates a new ListingSyncHandler.
func NewListingSyncHandler(syncService *service.ListingSyncService) *ListingSyncHandler {
	return &ListingSyncHandler{syncService: syncService}
}

// ListConfigs returns paginated listing sync configs.
// GET /v1/listing-sync/configs
func (h *ListingSyncHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)
	pagination := model.ParsePagination(r)

	var integrationID *uuid.UUID
	if raw := r.URL.Query().Get("integration_id"); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			integrationID = &parsed
		}
	}

	var status *string
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = &raw
	}

	filter := model.ListingSyncConfigFilter{
		IntegrationID:    integrationID,
		Status:           status,
		PaginationParams: pagination,
	}

	resp, err := h.syncService.List(ctx, tenantID, filter)
	if err != nil {
		slog.Error("listing sync: list configs failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać konfiguracji synchronizacji")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// CreateConfig creates a new listing sync config.
// POST /v1/listing-sync/configs
func (h *ListingSyncHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	var req model.CreateListingSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := h.syncService.Create(ctx, tenantID, req)
	if err != nil {
		slog.Error("listing sync: create config failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć konfiguracji synchronizacji")
		return
	}

	writeJSON(w, http.StatusCreated, cfg)
}

// GetConfig returns a single listing sync config.
// GET /v1/listing-sync/configs/{id}
func (h *ListingSyncHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	cfg, err := h.syncService.Get(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrListingSyncConfigNotFound) {
			writeError(w, http.StatusNotFound, "Konfiguracja synchronizacji nie została znaleziona")
			return
		}
		slog.Error("listing sync: get config failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać konfiguracji")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// UpdateConfig updates a listing sync config.
// PUT /v1/listing-sync/configs/{id}
func (h *ListingSyncHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	var req model.UpdateListingSyncConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := h.syncService.Update(ctx, tenantID, id, req)
	if err != nil {
		if errors.Is(err, service.ErrListingSyncConfigNotFound) {
			writeError(w, http.StatusNotFound, "Konfiguracja synchronizacji nie została znaleziona")
			return
		}
		slog.Error("listing sync: update config failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować konfiguracji")
		return
	}

	writeJSON(w, http.StatusOK, cfg)
}

// DeleteConfig deletes a listing sync config.
// DELETE /v1/listing-sync/configs/{id}
func (h *ListingSyncHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	if err := h.syncService.Delete(ctx, tenantID, id); err != nil {
		if errors.Is(err, service.ErrListingSyncConfigNotFound) {
			writeError(w, http.StatusNotFound, "Konfiguracja synchronizacji nie została znaleziona")
			return
		}
		slog.Error("listing sync: delete config failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Nie udało się usunąć konfiguracji")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerSync triggers a full manual sync.
// POST /v1/listing-sync/configs/{id}/sync
func (h *ListingSyncHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	result, err := h.syncService.RunFullSync(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrListingSyncConfigNotFound) {
			writeError(w, http.StatusNotFound, "Konfiguracja synchronizacji nie została znaleziona")
			return
		}
		slog.Error("listing sync: trigger sync failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Synchronizacja nie powiodła się")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// TriggerSyncPrices triggers a price-only sync.
// POST /v1/listing-sync/configs/{id}/sync-prices
func (h *ListingSyncHandler) TriggerSyncPrices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	result, err := h.syncService.SyncPrices(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrListingSyncConfigNotFound) {
			writeError(w, http.StatusNotFound, "Konfiguracja synchronizacji nie została znaleziona")
			return
		}
		slog.Error("listing sync: sync prices failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Synchronizacja cen nie powiodła się")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// TriggerSyncStock triggers a stock-only sync.
// POST /v1/listing-sync/configs/{id}/sync-stock
func (h *ListingSyncHandler) TriggerSyncStock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	result, err := h.syncService.SyncStock(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, service.ErrListingSyncConfigNotFound) {
			writeError(w, http.StatusNotFound, "Konfiguracja synchronizacji nie została znaleziona")
			return
		}
		slog.Error("listing sync: sync stock failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Synchronizacja stanów magazynowych nie powiodła się")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ListLogs returns paginated sync logs for a config.
// GET /v1/listing-sync/configs/{id}/logs
func (h *ListingSyncHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromContext(ctx)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe ID")
		return
	}

	pagination := model.ParsePagination(r)

	var direction *string
	if raw := r.URL.Query().Get("direction"); raw != "" {
		direction = &raw
	}
	var entityType *string
	if raw := r.URL.Query().Get("entity_type"); raw != "" {
		entityType = &raw
	}
	var status *string
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = &raw
	}

	filter := model.ListingSyncLogFilter{
		ConfigID:         id,
		Direction:        direction,
		EntityType:       entityType,
		Status:           status,
		PaginationParams: pagination,
	}

	resp, err := h.syncService.ListLogs(ctx, tenantID, filter)
	if err != nil {
		slog.Error("listing sync: list logs failed", "error", err)
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać dziennika synchronizacji")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
