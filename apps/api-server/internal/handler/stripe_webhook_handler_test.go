package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
