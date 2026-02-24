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

// --- SendMessage ---

func TestAllegroCommsHandler_SendMessage_InvalidJSON(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/messages/thread-1/messages", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.SendMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

func TestAllegroCommsHandler_SendMessage_EmptyText(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/messages/thread-1/messages", strings.NewReader(`{"text":""}`))
	rr := httptest.NewRecorder()

	h.SendMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Message text is required", resp["error"])
}

func TestAllegroCommsHandler_SendMessage_MissingTextKey(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/messages/thread-1/messages", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.SendMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Message text is required", resp["error"])
}

// --- RejectAllegroReturn ---

func TestAllegroCommsHandler_RejectReturn_InvalidJSON(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/returns/ret-1/reject", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.RejectAllegroReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

func TestAllegroCommsHandler_RejectReturn_MissingReason(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/returns/ret-1/reject", strings.NewReader(`{"reason":""}`))
	rr := httptest.NewRecorder()

	h.RejectAllegroReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Rejection reason is required", resp["error"])
}

func TestAllegroCommsHandler_RejectReturn_EmptyBody(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/returns/ret-1/reject", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.RejectAllegroReturn(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Rejection reason is required", resp["error"])
}

// --- CreateRefund ---

func TestAllegroCommsHandler_CreateRefund_InvalidJSON(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/refunds", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.CreateRefund(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid request body", resp["error"])
}

func TestAllegroCommsHandler_CreateRefund_MissingPaymentID(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	body := `{"payment":{"id":""},"reason":"damage"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/refunds", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateRefund(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Payment ID is required", resp["error"])
}

func TestAllegroCommsHandler_CreateRefund_MissingReason(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	body := `{"payment":{"id":"pay-123"},"reason":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/refunds", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateRefund(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Refund reason is required", resp["error"])
}

func TestAllegroCommsHandler_CreateRefund_EmptyBody(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/refunds", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	h.CreateRefund(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	// payment.id is empty, so this should fail on payment ID validation first
	assert.Equal(t, "Payment ID is required", resp["error"])
}

func TestAllegroCommsHandler_CreateRefund_ValidPaymentButMissingReason(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	body := `{"payment":{"id":"pay-456"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/refunds", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateRefund(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Refund reason is required", resp["error"])
}

func TestAllegroCommsHandler_CreateRefund_WithLineItems_MissingPaymentID(t *testing.T) {
	h := NewAllegroCommsHandler(nil, nil)

	body := `{"payment":{"id":""},"reason":"damage","lineItems":[{"offerId":"off-1","quantity":1,"amount":{"amount":"10.00","currency":"PLN"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/refunds", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateRefund(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Payment ID is required", resp["error"])
}
