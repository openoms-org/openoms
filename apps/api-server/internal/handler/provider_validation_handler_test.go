package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

type fakeProviderValidation struct {
	startRun     func(uuid.UUID, string, bool) (*model.ProviderValidationRun, error)
	recordResult func(uuid.UUID, string) (*model.ProviderValidationResult, error)
	completeRun  func(uuid.UUID) (*model.ProviderValidationRun, error)
	getRun       func(uuid.UUID) (*model.ProviderValidationRun, error)
}

func (f *fakeProviderValidation) SetProbes(context.Context, uuid.UUID, []model.ProviderValidationProbe) ([]model.ProviderValidationProbe, error) {
	return []model.ProviderValidationProbe{}, nil
}
func (f *fakeProviderValidation) GetProbes(context.Context, uuid.UUID) ([]model.ProviderValidationProbe, error) {
	return []model.ProviderValidationProbe{}, nil
}
func (f *fakeProviderValidation) StartRun(_ context.Context, versionID uuid.UUID, env string, allowDestructive bool, _ *uuid.UUID) (*model.ProviderValidationRun, error) {
	return f.startRun(versionID, env, allowDestructive)
}
func (f *fakeProviderValidation) RecordResult(_ context.Context, runID uuid.UUID, _, label, _, _, _, _ string) (*model.ProviderValidationResult, error) {
	return f.recordResult(runID, label)
}
func (f *fakeProviderValidation) CompleteRun(_ context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error) {
	return f.completeRun(runID)
}
func (f *fakeProviderValidation) ListRuns(context.Context, uuid.UUID) ([]model.ProviderValidationRun, error) {
	return []model.ProviderValidationRun{}, nil
}
func (f *fakeProviderValidation) GetRunWithResults(_ context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error) {
	return f.getRun(runID)
}

func TestProviderValidationHandler_StartRun_Destructive_422(t *testing.T) {
	f := &fakeProviderValidation{startRun: func(uuid.UUID, string, bool) (*model.ProviderValidationRun, error) {
		return nil, service.ErrDestructiveProbeNotConfirmed
	}}
	h := NewProviderValidationHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.StartRun(rr, provReq(http.MethodPost, "/validate", `{"environment":"sandbox"}`,
		map[string]string{"version_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderValidationHandler_StartRun_Success_Audits(t *testing.T) {
	audit := &fakePlatformAudit{}
	f := &fakeProviderValidation{startRun: func(versionID uuid.UUID, env string, _ bool) (*model.ProviderValidationRun, error) {
		return &model.ProviderValidationRun{ID: uuid.New(), ProviderVersionID: versionID, Environment: env, Verdict: model.RunVerdictPending}, nil
	}}
	h := NewProviderValidationHandler(f, audit)
	rr := httptest.NewRecorder()
	h.StartRun(rr, provReq(http.MethodPost, "/validate", `{"environment":"sandbox","allow_destructive":true}`,
		map[string]string{"version_id": uuid.New().String()}))
	require.Equal(t, http.StatusCreated, rr.Code)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "platform.provider.validation.started", audit.entries[0].Action)
}

func TestProviderValidationHandler_RecordResult_Finalized_422(t *testing.T) {
	f := &fakeProviderValidation{recordResult: func(uuid.UUID, string) (*model.ProviderValidationResult, error) {
		return nil, service.ErrValidationRunFinalized
	}}
	h := NewProviderValidationHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.RecordResult(rr, provReq(http.MethodPost, "/results", `{"label":"auth","status":"passed"}`,
		map[string]string{"run_id": uuid.New().String()}))
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestProviderValidationHandler_GetRun_NotFound_404(t *testing.T) {
	f := &fakeProviderValidation{getRun: func(uuid.UUID) (*model.ProviderValidationRun, error) {
		return nil, service.ErrValidationRunNotFound
	}}
	h := NewProviderValidationHandler(f, &fakePlatformAudit{})
	rr := httptest.NewRecorder()
	h.GetRun(rr, provReq(http.MethodGet, "/", "", map[string]string{"run_id": uuid.New().String()}))
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestProviderValidationHandler_CompleteRun_Success_Audits(t *testing.T) {
	audit := &fakePlatformAudit{}
	f := &fakeProviderValidation{completeRun: func(runID uuid.UUID) (*model.ProviderValidationRun, error) {
		return &model.ProviderValidationRun{ID: runID, Verdict: model.RunVerdictFailed}, nil
	}}
	h := NewProviderValidationHandler(f, audit)
	rr := httptest.NewRecorder()
	h.CompleteRun(rr, provReq(http.MethodPost, "/complete", ``, map[string]string{"run_id": uuid.New().String()}))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "platform.provider.validation.completed", audit.entries[0].Action)
}
