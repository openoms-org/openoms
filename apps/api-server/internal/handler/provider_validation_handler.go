package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ProviderValidation is the subset of the validation service the handler needs.
type ProviderValidation interface {
	SetProbes(ctx context.Context, versionID uuid.UUID, probes []model.ProviderValidationProbe) ([]model.ProviderValidationProbe, error)
	GetProbes(ctx context.Context, versionID uuid.UUID) ([]model.ProviderValidationProbe, error)
	StartRun(ctx context.Context, versionID uuid.UUID, environment string, allowDestructive bool, actorID *uuid.UUID) (*model.ProviderValidationRun, error)
	RecordResult(ctx context.Context, runID uuid.UUID, probeType, label, status, observation, payloadHash, findings string) (*model.ProviderValidationResult, error)
	CompleteRun(ctx context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error)
	ListRuns(ctx context.Context, versionID uuid.UUID) ([]model.ProviderValidationRun, error)
	GetRunWithResults(ctx context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error)
}

// ProviderValidationHandler serves the validation-engine endpoints under
// /v1/platform/providers/{id}/versions/{version_id}. Mutations are audited.
type ProviderValidationHandler struct {
	svc   ProviderValidation
	audit PlatformAuditLogger
}

// NewProviderValidationHandler creates a ProviderValidationHandler. audit may be nil.
func NewProviderValidationHandler(svc ProviderValidation, audit PlatformAuditLogger) *ProviderValidationHandler {
	return &ProviderValidationHandler{svc: svc, audit: audit}
}

func (h *ProviderValidationHandler) actor(r *http.Request) *uuid.UUID {
	if a := middleware.PlatformAdminFromContext(r.Context()); a != nil {
		id := a.UserID
		return &id
	}
	return nil
}

func (h *ProviderValidationHandler) logAudit(r *http.Request, action, targetID string, metadata any) {
	if h.audit == nil {
		return
	}
	var actor uuid.UUID
	if a := h.actor(r); a != nil {
		actor = *a
	}
	if err := h.audit.Log(r.Context(), model.PlatformAuditEntry{
		ActorUserID: actor, Action: action, TargetType: "provider_version", TargetID: targetID,
		Metadata: metadata, IPAddress: clientIP(r),
	}); err != nil {
		slog.Warn("failed to write platform audit entry", "error", err, "action", action)
	}
}

func (h *ProviderValidationHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProviderVersionNotFound), errors.Is(err, service.ErrValidationRunNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidProbe),
		errors.Is(err, service.ErrInvalidValidationEnv),
		errors.Is(err, service.ErrInvalidResultStatus),
		errors.Is(err, service.ErrDestructiveProbeNotConfirmed),
		errors.Is(err, service.ErrValidationRunFinalized),
		errors.Is(err, service.ErrProviderVersionFrozen):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeServerError(w, "provider validation error", err)
	}
}

type setProbesRequest struct {
	Probes []model.ProviderValidationProbe `json:"probes"`
}

// GetProbes returns a version's validation probes.
func (h *ProviderValidationHandler) GetProbes(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseIDParam(w, r, "version_id", "version")
	if !ok {
		return
	}
	probes, err := h.svc.GetProbes(r.Context(), versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"probes": probes})
}

// UpdateProbes replaces a version's validation probes.
func (h *ProviderValidationHandler) UpdateProbes(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseIDParam(w, r, "version_id", "version")
	if !ok {
		return
	}
	var req setProbesRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	probes, err := h.svc.SetProbes(r.Context(), versionID, req.Probes)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.version.probes_updated", versionID.String(), nil)
	writeJSON(w, http.StatusOK, map[string]any{"probes": probes})
}

type startRunRequest struct {
	Environment      string `json:"environment"`
	AllowDestructive bool   `json:"allow_destructive"`
}

// StartRun opens a validation run for a version.
func (h *ProviderValidationHandler) StartRun(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseIDParam(w, r, "version_id", "version")
	if !ok {
		return
	}
	var req startRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	run, err := h.svc.StartRun(r.Context(), versionID, req.Environment, req.AllowDestructive, h.actor(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.validation.started", versionID.String(), map[string]any{"environment": run.Environment, "run_id": run.ID.String()})
	writeJSON(w, http.StatusCreated, run)
}

// ListRuns returns a version's validation runs.
func (h *ProviderValidationHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	versionID, ok := parseIDParam(w, r, "version_id", "version")
	if !ok {
		return
	}
	runs, err := h.svc.ListRuns(r.Context(), versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

// GetRun returns a validation run with its probe results.
func (h *ProviderValidationHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseIDParam(w, r, "run_id", "run")
	if !ok {
		return
	}
	run, err := h.svc.GetRunWithResults(r.Context(), runID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

type recordResultRequest struct {
	ProbeType   string `json:"probe_type"`
	Label       string `json:"label"`
	Status      string `json:"status"`
	Observation string `json:"observation"`
	PayloadHash string `json:"payload_hash"`
	Findings    string `json:"findings"`
}

// RecordResult records a probe result on a pending run.
func (h *ProviderValidationHandler) RecordResult(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseIDParam(w, r, "run_id", "run")
	if !ok {
		return
	}
	var req recordResultRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	result, err := h.svc.RecordResult(r.Context(), runID, req.ProbeType, req.Label, req.Status, req.Observation, req.PayloadHash, req.Findings)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.validation.result_recorded", runID.String(), map[string]any{"label": req.Label, "status": req.Status})
	writeJSON(w, http.StatusCreated, result)
}

// CompleteRun finalizes a validation run.
func (h *ProviderValidationHandler) CompleteRun(w http.ResponseWriter, r *http.Request) {
	runID, ok := parseIDParam(w, r, "run_id", "run")
	if !ok {
		return
	}
	run, err := h.svc.CompleteRun(r.Context(), runID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.validation.completed", runID.String(), map[string]any{"verdict": run.Verdict})
	writeJSON(w, http.StatusOK, run)
}
