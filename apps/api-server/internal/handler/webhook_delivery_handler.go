package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

type WebhookDeliveryHandler struct {
	deliveryRepo repository.WebhookDeliveryRepo
	pool         *pgxpool.Pool
}

func NewWebhookDeliveryHandler(deliveryRepo repository.WebhookDeliveryRepo, pool *pgxpool.Pool) *WebhookDeliveryHandler {
	return &WebhookDeliveryHandler{deliveryRepo: deliveryRepo, pool: pool}
}

func (h *WebhookDeliveryHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.WebhookDeliveryFilter{
		PaginationParams: pagination,
	}

	if v := r.URL.Query().Get("event_type"); v != "" {
		filter.EventType = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	var deliveries []model.WebhookDelivery
	var total int

	err := database.WithTenant(r.Context(), h.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		deliveries, total, e = h.deliveryRepo.List(r.Context(), tx, filter)
		return e
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve webhook deliveries")
		return
	}

	if deliveries == nil {
		deliveries = []model.WebhookDelivery{}
	}

	writeJSON(w, http.StatusOK, model.ListResponse[model.WebhookDelivery]{
		Items:  deliveries,
		Total:  total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}
