package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type IntegrationHandler struct {
	integrationService *service.IntegrationService
	integrationRepo    repository.IntegrationRepo
	pool               *pgxpool.Pool
}

func NewIntegrationHandler(integrationService *service.IntegrationService, integrationRepo repository.IntegrationRepo, pool *pgxpool.Pool) *IntegrationHandler {
	return &IntegrationHandler{
		integrationService: integrationService,
		integrationRepo:    integrationRepo,
		pool:               pool,
	}
}

func (h *IntegrationHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	integrations, err := h.integrationService.List(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list integrations")
		return
	}
	if integrations == nil {
		integrations = []model.Integration{}
	}
	writeJSON(w, http.StatusOK, integrations)
}

func (h *IntegrationHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	integrationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration ID")
		return
	}

	integration, err := h.integrationService.Get(r.Context(), tenantID, integrationID)
	if err != nil {
		if errors.Is(err, service.ErrIntegrationNotFound) {
			writeError(w, http.StatusNotFound, "integration not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get integration")
		}
		return
	}
	writeJSON(w, http.StatusOK, integration)
}

func (h *IntegrationHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	integration, err := h.integrationService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateProvider):
			writeError(w, http.StatusConflict, "integration for this provider already exists")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to create integration")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, integration)
}

func (h *IntegrationHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	integrationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration ID")
		return
	}

	var req model.UpdateIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	integration, err := h.integrationService.Update(r.Context(), tenantID, integrationID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIntegrationNotFound):
			writeError(w, http.StatusNotFound, "integration not found")
		case errors.Is(err, service.ErrDuplicateProvider):
			writeError(w, http.StatusConflict, "integration for this provider already exists")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update integration")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, integration)
}

func (h *IntegrationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	integrationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration ID")
		return
	}

	err = h.integrationService.Delete(r.Context(), tenantID, integrationID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIntegrationNotFound):
			writeError(w, http.StatusNotFound, "integration not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete integration")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IntegrationHandler) GetGeowidgetToken(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	var token string
	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		integration, err := h.integrationRepo.FindByProvider(r.Context(), tx, "inpost")
		if err != nil {
			return err
		}
		if integration == nil {
			return nil
		}
		if len(integration.Settings) == 0 {
			return nil
		}
		var settings map[string]interface{}
		if err := json.Unmarshal(integration.Settings, &settings); err != nil {
			return nil
		}
		if v, ok := settings["geowidget_token"].(string); ok {
			token = v
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get geowidget token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"geowidget_token": token})
}
