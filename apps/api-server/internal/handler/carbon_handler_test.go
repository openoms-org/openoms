package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCarbonHandler(t *testing.T) {
	h := NewCarbonHandler(nil)
	assert.NotNil(t, h)
}

func TestCarbonHandler_GetStats_NilServicePanics(t *testing.T) {
	h := NewCarbonHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/carbon/stats?date_from=2026-01-01&date_to=2026-02-28", nil)
	rr := httptest.NewRecorder()

	// GetStats calls h.carbonService.GetCarbonStats directly with no input validation,
	// so with a nil service it will panic.
	assert.Panics(t, func() {
		h.GetStats(rr, req)
	})
}

func TestCarbonHandler_GetReport_NilServicePanics(t *testing.T) {
	h := NewCarbonHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/carbon/report?date_from=2026-01-01&date_to=2026-02-28", nil)
	rr := httptest.NewRecorder()

	// GetReport also has no input validation — nil service causes panic.
	assert.Panics(t, func() {
		h.GetReport(rr, req)
	})
}

func TestCarbonHandler_GetStats_NoQueryParamsPanics(t *testing.T) {
	h := NewCarbonHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/carbon/stats", nil)
	rr := httptest.NewRecorder()

	// Even without query params, the handler passes empty strings to service.
	// With nil service it panics (no validation gate).
	assert.Panics(t, func() {
		h.GetStats(rr, req)
	})
}
