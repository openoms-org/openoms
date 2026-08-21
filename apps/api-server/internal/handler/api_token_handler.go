package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// APITokenHandler handles owner API token create/list/revoke.
type APITokenHandler struct {
	svc *service.APITokenService
}

// NewAPITokenHandler constructs an APITokenHandler.
func NewAPITokenHandler(svc *service.APITokenService) *APITokenHandler {
	return &APITokenHandler{svc: svc}
}

// Create issues a long-lived token. The raw secret is in the response once.
func (h *APITokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID, role, ok := apiTokenActor(w, r)
	if !ok {
		return
	}
	var req model.CreateAPITokenRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	created, err := h.svc.Create(r.Context(), tenantID, actorID, role, req, actorID, clientIP(r))
	if err != nil {
		writeAPITokenError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// List returns active tokens (id, name, created, last used — never the secret).
func (h *APITokenHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID, role, ok := apiTokenActor(w, r)
	if !ok {
		return
	}
	tokens, err := h.svc.List(r.Context(), tenantID, actorID, role)
	if err != nil {
		writeAPITokenError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

// Revoke disables a token. Subsequent Bearer requests fail.
func (h *APITokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	tenantID, actorID, role, ok := apiTokenActor(w, r)
	if !ok {
		return
	}
	id, ok := parseIDParam(w, r, "id", "token")
	if !ok {
		return
	}
	if err := h.svc.Revoke(r.Context(), tenantID, actorID, role, id, actorID, clientIP(r)); err != nil {
		writeAPITokenError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func apiTokenActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, string, bool) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, uuid.Nil, "", false
	}
	return middleware.TenantIDFromContext(r.Context()), middleware.UserIDFromContext(r.Context()), claims.Role, true
}

func writeAPITokenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAPITokenOwnerRequired):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrAPITokenNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case isValidationError(err):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeServerError(w, "failed to manage API token", err)
	}
}
