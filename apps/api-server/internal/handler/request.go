package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
)

// parseUUID extracts a UUID URL param, writes 400 on failure, returns ok=false.
func parseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", param))
		return uuid.Nil, false
	}
	return id, true
}

// tenantID extracts tenant UUID from JWT context.
func tenantID(r *http.Request) uuid.UUID {
	return middleware.TenantIDFromContext(r.Context())
}

// actorID extracts user UUID from JWT context.
func actorID(r *http.Request) uuid.UUID {
	return middleware.UserIDFromContext(r.Context())
}

// decodeBody decodes JSON body + calls Validate() if the type implements it.
func decodeBody[T any](w http.ResponseWriter, r *http.Request, dest *T) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	if v, ok := any(dest).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return false
		}
	}
	return true
}
