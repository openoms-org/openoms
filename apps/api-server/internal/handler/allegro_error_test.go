package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestWriteAllegroError_APIError400EmptyBodyIncludesCause(t *testing.T) {
	rr := httptest.NewRecorder()
	writeAllegroError(rr, "failed to sync orders from Allegro", &allegrosdk.APIError{StatusCode: 400})

	assert.Equal(t, http.StatusBadRequest, rr.Code, "status mapping for Allegro 400 must stay 400")
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Contains(t, resp["error"], "failed to sync orders from Allegro")
	assert.Contains(t, resp["error"], "HTTP 400")
}
