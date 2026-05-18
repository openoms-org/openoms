package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

const testWebhookSecret = "test-webhook-secret-key-12345" // #nosec G101

// computeHMAC computes the HMAC-SHA256 signature for the given body using the secret.
func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

type fakeProviderWebhookSecretResolver struct {
	secret string
	scope  *service.ProviderWebhookScope
	err    error
}

func (f fakeProviderWebhookSecretResolver) Resolve(context.Context, string, uuid.UUID) (string, *service.ProviderWebhookScope, error) {
	return f.secret, f.scope, f.err
}

type recordingAllegroSyncer struct {
	importCh chan service.ProviderWebhookScope
}

func (r *recordingAllegroSyncer) ImportOrder(context.Context, string) {}

func (r *recordingAllegroSyncer) UpdateOrderStatus(context.Context, string) {}

func (r *recordingAllegroSyncer) ImportOrderForIntegration(_ context.Context, tenantID, integrationID uuid.UUID, _ string) {
	r.importCh <- service.ProviderWebhookScope{
		TenantID:      tenantID,
		IntegrationID: integrationID,
		Provider:      "allegro",
	}
}

func (r *recordingAllegroSyncer) UpdateOrderStatusForIntegration(context.Context, uuid.UUID, uuid.UUID, string) {
}

func requestWithIntegrationID(req *http.Request, integrationID uuid.UUID) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("integrationID", integrationID.String())
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// --- Valid webhook ---

