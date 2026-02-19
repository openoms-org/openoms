package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity_Webhook_InPostRejectsWhenNoSecret(t *testing.T) {
	h := NewInPostWebhookHandler("") // empty secret
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
