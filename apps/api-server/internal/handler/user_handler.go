package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Me returns the current authenticated user's profile.
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	userID := middleware.UserIDFromContext(r.Context())

	user, err := h.userService.GetCurrentUser(r.Context(), tenantID, userID)
	if err != nil {
		writeServerError(w, "failed to get user", err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// List returns all users for the authenticated tenant.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	users, err := h.userService.ListUsers(r.Context(), tenantID)
	if err != nil {
		writeServerError(w, "failed to list users", err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// Create creates a new user in the authenticated tenant.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Inject plan limit for atomic check inside service transaction
	if limits := middleware.PlanLimitsFromContext(r.Context()); limits != nil && limits.MaxUsers > 0 {
		req.MaxUsers = limits.MaxUsers
	}

	user, err := h.userService.CreateUser(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserLimitExceeded):
			writeError(w, http.StatusForbidden, fmt.Sprintf(
				"Osiągnięto limit użytkowników w planie (max: %d). Zmień plan aby dodać więcej użytkowników.", req.MaxUsers))
		case errors.Is(err, service.ErrDuplicateEmail):
			writeError(w, http.StatusConflict, "email already exists in this tenant")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to create user", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// Update updates a user's role or name.
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())
	claims := middleware.ClaimsFromContext(r.Context())

	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	actorRole := ""
	if claims != nil {
		actorRole = claims.Role
	}

	user, err := h.userService.UpdateUser(r.Context(), tenantID, targetID, req, actorID, actorRole, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, service.ErrOwnerRoleEscalation):
			writeError(w, http.StatusForbidden, "only owners can assign the owner role")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeServerError(w, "failed to update user", err)
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Delete removes a user from the authenticated tenant.
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	err = h.userService.DeleteUser(r.Context(), tenantID, targetID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCannotDeleteSelf):
			writeError(w, http.StatusBadRequest, "cannot delete your own account")
		case errors.Is(err, service.ErrCannotDeleteLastOwner):
			writeError(w, http.StatusBadRequest, "cannot delete the last owner of the tenant")
		case errors.Is(err, service.ErrUserNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		default:
			writeServerError(w, "failed to delete user", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
