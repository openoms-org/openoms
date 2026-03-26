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

func TestAccountingHandler_UpdateAccountingSettings_InvalidJSON(t *testing.T) {
	h := NewAccountingHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/settings/accounting", strings.NewReader("not json"))
	rr := httptest.NewRecorder()

	h.UpdateAccountingSettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", resp["error"])
}

func TestAccountingHandler_UpdateAccountingSettings_UnknownProvider(t *testing.T) {
	h := NewAccountingHandler(nil, nil, nil)

	body := `{"provider":"unknown_provider","credentials":{}}`
	req := httptest.NewRequest(http.MethodPut, "/v1/settings/accounting", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.UpdateAccountingSettings(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp["error"], "unknown provider")
}

func TestIsKnownProvider(t *testing.T) {
	tests := []struct {
		provider string
		expected bool
	}{
		{"fakturownia", true},
		{"wfirma", true},
		{"infakt", true},
		{"", false},
		{"unknown", false},
		{"quickbooks", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			assert.Equal(t, tt.expected, isKnownProvider(tt.provider))
		})
	}
}
