package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type HelpdeskHandler struct {
	freshdeskService *service.FreshdeskService
}

func NewHelpdeskHandler(freshdeskService *service.FreshdeskService) *HelpdeskHandler {
	return &HelpdeskHandler{freshdeskService: freshdeskService}
}

// ListOrderTickets handles GET /v1/orders/{id}/tickets
func (h *HelpdeskHandler) ListOrderTickets(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	tickets, err := h.freshdeskService.GetTickets(r.Context(), tenantID, orderID)
	if err != nil {
		if errors.Is(err, service.ErrFreshdeskNotConfigured) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"tickets": []service.FreshdeskTicket{},
			})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if tickets == nil {
		tickets = []service.FreshdeskTicket{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tickets": tickets,
	})
}

// CreateOrderTicket handles POST /v1/orders/{id}/tickets
func (h *HelpdeskHandler) CreateOrderTicket(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	var req struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Email       string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Subject == "" || req.Description == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "subject, description and email are required")
		return
	}

	ticket, err := h.freshdeskService.CreateTicket(r.Context(), tenantID, orderID, req.Subject, req.Description, req.Email)
	if err != nil {
		if errors.Is(err, service.ErrFreshdeskNotConfigured) {
			writeError(w, http.StatusBadRequest, "freshdesk is not configured")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ticket)
}

// ListAllTickets handles GET /v1/helpdesk/tickets
func (h *HelpdeskHandler) ListAllTickets(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	tickets, err := h.freshdeskService.ListAllTickets(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, service.ErrFreshdeskNotConfigured) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"tickets": []service.FreshdeskTicket{},
			})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if tickets == nil {
		tickets = []service.FreshdeskTicket{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tickets": tickets,
	})
}
