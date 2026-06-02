package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// When the tenant plan cannot be resolved (DB and cache both unavailable), the
// guard must fail CLOSED for state-changing requests so a suspended/past_due
// tenant cannot gain write access during an outage, while safe reads still pass.

func TestFailClosedOnUnavailablePlan_BlocksMutations(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/v1/orders", nil)

		blocked := failClosedOnUnavailablePlan(rr, req)

		assert.True(t, blocked, "%s must be blocked when plan data is unavailable", method)
		assert.Equal(t, http.StatusServiceUnavailable, rr.Code, "%s should get 503", method)
		assert.Contains(t, rr.Body.String(), "plan_check_unavailable")
	}
}

func TestFailClosedOnUnavailablePlan_AllowsSafeReads(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/v1/orders", nil)

		blocked := failClosedOnUnavailablePlan(rr, req)

		assert.False(t, blocked, "%s must be allowed through", method)
		// Nothing written: recorder keeps its default 200 and empty body.
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Empty(t, rr.Body.String())
	}
}
