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

func TestRateHandler_GetRates_InvalidBody(t *testing.T) {
	h := NewRateHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/shipping/rates", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.GetRates(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestRateHandler_GetRates_ZeroWeight(t *testing.T) {
	h := NewRateHandler(nil)

	body := `{"from_postal_code":"00-001","to_postal_code":"30-001","weight":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipping/rates", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.GetRates(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "weight must be greater than 0", resp["error"])
}

func TestRateHandler_GetRates_NegativeWeight(t *testing.T) {
	h := NewRateHandler(nil)

	body := `{"weight":-1}`
	req := httptest.NewRequest(http.MethodPost, "/v1/shipping/rates", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.GetRates(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
