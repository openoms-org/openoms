package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// ProviderRegistry is the subset of the registry service the handler needs.
type ProviderRegistry interface {
	CreateDefinition(ctx context.Context, in service.CreateProviderDefinitionInput) (*model.ProviderDefinition, error)
	ListDefinitions(ctx context.Context) ([]model.ProviderDefinition, error)
	GetDefinition(ctx context.Context, id uuid.UUID) (*model.ProviderDefinition, error)
	UpdateDefinitionMetadata(ctx context.Context, in model.ProviderDefinition) (*model.ProviderDefinition, error)
	CreateVersion(ctx context.Context, defID uuid.UUID, version, changelog, compat string, actorID *uuid.UUID) (*model.ProviderVersion, error)
	GetVersion(ctx context.Context, id uuid.UUID) (*model.ProviderVersion, error)
	ListVersions(ctx context.Context, defID uuid.UUID) ([]model.ProviderVersion, error)
	Transition(ctx context.Context, versionID uuid.UUID, toState string, actorID *uuid.UUID, reason string) (*model.ProviderVersion, error)
	EmergencyDisable(ctx context.Context, versionID uuid.UUID, actorID *uuid.UUID, reason string) (*model.ProviderVersion, error)
	EnableTenant(ctx context.Context, versionID, tenantID uuid.UUID, actorID *uuid.UUID) (*model.ProviderTenantEnable, error)
	ListPublicationEvents(ctx context.Context, versionID uuid.UUID) ([]model.ProviderPublicationEvent, error)
}

// ProviderHandler serves the Provider Integration Studio registry endpoints
// under /v1/platform/providers (each route additionally guarded by a platform
// permission). Mutations are recorded to the platform audit log.
type ProviderHandler struct {
	svc   ProviderRegistry
	audit PlatformAuditLogger
}

// NewProviderHandler creates a ProviderHandler. audit may be nil.
func NewProviderHandler(svc ProviderRegistry, audit PlatformAuditLogger) *ProviderHandler {
	return &ProviderHandler{svc: svc, audit: audit}
}

func (h *ProviderHandler) actor(r *http.Request) *uuid.UUID {
	if a := middleware.PlatformAdminFromContext(r.Context()); a != nil {
		id := a.UserID
		return &id
	}
	return nil
}

func (h *ProviderHandler) logAudit(r *http.Request, action, targetType, targetID string, metadata any) {
	if h.audit == nil {
		return
	}
	var actor uuid.UUID
	if a := h.actor(r); a != nil {
		actor = *a
	}
	if err := h.audit.Log(r.Context(), model.PlatformAuditEntry{
		ActorUserID: actor,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Metadata:    metadata,
		IPAddress:   clientIP(r),
	}); err != nil {
		slog.Warn("failed to write platform audit entry", "error", err, "action", action)
	}
}

func (h *ProviderHandler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProviderDefinitionNotFound), errors.Is(err, service.ErrProviderVersionNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrIllegalProviderTransition),
		errors.Is(err, service.ErrProviderVersionFrozen),
		errors.Is(err, service.ErrProviderEnableStateInvalid),
		errors.Is(err, service.ErrProviderDisableStateInvalid),
		errors.Is(err, service.ErrInvalidProviderType),
		errors.Is(err, service.ErrInvalidPublicationState):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeServerError(w, "provider registry error", err)
	}
}

func pathUUID(r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// ---- Definitions ----

type createDefinitionRequest struct {
	ProviderKey     string   `json:"provider_key"`
	DisplayName     string   `json:"display_name"`
	ProviderType    string   `json:"provider_type"`
	Regions         []string `json:"regions"`
	BusinessDomains []string `json:"business_domains"`
	Owner           string   `json:"owner"`
	SourceLinks     []string `json:"source_links"`
	Notes           string   `json:"notes"`
}

// ListDefinitions returns all provider definitions.
func (h *ProviderHandler) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	defs, err := h.svc.ListDefinitions(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"definitions": defs})
}

