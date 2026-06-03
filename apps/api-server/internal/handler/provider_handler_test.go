package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// fakeProviderRegistry implements handler.ProviderRegistry; unset funcs return zero values.
type fakeProviderRegistry struct {
	createDef  func(service.CreateProviderDefinitionInput) (*model.ProviderDefinition, error)
	getDef     func(uuid.UUID) (*model.ProviderDefinition, error)
	createVer  func(uuid.UUID, string) (*model.ProviderVersion, error)
	transition func(uuid.UUID, string) (*model.ProviderVersion, error)
	enableTen  func(uuid.UUID, uuid.UUID) (*model.ProviderTenantEnable, error)
	setSchema  func(uuid.UUID, []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error)
	setCaps    func([]model.ProviderCapability) ([]model.ProviderCapability, error)
	setMaps    func([]model.ProviderStatusMapping) ([]model.ProviderStatusMapping, error)
	createGap  func(uuid.UUID, string, string) (*model.ProviderIntegrationGap, error)
}

func (f *fakeProviderRegistry) CreateDefinition(_ context.Context, in service.CreateProviderDefinitionInput) (*model.ProviderDefinition, error) {
	return f.createDef(in)
}
func (f *fakeProviderRegistry) ListDefinitions(context.Context) ([]model.ProviderDefinition, error) {
	return []model.ProviderDefinition{}, nil
}
func (f *fakeProviderRegistry) GetDefinition(_ context.Context, id uuid.UUID) (*model.ProviderDefinition, error) {
	return f.getDef(id)
}
func (f *fakeProviderRegistry) UpdateDefinitionMetadata(_ context.Context, in model.ProviderDefinition) (*model.ProviderDefinition, error) {
	return &in, nil
}
func (f *fakeProviderRegistry) CreateVersion(_ context.Context, defID uuid.UUID, version, _, _ string, _ *uuid.UUID) (*model.ProviderVersion, error) {
	return f.createVer(defID, version)
}
func (f *fakeProviderRegistry) GetVersion(context.Context, uuid.UUID) (*model.ProviderVersion, error) {
	return &model.ProviderVersion{ID: uuid.New()}, nil
}
func (f *fakeProviderRegistry) ListVersions(context.Context, uuid.UUID) ([]model.ProviderVersion, error) {
	return []model.ProviderVersion{}, nil
}
func (f *fakeProviderRegistry) Transition(_ context.Context, versionID uuid.UUID, toState string, _ *uuid.UUID, _ string) (*model.ProviderVersion, error) {
	return f.transition(versionID, toState)
}
func (f *fakeProviderRegistry) EmergencyDisable(context.Context, uuid.UUID, *uuid.UUID, string) (*model.ProviderVersion, error) {
	return &model.ProviderVersion{ID: uuid.New(), PublicationState: model.ProviderStateInternalValidation}, nil
}
func (f *fakeProviderRegistry) EnableTenant(_ context.Context, versionID, tenantID uuid.UUID, _ *uuid.UUID) (*model.ProviderTenantEnable, error) {
	return f.enableTen(versionID, tenantID)
}
func (f *fakeProviderRegistry) ListPublicationEvents(context.Context, uuid.UUID) ([]model.ProviderPublicationEvent, error) {
	return []model.ProviderPublicationEvent{}, nil
}
func (f *fakeProviderRegistry) SetSchema(_ context.Context, versionID uuid.UUID, groups []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error) {
	return f.setSchema(versionID, groups)
}
func (f *fakeProviderRegistry) GetSchema(_ context.Context, versionID uuid.UUID) (*model.ProviderFieldSchema, error) {
	return &model.ProviderFieldSchema{ProviderVersionID: versionID, Groups: []model.ProviderFieldGroup{}}, nil
}
func (f *fakeProviderRegistry) SetCapabilities(_ context.Context, _ uuid.UUID, caps []model.ProviderCapability) ([]model.ProviderCapability, error) {
	return f.setCaps(caps)
}
func (f *fakeProviderRegistry) GetCapabilities(context.Context, uuid.UUID) ([]model.ProviderCapability, error) {
	return []model.ProviderCapability{}, nil
}
func (f *fakeProviderRegistry) SetStatusMappings(_ context.Context, _ uuid.UUID, m []model.ProviderStatusMapping) ([]model.ProviderStatusMapping, error) {
	return f.setMaps(m)
}
func (f *fakeProviderRegistry) GetStatusMappings(context.Context, uuid.UUID) ([]model.ProviderStatusMapping, error) {
	return []model.ProviderStatusMapping{}, nil
}
func (f *fakeProviderRegistry) CreateGap(_ context.Context, versionID uuid.UUID, gapType, severity, _ string) (*model.ProviderIntegrationGap, error) {
	return f.createGap(versionID, gapType, severity)
}
func (f *fakeProviderRegistry) ListGaps(context.Context, uuid.UUID) ([]model.ProviderIntegrationGap, error) {
	return []model.ProviderIntegrationGap{}, nil
}
func (f *fakeProviderRegistry) UpdateGapStatus(_ context.Context, gapID uuid.UUID, status string) (*model.ProviderIntegrationGap, error) {
	return &model.ProviderIntegrationGap{ID: gapID, Status: status}, nil
}

