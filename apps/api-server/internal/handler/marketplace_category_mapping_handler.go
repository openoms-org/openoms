package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// MarketplaceCategoryMappingHandler handles CRUD for marketplace category mappings scoped to an integration.
type MarketplaceCategoryMappingHandler struct {
	repo repository.MarketplaceCategoryMappingRepo
	pool *pgxpool.Pool
}

// NewMarketplaceCategoryMappingHandler creates a new handler instance.
func NewMarketplaceCategoryMappingHandler(repo repository.MarketplaceCategoryMappingRepo, pool *pgxpool.Pool) *MarketplaceCategoryMappingHandler {
	return &MarketplaceCategoryMappingHandler{repo: repo, pool: pool}
}

// List handles GET /v1/integrations/{id}/category-mappings — returns all mappings for an integration.
func (h *MarketplaceCategoryMappingHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	integrationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration ID")
		return
	}

	var mappings []model.MarketplaceCategoryMapping
	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		mappings, e = h.repo.ListByIntegration(r.Context(), tx, integrationID)
		return e
	})
	if err != nil {
		writeServerError(w, "failed to list category mappings", err)
		return
	}

	// Return empty array instead of null
	if mappings == nil {
		mappings = []model.MarketplaceCategoryMapping{}
	}
	writeJSON(w, http.StatusOK, mappings)
}

// Upsert handles PUT /v1/integrations/{id}/category-mappings — creates or updates a mapping.
func (h *MarketplaceCategoryMappingHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	integrationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration ID")
		return
	}

	var req model.UpsertMarketplaceCategoryMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	mapping := model.MarketplaceCategoryMapping{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		IntegrationID:        integrationID,
		ExternalCategoryID:   req.ExternalCategoryID,
		ExternalCategoryName: req.ExternalCategoryName,
		CategoryID:           req.CategoryID,
		Confirmed:            req.Confirmed,
	}

	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.repo.Upsert(r.Context(), tx, &mapping)
	})
	if err != nil {
		writeServerError(w, "failed to upsert category mapping", err)
		return
	}

	writeJSON(w, http.StatusOK, mapping)
}

// Delete handles DELETE /v1/integrations/{id}/category-mappings/{mid} — removes a mapping.
func (h *MarketplaceCategoryMappingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	integrationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid integration ID")
		return
	}

	mappingID, err := uuid.Parse(chi.URLParam(r, "mid"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mapping ID")
		return
	}

	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		return h.repo.Delete(r.Context(), tx, integrationID, mappingID)
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "category mapping not found")
			return
		}
		writeServerError(w, "failed to delete category mapping", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