func TestAllegroWebhookHandler_ValidSignature_OrderStatusChanged(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `{"type":"ORDER_STATUS_CHANGED","id":"evt-1","occurredAt":"2026-01-15T10:00:00Z","payload":{}}`
	signature := computeHMAC(testWebhookSecret, []byte(body))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Webhook handler always returns 200 OK to Allegro
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAllegroWebhookHandler_ValidSignature_OrderFilledIn(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `{"type":"ORDER_FILLED_IN","id":"evt-2","occurredAt":"2026-01-15T11:00:00Z","payload":{"orderId":"ord-123"}}`
	signature := computeHMAC(testWebhookSecret, []byte(body))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAllegroWebhookHandler_ValidSignature_UnknownEventType(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `{"type":"SOME_NEW_EVENT","id":"evt-3","occurredAt":"2026-01-15T12:00:00Z","payload":{}}`
	signature := computeHMAC(testWebhookSecret, []byte(body))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Still returns 200 — unknown event types are gracefully ignored
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAllegroWebhookHandler_ScopedWebhookDispatchesOnlyVerifiedIntegration(t *testing.T) {
	tenantID := uuid.New()
	integrationID := uuid.New()
	secret := "tenant-allegro-webhook-secret" //nolint:gosec // test credential
	syncer := &recordingAllegroSyncer{importCh: make(chan service.ProviderWebhookScope, 1)}
	h := NewAllegroWebhookHandler(testWebhookSecret, syncer)
	h.SetProviderWebhookSecretResolver(fakeProviderWebhookSecretResolver{
		secret: secret,
		scope: &service.ProviderWebhookScope{
			TenantID:      tenantID,
			IntegrationID: integrationID,
			Provider:      "allegro",
		},
	})

	body := `{"type":"ORDER_FILLED_IN","id":"evt-scoped","occurredAt":"2026-05-18T08:00:00Z","order":{"id":"ord-tenant-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro/"+integrationID.String(), strings.NewReader(body))
	req = requestWithIntegrationID(req, integrationID)
	req.Header.Set("X-Allegro-Signature", computeHMAC(secret, []byte(body)))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	select {
	case got := <-syncer.importCh:
		assert.Equal(t, tenantID, got.TenantID)
		assert.Equal(t, integrationID, got.IntegrationID)
	case <-time.After(time.Second):
		require.Fail(t, "expected scoped import dispatch")
	}
}

func TestAllegroWebhookHandler_ScopedWebhookWrongSecretDoesNotDispatch(t *testing.T) {
	integrationID := uuid.New()
	syncer := &recordingAllegroSyncer{importCh: make(chan service.ProviderWebhookScope, 1)}
	h := NewAllegroWebhookHandler(testWebhookSecret, syncer)
	h.SetProviderWebhookSecretResolver(fakeProviderWebhookSecretResolver{
		secret: "tenant-allegro-webhook-secret",
		scope: &service.ProviderWebhookScope{
			TenantID:      uuid.New(),
			IntegrationID: integrationID,
			Provider:      "allegro",
		},
	})

	body := `{"type":"ORDER_FILLED_IN","id":"evt-scoped","occurredAt":"2026-05-18T08:00:00Z","order":{"id":"ord-tenant-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro/"+integrationID.String(), strings.NewReader(body))
	req = requestWithIntegrationID(req, integrationID)
	req.Header.Set("X-Allegro-Signature", computeHMAC("wrong-secret", []byte(body)))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	select {
	case <-syncer.importCh:
		require.Fail(t, "did not expect scoped import dispatch")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestAllegroWebhookHandler_ScopedWebhookWithoutResolverFailsClosed(t *testing.T) {
	integrationID := uuid.New()
	syncer := &recordingAllegroSyncer{importCh: make(chan service.ProviderWebhookScope, 1)}
	h := NewAllegroWebhookHandler(testWebhookSecret, syncer)

	body := `{"type":"ORDER_FILLED_IN","id":"evt-scoped","occurredAt":"2026-05-18T08:00:00Z","order":{"id":"ord-tenant-1"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro/"+integrationID.String(), strings.NewReader(body))
	req = requestWithIntegrationID(req, integrationID)
	req.Header.Set("X-Allegro-Signature", computeHMAC(testWebhookSecret, []byte(body)))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	select {
	case <-syncer.importCh:
		require.Fail(t, "did not expect scoped import dispatch")
	case <-time.After(50 * time.Millisecond):
	}
}

// --- Invalid HMAC signature ---

func TestAllegroWebhookHandler_InvalidSignature(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `{"type":"ORDER_STATUS_CHANGED","id":"evt-4","occurredAt":"2026-01-15T13:00:00Z","payload":{}}`

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", "deadbeef0000000000000000000000000000000000000000000000000000dead")
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// The handler always returns 200 to Allegro (to prevent retries), but the event is not processed.
	// This is the intended behavior per the webhook handler code.
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAllegroWebhookHandler_WrongSecret(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `{"type":"ORDER_STATUS_CHANGED","id":"evt-5","occurredAt":"2026-01-15T14:00:00Z","payload":{}}`
	// Sign with a different secret
	wrongSignature := computeHMAC("wrong-secret", []byte(body))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", wrongSignature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Returns 200 (Allegro convention), but event is silently dropped
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- Missing signature header ---

func TestAllegroWebhookHandler_MissingSignatureHeader(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `{"type":"ORDER_STATUS_CHANGED","id":"evt-6","occurredAt":"2026-01-15T15:00:00Z","payload":{}}`

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	// No X-Allegro-Signature header
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Returns 200 (handler's policy: always 200 to Allegro)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- Malformed body ---

func TestAllegroWebhookHandler_MalformedJSON(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := `not valid json at all`
	signature := computeHMAC(testWebhookSecret, []byte(body))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Returns 200 (graceful — never retry)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAllegroWebhookHandler_EmptyBody(t *testing.T) {
	h := NewAllegroWebhookHandler(testWebhookSecret, nil)

	body := ""
	signature := computeHMAC(testWebhookSecret, []byte(body))

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", signature)
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Returns 200 (graceful handling of empty body)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- No webhook secret configured ---

func TestAllegroWebhookHandler_NoSecretConfigured_ValidEvent(t *testing.T) {
	// When no secret is configured, webhooks are rejected (422)
	h := NewAllegroWebhookHandler("", nil)

	body := `{"type":"ORDER_STATUS_CHANGED","id":"evt-7","occurredAt":"2026-01-15T16:00:00Z","payload":{}}`

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestAllegroWebhookHandler_NoSecretConfigured_MalformedBody(t *testing.T) {
	// When no secret is configured, webhooks are rejected before parsing body
	h := NewAllegroWebhookHandler("", nil)

	body := `{broken json`

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

// --- Signature header present but secret is empty ---

func TestAllegroWebhookHandler_NoSecretConfigured_WithSignatureHeader(t *testing.T) {
	// When no secret is configured, webhooks are rejected even if signature header is present
	h := NewAllegroWebhookHandler("", nil)

	body := `{"type":"ORDER_STATUS_CHANGED","id":"evt-8","occurredAt":"2026-01-15T17:00:00Z","payload":{}}`

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/allegro", strings.NewReader(body))
	req.Header.Set("X-Allegro-Signature", "some-signature")
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}
