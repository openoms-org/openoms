package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// LoyaltyHandler handles HTTP requests for loyalty program management.
type LoyaltyHandler struct {
	loyaltyService *service.LoyaltyService
}

// NewLoyaltyHandler creates a new LoyaltyHandler.
func NewLoyaltyHandler(loyaltyService *service.LoyaltyService) *LoyaltyHandler {
	return &LoyaltyHandler{loyaltyService: loyaltyService}
}

// ListPrograms returns all loyalty programs for the tenant.
func (h *LoyaltyHandler) ListPrograms(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.LoyaltyProgramListFilter{
		PaginationParams: pagination,
	}

	resp, err := h.loyaltyService.ListPrograms(r.Context(), tenantID, filter)
	if err != nil {
		writeServerError(w, "failed to list loyalty programs", err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetProgram returns a single loyalty program by ID.
func (h *LoyaltyHandler) GetProgram(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	programID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid program ID")
		return
	}

	program, err := h.loyaltyService.GetProgram(r.Context(), tenantID, programID)
	if err != nil {
		if errors.Is(err, service.ErrLoyaltyProgramNotFound) {
			writeError(w, http.StatusNotFound, "loyalty program not found")
		} else {
			writeServerError(w, "failed to get loyalty program", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, program)
}

// CreateProgram inserts a new loyalty program.
func (h *LoyaltyHandler) CreateProgram(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateLoyaltyProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	program, err := h.loyaltyService.CreateProgram(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
		} else {
			writeServerError(w, "failed to create loyalty program", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, program)
}

// UpdateProgram modifies an existing loyalty program.
func (h *LoyaltyHandler) UpdateProgram(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	programID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid program ID")
		return
	}

	var req model.UpdateLoyaltyProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	program, err := h.loyaltyService.UpdateProgram(r.Context(), tenantID, programID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLoyaltyProgramNotFound):
			writeError(w, http.StatusNotFound, "loyalty program not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update loyalty program", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, program)
}

// DeleteProgram removes a loyalty program by ID.
func (h *LoyaltyHandler) DeleteProgram(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	programID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid program ID")
		return
	}

	err = h.loyaltyService.DeleteProgram(r.Context(), tenantID, programID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLoyaltyProgramNotFound):
			writeError(w, http.StatusNotFound, "loyalty program not found")
		default:
			writeServerError(w, "failed to delete loyalty program", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetCustomerLoyaltyStatus returns a customer's current points and tier.
func (h *LoyaltyHandler) GetCustomerLoyaltyStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	customerID, err := uuid.Parse(chi.URLParam(r, "customer_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}

	results, err := h.loyaltyService.GetCustomerLoyaltyStatus(r.Context(), tenantID, customerID)
	if err != nil {
		writeServerError(w, "failed to get customer loyalty status", err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// AwardPoints manually grants loyalty points to a customer.
func (h *LoyaltyHandler) AwardPoints(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	programID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid program ID")
		return
	}

	var req model.AwardPointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.loyaltyService.AwardPoints(r.Context(), tenantID, programID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLoyaltyProgramNotFound):
			writeError(w, http.StatusNotFound, "loyalty program not found")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to award points", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RedeemPoints redeems loyalty points for a discount or reward.
func (h *LoyaltyHandler) RedeemPoints(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	programID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid program ID")
		return
	}

	var req model.RedeemPointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.loyaltyService.RedeemPoints(r.Context(), tenantID, programID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLoyaltyProgramNotFound):
			writeError(w, http.StatusNotFound, "loyalty program not found")
		case errors.Is(err, service.ErrInsufficientPoints):
			writeError(w, http.StatusBadRequest, "insufficient points")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to redeem points", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetLeaderboard returns the top customers ranked by loyalty points.
func (h *LoyaltyHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	programID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid program ID")
		return
	}

	limit := 20
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	entries, err := h.loyaltyService.GetLeaderboard(r.Context(), tenantID, programID, limit)
	if err != nil {
		if errors.Is(err, service.ErrLoyaltyProgramNotFound) {
			writeError(w, http.StatusNotFound, "loyalty program not found")
		} else {
			writeServerError(w, "failed to get leaderboard", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
