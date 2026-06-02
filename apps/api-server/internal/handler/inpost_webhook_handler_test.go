package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// --- fakes ---

type fakeShipmentStatusUpdater struct {
	mu          sync.Mutex
	tenantCalls []struct {
		tenant           uuid.UUID
		tracking, status string
	}
	globalCalls []struct{ tracking, status string }
	called      chan struct{}
}

func newFakeUpdater() *fakeShipmentStatusUpdater {
	return &fakeShipmentStatusUpdater{called: make(chan struct{}, 4)}
}

func (f *fakeShipmentStatusUpdater) UpdateStatusByTrackingNumber(_ context.Context, trackingNumber, _ string, newStatus string) error {
	f.mu.Lock()
	f.globalCalls = append(f.globalCalls, struct{ tracking, status string }{trackingNumber, newStatus})
	f.mu.Unlock()
	f.signal()
	return nil
}

func (f *fakeShipmentStatusUpdater) UpdateStatusByTrackingNumberForTenant(_ context.Context, tenantID uuid.UUID, trackingNumber, _ string, newStatus string) error {
	f.mu.Lock()
	f.tenantCalls = append(f.tenantCalls, struct {
		tenant           uuid.UUID
		tracking, status string
	}{tenantID, trackingNumber, newStatus})
	f.mu.Unlock()
	f.signal()
	return nil
}

func (f *fakeShipmentStatusUpdater) signal() {
	select {
	case f.called <- struct{}{}:
	default:
	}
}

func (f *fakeShipmentStatusUpdater) waitCalled(t *testing.T) {
	t.Helper()
	select {
	case <-f.called:
	case <-time.After(2 * time.Second):
		t.Fatal("expected shipment updater to be called")
	}
}

func (f *fakeShipmentStatusUpdater) anyCallWithin(d time.Duration) bool {
	select {
	case <-f.called:
		return true
	case <-time.After(d):
		return false
	}
}

type fakeWebhookSecretResolver struct {
	secret string
	scope  *service.ProviderWebhookScope
	err    error
}

func (f *fakeWebhookSecretResolver) Resolve(context.Context, string, uuid.UUID) (string, *service.ProviderWebhookScope, error) {
	return f.secret, f.scope, f.err
}

// --- helpers ---

func signInPost(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func inPostRequest(body []byte, integrationID, signature string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inpost", bytes.NewReader(body))
	if signature != "" {
		req.Header.Set("X-InPost-Signature", signature)
	}
	if integrationID != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("integrationID", integrationID)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return req
}

const inPostStatusBody = `{"type":"shipment_status_changed","payload":{"shipment_id":1,"tracking_number":"T123","status":"taken_by_courier"}}`

// --- tests ---

func TestInPostWebhook_InvalidIntegrationID_Returns400(t *testing.T) {
	updater := newFakeUpdater()
	h := NewInPostWebhookHandler("deploy-secret", updater)
	h.SetProviderWebhookSecretResolver(&fakeWebhookSecretResolver{secret: "s"})

	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest([]byte(inPostStatusBody), "not-a-uuid", "sig"))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.False(t, updater.anyCallWithin(100*time.Millisecond), "must not process an invalid integration id")
}

func TestInPostWebhook_ScopedButNoResolver_Returns422(t *testing.T) {
	updater := newFakeUpdater()
	h := NewInPostWebhookHandler("deploy-secret", updater) // no resolver set

	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest([]byte(inPostStatusBody), uuid.NewString(), "sig"))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestInPostWebhook_ResolverEmptySecret_Returns422(t *testing.T) {
	updater := newFakeUpdater()
	h := NewInPostWebhookHandler("deploy-secret", updater)
	h.SetProviderWebhookSecretResolver(&fakeWebhookSecretResolver{secret: ""}) // resolved but unconfigured

	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest([]byte(inPostStatusBody), uuid.NewString(), "sig"))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestInPostWebhook_LegacyNoSecret_Returns422(t *testing.T) {
	updater := newFakeUpdater()
	h := NewInPostWebhookHandler("", updater) // legacy path, deployment secret empty

	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest([]byte(inPostStatusBody), "", "sig"))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestInPostWebhook_MissingSignature_Returns200_NoProcessing(t *testing.T) {
	updater := newFakeUpdater()
	h := NewInPostWebhookHandler("deploy-secret", updater)

	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest([]byte(inPostStatusBody), "", "")) // no X-InPost-Signature

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, updater.anyCallWithin(100*time.Millisecond), "must not process without a signature")
}

func TestInPostWebhook_InvalidSignature_Returns200_NoProcessing(t *testing.T) {
	updater := newFakeUpdater()
	h := NewInPostWebhookHandler("deploy-secret", updater)

	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest([]byte(inPostStatusBody), "", "deadbeef")) // wrong signature

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, updater.anyCallWithin(100*time.Millisecond), "must not process an invalid signature")
}

func TestInPostWebhook_ValidScopedSignature_UpdatesWithinTenant(t *testing.T) {
	updater := newFakeUpdater()
	tenantID := uuid.New()
	secret := "tenant-secret"
	h := NewInPostWebhookHandler("deploy-secret", updater)
	h.SetProviderWebhookSecretResolver(&fakeWebhookSecretResolver{
		secret: secret,
		scope:  &service.ProviderWebhookScope{TenantID: tenantID},
	})

	body := []byte(inPostStatusBody)
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest(body, uuid.NewString(), signInPost(secret, body)))

	assert.Equal(t, http.StatusOK, rr.Code)
	updater.waitCalled(t)
	updater.mu.Lock()
	defer updater.mu.Unlock()
	require.Len(t, updater.tenantCalls, 1, "scoped webhook must route through the tenant-scoped updater")
	assert.Empty(t, updater.globalCalls, "scoped webhook must not use the cross-tenant updater")
	assert.Equal(t, tenantID, updater.tenantCalls[0].tenant)
	assert.Equal(t, "T123", updater.tenantCalls[0].tracking)
	assert.Equal(t, "in_transit", updater.tenantCalls[0].status) // taken_by_courier -> in_transit
}

func TestInPostWebhook_ValidLegacySignature_UsesCrossTenantUpdater(t *testing.T) {
	updater := newFakeUpdater()
	secret := "deploy-secret"
	h := NewInPostWebhookHandler(secret, updater) // legacy: no integrationID, deployment secret

	body := []byte(inPostStatusBody)
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, inPostRequest(body, "", signInPost(secret, body)))

	assert.Equal(t, http.StatusOK, rr.Code)
	updater.waitCalled(t)
	updater.mu.Lock()
	defer updater.mu.Unlock()
	require.Len(t, updater.globalCalls, 1, "legacy webhook uses the cross-tenant updater")
	assert.Empty(t, updater.tenantCalls)
	assert.Equal(t, "T123", updater.globalCalls[0].tracking)
	assert.Equal(t, "in_transit", updater.globalCalls[0].status)
}
