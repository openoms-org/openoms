package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type AuditHandler struct {
	auditRepo *repository.AuditRepository
	pool      *pgxpool.Pool
}

func NewAuditHandler(auditRepo *repository.AuditRepository, pool *pgxpool.Pool) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo, pool: pool}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.AuditListFilter{
		PaginationParams: pagination,
	}

	if v := r.URL.Query().Get("entity_type"); v != "" {
		filter.EntityType = &v
	}
	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = &v
	}
	if v := r.URL.Query().Get("user_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err == nil {
			filter.UserID = &uid
		}
	}

	var entries []model.AuditLogEntry
	var total int

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		entries, total, e = h.auditRepo.List(r.Context(), tx, filter)
		return e
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve audit log")
		return
	}

	if entries == nil {
		entries = []model.AuditLogEntry{}
	}

	writeJSON(w, http.StatusOK, model.ListResponse[model.AuditLogEntry]{
		Items:  entries,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}
