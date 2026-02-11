package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInPostPointHandler_Search_MissingQuery(t *testing.T) {
	h := NewInPostPointHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "query parameter is required", resp["error"])
}

func TestInPostPointHandler_Search_QueryTooShort(t *testing.T) {
	h := NewInPostPointHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/inpost/points?query=a", nil)
	rr := httptest.NewRecorder()

	h.Search(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "query must be at least 2 characters", resp["error"])
}