func provReq(method, target, body string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestProviderHandler_CreateDefinition_Success(t *testing.T) {
	f := &fakeProviderRegistry{createDef: func(in service.CreateProviderDefinitionInput) (*model.ProviderDefinition, error) {
		return &model.ProviderDefinition{ID: uuid.New(), ProviderKey: in.ProviderKey, ProviderType: in.ProviderType}, nil
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.CreateDefinition(rr, provReq(http.MethodPost, "/v1/platform/providers",
		`{"provider_key":"bigbuy","display_name":"BigBuy","provider_type":"supplier"}`, nil))
	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestProviderHandler_CreateDefinition_InvalidType_422(t *testing.T) {
	f := &fakeProviderRegistry{createDef: func(service.CreateProviderDefinitionInput) (*model.ProviderDefinition, error) {
		return nil, service.ErrInvalidProviderType
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.CreateDefinition(rr, provReq(http.MethodPost, "/v1/platform/providers",
		`{"provider_key":"x","display_name":"X","provider_type":"banana"}`, nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_CreateDefinition_MissingFields_400(t *testing.T) {
	h := NewProviderHandler(&fakeProviderRegistry{}, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.CreateDefinition(rr, provReq(http.MethodPost, "/v1/platform/providers", `{"provider_key":"x"}`, nil))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestProviderHandler_GetDefinition_NotFound_404(t *testing.T) {
	f := &fakeProviderRegistry{getDef: func(uuid.UUID) (*model.ProviderDefinition, error) {
		return nil, service.ErrProviderDefinitionNotFound
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.GetDefinition(rr, provReq(http.MethodGet, "/", "", map[string]string{"id": uuid.New().String()}))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestProviderHandler_GetDefinition_BadUUID_400(t *testing.T) {
	h := NewProviderHandler(&fakeProviderRegistry{}, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.GetDefinition(rr, provReq(http.MethodGet, "/", "", map[string]string{"id": "not-a-uuid"}))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestProviderHandler_Publish_IllegalTransition_422(t *testing.T) {
	f := &fakeProviderRegistry{transition: func(uuid.UUID, string) (*model.ProviderVersion, error) {
		return nil, service.ErrIllegalProviderTransition
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.Publish(rr, provReq(http.MethodPost, "/publish", `{"to_state":"available"}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_Publish_Success_Audits(t *testing.T) {
	audit := &fakePlatformAudit{}
	f := &fakeProviderRegistry{transition: func(id uuid.UUID, to string) (*model.ProviderVersion, error) {
		return &model.ProviderVersion{ID: id, PublicationState: to}, nil
	}}
	h := NewProviderHandler(f, audit)
	rr := httptest.NewRecorder()
	h.Publish(rr, provReq(http.MethodPost, "/publish", `{"to_state":"designed","reason":"ready"}`,
		map[string]string{"version_id": uuid.New().String()}))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "platform.provider.version.published", audit.entries[0].Action)
}

func TestProviderHandler_EnableTenant_BadTenant_400(t *testing.T) {
	h := NewProviderHandler(&fakeProviderRegistry{}, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.EnableTenant(rr, provReq(http.MethodPost, "/enable-tenant", `{"tenant_id":"nope"}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestProviderHandler_UpdateSchema_Invalid_422(t *testing.T) {
	f := &fakeProviderRegistry{setSchema: func(uuid.UUID, []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error) {
		return nil, service.ErrInvalidFieldSchema
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.UpdateSchema(rr, provReq(http.MethodPatch, "/schema", `{"groups":[]}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_UpdateSchema_Frozen_422(t *testing.T) {
	f := &fakeProviderRegistry{setSchema: func(uuid.UUID, []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error) {
		return nil, service.ErrProviderVersionFrozen
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.UpdateSchema(rr, provReq(http.MethodPatch, "/schema", `{"groups":[]}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_UpdateCapabilities_Invalid_422(t *testing.T) {
	f := &fakeProviderRegistry{setCaps: func([]model.ProviderCapability) ([]model.ProviderCapability, error) {
		return nil, service.ErrInvalidCapability
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.UpdateCapabilities(rr, provReq(http.MethodPatch, "/capabilities", `{"capabilities":[]}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_UpdateStatusMappings_Frozen_422(t *testing.T) {
	f := &fakeProviderRegistry{setMaps: func([]model.ProviderStatusMapping) ([]model.ProviderStatusMapping, error) {
		return nil, service.ErrProviderVersionFrozen
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.UpdateStatusMappings(rr, provReq(http.MethodPatch, "/status-mappings", `{"status_mappings":[]}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_CreateGap_Success_Audits(t *testing.T) {
	audit := &fakePlatformAudit{}
	f := &fakeProviderRegistry{createGap: func(versionID uuid.UUID, gapType, severity string) (*model.ProviderIntegrationGap, error) {
		return &model.ProviderIntegrationGap{ID: uuid.New(), ProviderVersionID: versionID, GapType: gapType, Severity: severity, Status: model.GapStatusOpen}, nil
	}}
	h := NewProviderHandler(f, audit)
	rr := httptest.NewRecorder()
	h.CreateGap(rr, provReq(http.MethodPost, "/gaps", `{"gap_type":"missing_status_mapping","severity":"warning"}`,
		map[string]string{"version_id": uuid.New().String()}))
	require.Equal(t, http.StatusCreated, rr.Code)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "platform.provider.gap.created", audit.entries[0].Action)
}

func TestProviderHandler_CreateGap_Invalid_422(t *testing.T) {
	f := &fakeProviderRegistry{createGap: func(uuid.UUID, string, string) (*model.ProviderIntegrationGap, error) {
		return nil, service.ErrInvalidGap
	}}
	h := NewProviderHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.CreateGap(rr, provReq(http.MethodPost, "/gaps", `{"gap_type":"bogus","severity":"warning"}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderHandler_UpdateSchema_Success_Audits(t *testing.T) {
	audit := &fakePlatformAudit{}
	f := &fakeProviderRegistry{setSchema: func(versionID uuid.UUID, groups []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error) {
		return &model.ProviderFieldSchema{ProviderVersionID: versionID, Groups: groups}, nil
	}}
	h := NewProviderHandler(f, audit)
	rr := httptest.NewRecorder()
	h.UpdateSchema(rr, provReq(http.MethodPatch, "/schema",
		`{"groups":[{"key":"settings","label":"S","fields":[{"key":"region","label":"R","type":"string"}]}]}`,
		map[string]string{"version_id": uuid.New().String()}))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "platform.provider.version.schema_updated", audit.entries[0].Action)
}
