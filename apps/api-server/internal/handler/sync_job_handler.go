package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type SyncJobHandler struct {
	syncJobRepo repository.SyncJobRepo
	pool        *pgxpool.Pool
}

func NewSyncJobHandler(syncJobRepo repository.SyncJobRepo, pool *pgxpool.Pool) *SyncJobHandler {
	return &SyncJobHandler{syncJobRepo: syncJobRepo, pool: pool}
}

func (h *SyncJobHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.SyncJobListFilter{
		PaginationParams: pagination,
	}

	if v := r.URL.Query().Get("integration_id"); v != "" {
		uid, err := uuid.Parse(v)
		if err == nil {
			filter.IntegrationID = &uid
		}
	}
	if v := r.URL.Query().Get("job_type"); v != "" {
		filter.JobType = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	var jobs []*model.SyncJob
	var total int

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		jobs, total, e = h.syncJobRepo.List(r.Context(), tx, filter)
		return e
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve sync jobs")
		return
	}

	items := make([]model.SyncJob, 0, len(jobs))
	for _, j := range jobs {
		items = append(items, *j)
	}

	writeJSON(w, http.StatusOK, model.ListResponse[model.SyncJob]{
		Items:  items,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

func (h *SyncJobHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sync job ID")
		return
	}

	var job *model.SyncJob

	err = database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		job, e = h.syncJobRepo.GetByID(r.Context(), tx, jobID)
		return e
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve sync job")
		return
	}
	if job == nil {
		writeError(w, http.StatusNotFound, "sync job not found")
		return
	}

	writeJSON(w, http.StatusOK, job)
}
