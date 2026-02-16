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

// --- GetDispute ---

func TestAllegroDisputesHandler_GetDispute_MissingDisputeID(t *testing.T) {
	h := NewAllegroDisputesHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/allegro/disputes/", nil)
	rr := httptest.NewRecorder()

	h.GetDispute(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID sporu jest wymagane", resp["error"])
}

// --- ListDisputeMessages ---

func TestAllegroDisputesHandler_ListDisputeMessages_MissingDisputeID(t *testing.T) {
	h := NewAllegroDisputesHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/allegro/disputes//messages", nil)
	rr := httptest.NewRecorder()

	h.ListDisputeMessages(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID sporu jest wymagane", resp["error"])
}

// --- SendDisputeMessage ---

func TestAllegroDisputesHandler_SendDisputeMessage_MissingDisputeID(t *testing.T) {
	h := NewAllegroDisputesHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/disputes//messages", strings.NewReader(`{"text":"hello"}`))
	rr := httptest.NewRecorder()

	h.SendDisputeMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID sporu jest wymagane", resp["error"])
}

func TestAllegroDisputesHandler_SendDisputeMessage_InvalidJSON(t *testing.T) {
	h := NewAllegroDisputesHandler(nil, nil)

	rctx := newChiContextWithParam("disputeId", "dispute-123")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/disputes/dispute-123/messages", strings.NewReader("bad"))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.SendDisputeMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Nieprawidlowe dane zadania", resp["error"])
}

func TestAllegroDisputesHandler_SendDisputeMessage_EmptyText(t *testing.T) {
	h := NewAllegroDisputesHandler(nil, nil)

	rctx := newChiContextWithParam("disputeId", "dispute-123")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/disputes/dispute-123/messages", strings.NewReader(`{"text":""}`))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.SendDisputeMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Tresc wiadomosci jest wymagana", resp["error"])
}

func TestAllegroDisputesHandler_SendDisputeMessage_MissingTextKey(t *testing.T) {
	h := NewAllegroDisputesHandler(nil, nil)

	rctx := newChiContextWithParam("disputeId", "dispute-123")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/disputes/dispute-123/messages", strings.NewReader(`{}`))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.SendDisputeMessage(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Tresc wiadomosci jest wymagana", resp["error"])
}
