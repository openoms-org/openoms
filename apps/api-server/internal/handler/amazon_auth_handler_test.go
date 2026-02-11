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

func TestAmazonAuthHandler_Setup_InvalidJSON(t *testing.T) {
	h := NewAmazonAuthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/amazon/setup", strings.NewReader("bad"))
	rr := httptest.NewRecorder()

	h.Setup(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAmazonAuthHandler_Setup_MissingFields(t *testing.T) {
	h := NewAmazonAuthHandler(nil, nil)

	// Missing all required fields
	body := `{"client_id":"","client_secret":"","refresh_token":"","marketplace_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/amazon/setup", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Setup(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "required")
}

func TestAmazonAuthHandler_Setup_MissingClientSecret(t *testing.T) {
	h := NewAmazonAuthHandler(nil, nil)

	body := `{"client_id":"cid","client_secret":"","refresh_token":"rt","marketplace_id":"mp"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/amazon/setup", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Setup(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
