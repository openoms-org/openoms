package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type OrderGroupHandler struct {
	orderGroupService *service.OrderGroupService
}

func NewOrderGroupHandler(orderGroupService *service.OrderGroupService) *OrderGroupHandler {
	return &OrderGroupHandler{orderGroupService: orderGroupService}
}

func (h *OrderGroupHandler) MergeOrders(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.MergeOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	group, err := h.orderGroupService.MergeOrders(r.Context(), tenantID, actorID, req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to merge orders")
		}
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h *OrderGroupHandler) SplitOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req model.SplitOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	group, err := h.orderGroupService.SplitOrder(r.Context(), tenantID, actorID, orderID, req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to split order")
		}
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h *OrderGroupHandler) ListByOrder(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	groups, err := h.orderGroupService.ListByOrderID(r.Context(), tenantID, orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list order groups")
		return
	}
	writeJSON(w, http.StatusOK, groups)
}
