package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allegrosdk "github.com/openoms-org/openoms/packages/allegro-go-sdk"
)

func TestWriteAllegroError_NonAPIErrorIncludesCause(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAllegroError(rr, "failed to sync orders from Allegro", errors.New("allegro: list checkout-forms: boom"))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "failed to sync orders from Allegro")
	assert.Contains(t, resp["error"], "boom")
}

func TestWriteAllegroError_ReconnectRequiredIsClearAndDoesNotAskPassword(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAllegroError(rr, "failed to sync orders from Allegro",
		errors.Join(allegrosdk.ErrReconnectRequired, errors.New("allegro: proactive token refresh failed: allegro: HTTP 400")))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, "must not return JWT-like 401")
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, strings.ToLower(resp["error"]), "reconnect allegro")
	assert.NotContains(t, strings.ToLower(resp["error"]), "password")
}

func TestWriteAllegroError_CreateCommandERRORIncludesCodeNotTimeout(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAllegroError(rr, "Failed to create shipment on Allegro",
		errors.New("allegro: create shipment: allegro: create-commands 58d250bc-5441-48a0-a7f9-ea7497b4a3a1 ERROR: SHIPMENT_VALIDATION_ERROR: A request can contain only one parcel"))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "58d250bc-5441-48a0-a7f9-ea7497b4a3a1")
	assert.Contains(t, resp["error"], "SHIPMENT_VALIDATION_ERROR")
	assert.Contains(t, resp["error"], "A request can contain only one parcel")
	assert.NotContains(t, resp["error"], "timed out")
}

func TestWriteAllegroError_CreateCommandTimeoutIncludesCommandIDAndStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAllegroError(rr, "Failed to create shipment on Allegro",
		errors.New("allegro: create shipment: allegro: create-commands 58d250bc-5441-48a0-a7f9-ea7497b4a3a1 timed out waiting for shipmentId (last status IN_PROGRESS)"))

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "58d250bc-5441-48a0-a7f9-ea7497b4a3a1")
	assert.Contains(t, resp["error"], "IN_PROGRESS")
	assert.Contains(t, resp["error"], "timed out")
}

func TestWriteAllegroError_APIError400EmptyBodyIncludesCause(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAllegroError(rr, "failed to sync orders from Allegro", &allegrosdk.APIError{StatusCode: 400})

	assert.Equal(t, http.StatusBadRequest, rr.Code, "status mapping for Allegro 400 must stay 400")
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "failed to sync orders from Allegro")
	assert.Contains(t, resp["error"], "HTTP 400")
}
