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

// --- GetAnswer ---

func TestAllegroRatingsHandler_GetAnswer_MissingRatingID(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/integrations/allegro/ratings//answer", nil)
	rr := httptest.NewRecorder()

	h.GetAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID oceny jest wymagane", resp["error"])
}

// --- CreateAnswer ---

func TestAllegroRatingsHandler_CreateAnswer_MissingRatingID(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/integrations/allegro/ratings//answer", strings.NewReader(`{"text":"thank you"}`))
	rr := httptest.NewRecorder()

	h.CreateAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID oceny jest wymagane", resp["error"])
}

func TestAllegroRatingsHandler_CreateAnswer_InvalidJSON(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	rctx := newChiContextWithParam("ratingId", "rating-123")
	req := httptest.NewRequest(http.MethodPut, "/v1/integrations/allegro/ratings/rating-123/answer", strings.NewReader("bad"))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.CreateAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Nieprawidlowe dane zadania", resp["error"])
}

func TestAllegroRatingsHandler_CreateAnswer_EmptyText(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	rctx := newChiContextWithParam("ratingId", "rating-123")
	req := httptest.NewRequest(http.MethodPut, "/v1/integrations/allegro/ratings/rating-123/answer", strings.NewReader(`{"text":""}`))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.CreateAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Tresc odpowiedzi jest wymagana", resp["error"])
}

func TestAllegroRatingsHandler_CreateAnswer_MissingTextKey(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	rctx := newChiContextWithParam("ratingId", "rating-123")
	req := httptest.NewRequest(http.MethodPut, "/v1/integrations/allegro/ratings/rating-123/answer", strings.NewReader(`{}`))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.CreateAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Tresc odpowiedzi jest wymagana", resp["error"])
}

// --- DeleteAnswer ---

func TestAllegroRatingsHandler_DeleteAnswer_MissingRatingID(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/integrations/allegro/ratings//answer", nil)
	rr := httptest.NewRecorder()

	h.DeleteAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID oceny jest wymagane", resp["error"])
}

// --- RequestRemoval ---

func TestAllegroRatingsHandler_RequestRemoval_MissingRatingID(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/ratings//removal", strings.NewReader(`{"reason":"offensive"}`))
	rr := httptest.NewRecorder()

	h.RequestRemoval(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ID oceny jest wymagane", resp["error"])
}

func TestAllegroRatingsHandler_RequestRemoval_InvalidJSON(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	rctx := newChiContextWithParam("ratingId", "rating-456")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/ratings/rating-456/removal", strings.NewReader("bad"))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.RequestRemoval(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Nieprawidlowe dane zadania", resp["error"])
}

func TestAllegroRatingsHandler_RequestRemoval_EmptyReason(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	rctx := newChiContextWithParam("ratingId", "rating-456")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/ratings/rating-456/removal", strings.NewReader(`{"reason":""}`))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.RequestRemoval(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Powod zgloszenia jest wymagany", resp["error"])
}

func TestAllegroRatingsHandler_RequestRemoval_MissingReasonKey(t *testing.T) {
	h := NewAllegroRatingsHandler(nil, nil)

	rctx := newChiContextWithParam("ratingId", "rating-456")
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/ratings/rating-456/removal", strings.NewReader(`{}`))
	req = req.WithContext(rctx)
	rr := httptest.NewRecorder()

	h.RequestRemoval(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Powod zgloszenia jest wymagany", resp["error"])
}
