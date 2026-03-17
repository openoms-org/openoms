package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func computeHMACSHA256(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestSecurity_Webhook_InPostRejectsWhenNoSecret(t *testing.T) {
	h := NewInPostWebhookHandler("", nil) // empty secret
	req := httptest.NewRequest("POST", "/v1/webhooks/inpost", strings.NewReader(`{"type":"test"}`))
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, req)
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestSecurity_Webhook_AllegroRejectsWhenNoSecret(t *testing.T) {
	h := NewAllegroWebhookHandler("", nil) // empty secret
	req := httptest.NewRequest("POST", "/v1/webhooks/allegro", strings.NewReader(`{"type":"test"}`))
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, req)
	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}

func TestSecurity_Webhook_InPostValidSignaturePasses(t *testing.T) {
	secret := "test-webhook-secret" //nolint:gosec // test credential
	h := NewInPostWebhookHandler(secret, nil)

	body := []byte(`{"type":"shipment_status_changed","payload":{"shipment_id":123,"tracking_number":"ABC123","status":"delivered"}}`)
	sig := computeHMACSHA256(secret, body)

	req := httptest.NewRequest("POST", "/v1/webhooks/inpost", bytes.NewReader(body))
	req.Header.Set("X-InPost-Signature", sig)
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecurity_Webhook_InPostInvalidSignatureRejected(t *testing.T) {
	h := NewInPostWebhookHandler("real-secret", nil)
	body := []byte(`{"type":"test","payload":{}}`)
	req := httptest.NewRequest("POST", "/v1/webhooks/inpost", bytes.NewReader(body))
	req.Header.Set("X-InPost-Signature", "invalid-signature")
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, req)
	// Handler returns 200 even on invalid sig (to prevent InPost retries)
	// but the event is NOT processed
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecurity_Webhook_AllegroValidSignaturePasses(t *testing.T) {
	secret := "test-allegro-secret"
	h := NewAllegroWebhookHandler(secret, nil)

	body := []byte(`{"id":"evt-1","type":"ORDER_FILLED_IN","occurred_at":"2026-01-01T00:00:00Z","order":{"id":"ord-1"}}`)
	sig := computeHMACSHA256(secret, body)

	req := httptest.NewRequest("POST", "/v1/webhooks/allegro", bytes.NewReader(body))
	req.Header.Set("X-Allegro-Signature", sig)
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestSecurity_Webhook_AllegroInvalidSignatureRejected(t *testing.T) {
	h := NewAllegroWebhookHandler("real-secret", nil)
	body := []byte(`{"id":"evt-1","type":"ORDER_FILLED_IN","occurred_at":"2026-01-01T00:00:00Z","order":{"id":"ord-1"}}`)
	req := httptest.NewRequest("POST", "/v1/webhooks/allegro", bytes.NewReader(body))
	req.Header.Set("X-Allegro-Signature", "bad-sig")
	rr := httptest.NewRecorder()
	h.HandleWebhook(rr, req)
	// Handler returns 200 even on invalid sig (to prevent Allegro retries)
	assert.Equal(t, http.StatusOK, rr.Code)
}