// CreateDefinition creates a new provider family definition.
func (h *ProviderHandler) CreateDefinition(w http.ResponseWriter, r *http.Request) {
	var req createDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProviderKey == "" || req.DisplayName == "" || req.ProviderType == "" {
		writeError(w, http.StatusBadRequest, "provider_key, display_name and provider_type are required")
		return
	}
	def, err := h.svc.CreateDefinition(r.Context(), service.CreateProviderDefinitionInput{
		ProviderKey:     req.ProviderKey,
		DisplayName:     req.DisplayName,
		ProviderType:    req.ProviderType,
		Regions:         req.Regions,
		BusinessDomains: req.BusinessDomains,
		Owner:           req.Owner,
		SourceLinks:     req.SourceLinks,
		Notes:           req.Notes,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.definition.created", "provider_definition", def.ID.String(), map[string]string{"provider_key": def.ProviderKey})
	writeJSON(w, http.StatusCreated, def)
}

// GetDefinition returns a provider definition by id.
func (h *ProviderHandler) GetDefinition(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	def, err := h.svc.GetDefinition(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, def)
}

type patchDefinitionRequest struct {
	DisplayName     string   `json:"display_name"`
	ProviderType    string   `json:"provider_type"`
	Regions         []string `json:"regions"`
	BusinessDomains []string `json:"business_domains"`
	Owner           string   `json:"owner"`
	SourceLinks     []string `json:"source_links"`
	Notes           string   `json:"notes"`
}

// PatchDefinition updates editable provider definition metadata.
func (h *ProviderHandler) PatchDefinition(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	var req patchDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	def, err := h.svc.UpdateDefinitionMetadata(r.Context(), model.ProviderDefinition{
		ID:              id,
		DisplayName:     req.DisplayName,
		ProviderType:    req.ProviderType,
		Regions:         req.Regions,
		BusinessDomains: req.BusinessDomains,
		Owner:           req.Owner,
		SourceLinks:     req.SourceLinks,
		Notes:           req.Notes,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.definition.updated", "provider_definition", def.ID.String(), nil)
	writeJSON(w, http.StatusOK, def)
}

// ---- Versions ----

type createVersionRequest struct {
	Version            string `json:"version"`
	Changelog          string `json:"changelog"`
	CompatibilityNotes string `json:"compatibility_notes"`
}

// CreateVersion creates a new draft (research) version under a definition.
func (h *ProviderHandler) CreateVersion(w http.ResponseWriter, r *http.Request) {
	defID, ok := pathUUID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	var req createVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Version == "" {
		writeError(w, http.StatusBadRequest, "version is required")
		return
	}
	v, err := h.svc.CreateVersion(r.Context(), defID, req.Version, req.Changelog, req.CompatibilityNotes, h.actor(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.version.created", "provider_version", v.ID.String(), map[string]string{"version": v.Version})
	writeJSON(w, http.StatusCreated, v)
}

// ListVersions returns the versions of a provider definition.
func (h *ProviderHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	defID, ok := pathUUID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	vers, err := h.svc.ListVersions(r.Context(), defID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"versions": vers})
}

// GetVersion returns a provider version by id.
func (h *ProviderHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	versionID, ok := pathUUID(r, "version_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid version id")
		return
	}
	v, err := h.svc.GetVersion(r.Context(), versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// ListPublicationEvents returns the publication audit trail for a version.
func (h *ProviderHandler) ListPublicationEvents(w http.ResponseWriter, r *http.Request) {
	versionID, ok := pathUUID(r, "version_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid version id")
		return
	}
	events, err := h.svc.ListPublicationEvents(r.Context(), versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

type publishRequest struct {
	ToState string `json:"to_state"`
	Reason  string `json:"reason"`
}

// Publish transitions a version to the requested lifecycle state.
func (h *ProviderHandler) Publish(w http.ResponseWriter, r *http.Request) {
	versionID, ok := pathUUID(r, "version_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid version id")
		return
	}
	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ToState == "" {
		writeError(w, http.StatusBadRequest, "to_state is required")
		return
	}
	v, err := h.svc.Transition(r.Context(), versionID, req.ToState, h.actor(r), req.Reason)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.version.published", "provider_version", v.ID.String(), map[string]string{"to_state": v.PublicationState})
	writeJSON(w, http.StatusOK, v)
}

type disableRequest struct {
	Reason string `json:"reason"`
}

// EmergencyDisable pulls an available/private-beta version back to internal_validation.
func (h *ProviderHandler) EmergencyDisable(w http.ResponseWriter, r *http.Request) {
	versionID, ok := pathUUID(r, "version_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid version id")
		return
	}
	var req disableRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason optional
	v, err := h.svc.EmergencyDisable(r.Context(), versionID, h.actor(r), req.Reason)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.version.emergency_disabled", "provider_version", v.ID.String(), map[string]string{"reason": req.Reason})
	writeJSON(w, http.StatusOK, v)
}

type enableTenantRequest struct {
	TenantID string `json:"tenant_id"`
}

// EnableTenant adds a tenant to a version's private-beta allowlist.
func (h *ProviderHandler) EnableTenant(w http.ResponseWriter, r *http.Request) {
	versionID, ok := pathUUID(r, "version_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid version id")
		return
	}
	var req enableTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}
	enable, err := h.svc.EnableTenant(r.Context(), versionID, tenantID, h.actor(r))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	h.logAudit(r, "platform.provider.version.tenant_enabled", "provider_version", versionID.String(), map[string]string{"tenant_id": tenantID.String()})
	writeJSON(w, http.StatusCreated, enable)
}
