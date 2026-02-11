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

func TestAllegroAuthHandler_HandleCallback_InvalidJSON(t *testing.T) {
	h := NewAllegroAuthHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAllegroAuthHandler_HandleCallback_MissingCode(t *testing.T) {
	h := NewAllegroAuthHandler(nil, nil, nil)

	body := `{"code":"","state":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/allegro/callback", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.HandleCallback(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "code is required", resp["error"])
}
