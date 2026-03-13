package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testWebhookSecret = "test-webhook-secret-key-12345" // #nosec G101

// computeHMAC computes the HMAC-SHA256 signature for the given body using the secret.
func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
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
