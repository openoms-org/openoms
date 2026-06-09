package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ExternalWorkflowCallbackHandler serves the external engine's callback. Auth = bearer token
// (hashed lookup) + HMAC body signature + single-use correlation nonce. No JWT.
type ExternalWorkflowCallbackHandler struct {
	svc *service.ExternalWorkflowService
}

// NewExternalWorkflowCallbackHandler constructs the handler.
func NewExternalWorkflowCallbackHandler(svc *service.ExternalWorkflowService) *ExternalWorkflowCallbackHandler {
	return &ExternalWorkflowCallbackHandler{svc: svc}
}

// Callback handles POST /v1/external-workflows/callback.
func (h *ExternalWorkflowCallbackHandler) Callback(w http.ResponseWriter, r *http.Request) {
	bearer := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if bearer == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "body too large")
		return
	}
	var req model.ExternalWorkflowCallbackRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || req.CorrelationNonce == "" {
		writeError(w, http.StatusBadRequest, "invalid callback body")
		return
	}
	// The service resolves the token (hashed, cross-tenant) -> the integration, verifies the
	// HMAC over the raw body against the integration secret (constant-time), then resolves the
	// correlated event + at most one in-scope follow-on command.
	tenantID, err := h.svc.ResolveCallbackHTTP(r.Context(), model.HashExternalWorkflowToken(bearer),
		r.Header.Get("X-Signature-256"), body, req, clientIP(r), time.Now())
	switch {
	case errors.Is(err, repository.ErrExternalWorkflowTokenNotFound):
		writeError(w, http.StatusUnauthorized, "invalid token")
	case errors.Is(err, service.ErrExternalWorkflowBadSignature):
		writeError(w, http.StatusUnauthorized, "invalid signature")
	case errors.Is(err, service.ErrCorrelationNotResolvable):
		writeError(w, http.StatusConflict, "correlation not resolvable")
	case err != nil && strings.Contains(err.Error(), "not permitted"):
		writeError(w, http.StatusForbidden, "command not permitted")
	case err != nil:
		writeServerError(w, "external workflow callback failed", err)
	default:
		_ = tenantID
		writeJSON(w, http.StatusOK, map[string]string{"result": "resolved"})
	}
}
