package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// RoleHandler handles HTTP requests for role CRUD.
type RoleHandler struct {
	roleService *service.RoleService
}

// NewRoleHandler creates a new RoleHandler.
func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

// List returns all roles for the current tenant.
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	filter := model.RoleListFilter{
		PaginationParams: pagination,
	}

	resp, err := h.roleService.List(r.Context(), tenantID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list roles")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get retrieves a single role by ID.
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	role, err := h.roleService.Get(r.Context(), tenantID, roleID)
	if err != nil {
		if errors.Is(err, service.ErrRoleNotFound) {
			writeError(w, http.StatusNotFound, "role not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get role")
		}
		return
	}
	writeJSON(w, http.StatusOK, role)
}

// Create creates a new role.
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	var req model.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.roleService.Create(r.Context(), tenantID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleDuplicateName):
			writeError(w, http.StatusConflict, "role with this name already exists")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to create role")
			}
		}
		return
	}
	writeJSON(w, http.StatusCreated, role)
}

// Update updates an existing role.
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	var req model.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.roleService.Update(r.Context(), tenantID, roleID, req, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, "role not found")
		case errors.Is(err, service.ErrRoleDuplicateName):
			writeError(w, http.StatusConflict, "role with this name already exists")
		default:
			if isValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "failed to update role")
			}
		}
		return
	}
	writeJSON(w, http.StatusOK, role)
}

// Delete removes a role.
func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	actorID := middleware.UserIDFromContext(r.Context())

	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid role ID")
		return
	}

	err = h.roleService.Delete(r.Context(), tenantID, roleID, actorID, clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, "role not found")
		case errors.Is(err, service.ErrRoleIsSystem):
			writeError(w, http.StatusForbidden, "system roles cannot be deleted")
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete role")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListPermissions returns all available permissions with group metadata.
func (h *RoleHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	type permissionGroup struct {
		Group       string   `json:"group"`
		Permissions []string `json:"permissions"`
	}

	groups := []permissionGroup{
		{Group: "Zamówienia", Permissions: model.PermissionGroups["Zamówienia"]},
		{Group: "Produkty", Permissions: model.PermissionGroups["Produkty"]},
		{Group: "Przesyłki", Permissions: model.PermissionGroups["Przesyłki"]},
		{Group: "Zwroty", Permissions: model.PermissionGroups["Zwroty"]},
		{Group: "Klienci", Permissions: model.PermissionGroups["Klienci"]},
		{Group: "Faktury", Permissions: model.PermissionGroups["Faktury"]},
		{Group: "Administracja", Permissions: model.PermissionGroups["Administracja"]},
	}

	writeJSON(w, http.StatusOK, groups)
}
