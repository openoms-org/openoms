package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func TestStripeWebhookHandler_MissingSignature(t *testing.T) {
	h := NewStripeWebhookHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "missing Stripe-Signature header", resp["error"])
}

func TestStripeWebhookHandler_InvalidSignature(t *testing.T) {
	// Create a real webhook service with a known secret — any payload with wrong signature fails
	webhookSvc := service.NewStripeWebhookService("whsec_test_secret", nil, nil)
	h := NewStripeWebhookHandler(webhookSvc)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=bad_signature")
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	// Should return 400 (not 500) for signature failures to prevent Stripe retries
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid webhook signature", resp["error"])
}
