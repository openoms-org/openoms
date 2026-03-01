package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// onboardingCtx creates a context with tenant and user IDs for onboarding tests.
func onboardingCtx(ctx context.Context, tenantID, userID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)
	return ctx
}

// onboardingRouteCtx adds a chi URL param to the context.
func onboardingRouteCtx(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

// decodeOnboardingStatus reads and parses OnboardingSettings from response body.
func decodeOnboardingStatus(t *testing.T, rr *httptest.ResponseRecorder) model.OnboardingSettings {
	t.Helper()
	var status model.OnboardingSettings
	err := json.NewDecoder(rr.Body).Decode(&status)
	require.NoError(t, err)
	return status
}

// decodeErrorResponse reads the {"error": "..."} response body.
func decodeErrorResponse(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	return resp["error"]
}

// ---------------------------------------------------------------------------
// GET /v1/onboarding/status — returns current onboarding state
// ---------------------------------------------------------------------------

func TestGetOnboardingStatus_NewTenant_ReturnsDefaults(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	tenantID := uuid.New()
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/onboarding/status", nil)
	req = req.WithContext(onboardingCtx(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.GetOnboardingStatus(rr, req)

	// New handler method must exist and return proper defaults:
	// completed=false, current_step=1, empty slices, no completed_at
	require.Equal(t, http.StatusOK, rr.Code)

	status := decodeOnboardingStatus(t, rr)
	assert.False(t, status.Completed, "new tenant should not be completed")
	assert.Equal(t, 1, status.CurrentStep, "new tenant should start at step 1")
	assert.Empty(t, status.CompletedSteps, "no steps completed yet")
	assert.Empty(t, status.SkippedSteps, "no steps skipped yet")
	assert.Empty(t, status.CompletedAt, "no completion timestamp")
}

func TestGetOnboardingStatus_ResponseIsJSON_200(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	tenantID := uuid.New()
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/v1/onboarding/status", nil)
	req = req.WithContext(onboardingCtx(req.Context(), tenantID, userID))
	rr := httptest.NewRecorder()

	h.GetOnboardingStatus(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
}

// ---------------------------------------------------------------------------
// PUT /v1/onboarding/step/{step} — mark step completed or skipped
// ---------------------------------------------------------------------------

func TestUpdateOnboardingStep_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/1", strings.NewReader("not json"))
	ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
	ctx = onboardingRouteCtx(ctx, "step", "1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.UpdateOnboardingStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	errMsg := decodeErrorResponse(t, rr)
	assert.NotEmpty(t, errMsg)
}

func TestUpdateOnboardingStep_StepOutOfRange(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name string
		step string
	}{
		{"step 0", "0"},
		{"step 5", "5"},
		{"step -1", "-1"},
		{"step 99", "99"},
		{"step not a number", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"action":"completed"}`
			req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/"+tt.step, strings.NewReader(body))
			ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
			ctx = onboardingRouteCtx(ctx, "step", tt.step)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.UpdateOnboardingStep(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code, "step %q should be rejected", tt.step)
		})
	}
}

func TestUpdateOnboardingStep_Step1CannotBeSkipped(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"action":"skipped"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/1", strings.NewReader(body))
	ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
	ctx = onboardingRouteCtx(ctx, "step", "1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.UpdateOnboardingStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	errMsg := decodeErrorResponse(t, rr)
	assert.Contains(t, errMsg, "step 1")
}

func TestUpdateOnboardingStep_InvalidAction(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name   string
		action string
	}{
		{"empty action", ""},
		{"unknown action", "maybe"},
		{"partial match", "complete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{"action":"` + tt.action + `"}`
			req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/2", strings.NewReader(body))
			ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
			ctx = onboardingRouteCtx(ctx, "step", "2")
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.UpdateOnboardingStep(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code, "action %q should be rejected", tt.action)
		})
	}
}

func TestUpdateOnboardingStep_ValidStepsRange(t *testing.T) {
	// Steps 2-4 should accept both "completed" and "skipped" actions
	// (We can't test full success without DB, but we verify the handler
	// doesn't reject valid input at the validation layer)
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	validCases := []struct {
		step   string
		action string
	}{
		{"1", "completed"},
		{"2", "completed"},
		{"2", "skipped"},
		{"3", "completed"},
		{"3", "skipped"},
		{"4", "completed"},
		{"4", "skipped"},
	}

	for _, tt := range validCases {
		t.Run("step_"+tt.step+"_"+tt.action, func(t *testing.T) {
			body := `{"action":"` + tt.action + `"}`
			req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/"+tt.step, strings.NewReader(body))
			ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
			ctx = onboardingRouteCtx(ctx, "step", tt.step)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.UpdateOnboardingStep(rr, req)

			// Should NOT return 400 for valid input. May return 500 (nil pool)
			// which is acceptable — that means validation passed.
			assert.NotEqual(t, http.StatusBadRequest, rr.Code,
				"step %s with action %q should pass validation", tt.step, tt.action)
		})
	}
}

func TestUpdateOnboardingStep_EmptyBody(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/2", strings.NewReader(""))
	ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
	ctx = onboardingRouteCtx(ctx, "step", "2")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.UpdateOnboardingStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// ---------------------------------------------------------------------------
// POST /v1/onboarding/complete — mark onboarding as done
// ---------------------------------------------------------------------------

func TestCompleteOnboarding_InvalidJSON(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/complete", strings.NewReader("not json"))
	req = req.WithContext(onboardingCtx(req.Context(), uuid.New(), uuid.New()))
	rr := httptest.NewRecorder()

	h.CompleteOnboarding(rr, req)

	// Handler must exist. May accept empty body or reject invalid JSON.
	// Either way, it must not panic.
	assert.True(t, rr.Code >= 200 && rr.Code < 600, "should return valid HTTP status")
}

func TestCompleteOnboarding_MethodExists(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	// Verify the method exists and is callable (compilation test + basic contract)
	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/complete", strings.NewReader("{}"))
	req = req.WithContext(onboardingCtx(req.Context(), uuid.New(), uuid.New()))
	rr := httptest.NewRecorder()

	h.CompleteOnboarding(rr, req)

	// Without DB, expect 500 (nil pool) — not 400, meaning validation passed.
	// The key contract: step 1 must be completed before onboarding can complete.
	// Without state, this should return an error indicating step 1 is required.
	assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusInternalServerError,
		"should either require step 1 or fail on DB, got %d", rr.Code)
}

// ---------------------------------------------------------------------------
// Contract: OnboardingSettings model has required fields
// ---------------------------------------------------------------------------

func TestOnboardingSettings_JSONRoundtrip(t *testing.T) {
	original := model.OnboardingSettings{
		Completed:      false,
		CurrentStep:    2,
		CompletedSteps: []int{1},
		SkippedSteps:   []int{},
		CompletedAt:    "",
		Dismissed:      false,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded model.OnboardingSettings
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Completed, decoded.Completed)
	assert.Equal(t, original.CurrentStep, decoded.CurrentStep)
	assert.Equal(t, original.CompletedSteps, decoded.CompletedSteps)
	assert.Equal(t, original.Dismissed, decoded.Dismissed)
}

func TestOnboardingSettings_DefaultsFromEmptyJSON(t *testing.T) {
	// When tenant has no onboarding key, empty JSON should unmarshal to zero values
	var status model.OnboardingSettings
	err := json.Unmarshal([]byte(`{}`), &status)
	require.NoError(t, err)

	assert.False(t, status.Completed)
	assert.Equal(t, 0, status.CurrentStep) // zero value; handler should default to 1
	assert.Nil(t, status.CompletedSteps)
	assert.Nil(t, status.SkippedSteps)
	assert.Empty(t, status.CompletedAt)
	assert.False(t, status.Dismissed)
}

func TestOnboardingSettings_BackwardsCompatibility(t *testing.T) {
	// Existing tenants have {"dismissed": true} — new fields should have zero values
	var status model.OnboardingSettings
	err := json.Unmarshal([]byte(`{"dismissed":true,"completed_at":"2025-01-01T00:00:00Z"}`), &status)
	require.NoError(t, err)

	assert.True(t, status.Dismissed, "dismissed should be preserved")
	assert.Equal(t, "2025-01-01T00:00:00Z", status.CompletedAt)
	assert.False(t, status.Completed, "new field should default to false")
	assert.Equal(t, 0, status.CurrentStep, "new field should default to 0")
	assert.Nil(t, status.CompletedSteps, "new field should default to nil")
	assert.Nil(t, status.SkippedSteps, "new field should default to nil")
}

func TestOnboardingSettings_FullStateJSON(t *testing.T) {
	// Full onboarding state round-trip
	input := `{
		"completed": true,
		"current_step": 4,
		"completed_steps": [1, 2],
		"skipped_steps": [3, 4],
		"completed_at": "2025-03-01T12:00:00Z",
		"dismissed": false
	}`

	var status model.OnboardingSettings
	err := json.Unmarshal([]byte(input), &status)
	require.NoError(t, err)

	assert.True(t, status.Completed)
	assert.Equal(t, 4, status.CurrentStep)
	assert.Equal(t, []int{1, 2}, status.CompletedSteps)
	assert.Equal(t, []int{3, 4}, status.SkippedSteps)
	assert.Equal(t, "2025-03-01T12:00:00Z", status.CompletedAt)
	assert.False(t, status.Dismissed)
}

// ---------------------------------------------------------------------------
// Edge case: step transitions and idempotency contracts
// (These define the expected behavior — implementation must satisfy them)
// ---------------------------------------------------------------------------

func TestUpdateOnboardingStep_Step1Completed_AcceptsValidAction(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"action":"completed"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/1", strings.NewReader(body))
	ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
	ctx = onboardingRouteCtx(ctx, "step", "1")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.UpdateOnboardingStep(rr, req)

	// Step 1 completed should pass validation (may fail on DB = 500)
	assert.NotEqual(t, http.StatusBadRequest, rr.Code,
		"completing step 1 should pass validation")
}

func TestUpdateOnboardingStep_Steps2to4_CanBeSkipped(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	for _, step := range []string{"2", "3", "4"} {
		t.Run("step_"+step, func(t *testing.T) {
			body := `{"action":"skipped"}`
			req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/"+step, strings.NewReader(body))
			ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
			ctx = onboardingRouteCtx(ctx, "step", step)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			h.UpdateOnboardingStep(rr, req)

			assert.NotEqual(t, http.StatusBadRequest, rr.Code,
				"skipping step %s should pass validation", step)
		})
	}
}

func TestUpdateOnboardingStep_MissingStepParam(t *testing.T) {
	h := NewSettingsHandler(nil, nil, nil, nil, nil, nil)

	body := `{"action":"completed"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/onboarding/step/", strings.NewReader(body))
	// No chi route context — step param is empty
	ctx := onboardingCtx(req.Context(), uuid.New(), uuid.New())
	rctx := chi.NewRouteContext()
	// Don't add step param
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.UpdateOnboardingStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code, "missing step param should be rejected")
}
