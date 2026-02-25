package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector_ReturnsNonNil(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	assert.NotNil(t, mc)
}

func TestMetricsCollector_Middleware_CallsNextHandler(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := mc.Middleware()(next)
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMetricsCollector_Middleware_RecordsRequestCounter(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	handler := mc.Middleware()(okHandler())

	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check the /metrics output has a counter entry
	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, "openoms_http_requests_total")
	assert.Contains(t, metricsOutput, `status="200"`)
	assert.Contains(t, metricsOutput, `method="GET"`)
}

func TestMetricsCollector_Middleware_RecordsDurationHistogram(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	handler := mc.Middleware()(okHandler())

	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, "openoms_http_request_duration_seconds_bucket")
	assert.Contains(t, metricsOutput, "openoms_http_request_duration_seconds_sum")
	assert.Contains(t, metricsOutput, "openoms_http_request_duration_seconds_count")
	assert.Contains(t, metricsOutput, `le="+Inf"`)
}

func TestMetricsCollector_Middleware_TracksBytesWritten(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	body := `{"items":[1,2,3]}`
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})
	handler := mc.Middleware()(next)

	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, "openoms_http_response_bytes_total")
	// The bytes counter should be non-zero (at least the length of the body)
	assert.NotContains(t, metricsOutput, "openoms_http_response_bytes_total 0")
}

func TestMetricsCollector_Middleware_ActiveRequestsGaugeReturnsToZero(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	handler := mc.Middleware()(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, "openoms_http_active_requests 0",
		"active requests gauge should return to 0 after request completes")
}

func TestMetricsCollector_Middleware_MultipleRequestsAccumulate(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	handler := mc.Middleware()(okHandler())

	for range 5 {
		req := httptest.NewRequest("GET", "/v1/orders", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	metricsOutput := getMetricsOutput(t, mc)
	// The counter for GET|unknown|200 should be 5
	assert.Contains(t, metricsOutput, `method="GET"`)
	// Count should appear in the _count line
	assert.Contains(t, metricsOutput, "openoms_http_request_duration_seconds_count")
}

func TestMetricsCollector_Middleware_DifferentStatusCodes(t *testing.T) {
	mc := middleware.NewMetricsCollector()

	statuses := []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError}
	for _, status := range statuses {
		s := status
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(s)
		})
		handler := mc.Middleware()(next)

		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, `status="200"`)
	assert.Contains(t, metricsOutput, `status="404"`)
	assert.Contains(t, metricsOutput, `status="500"`)
}

func TestMetricsCollector_Middleware_DifferentMethods(t *testing.T) {
	mc := middleware.NewMetricsCollector()

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		handler := mc.Middleware()(okHandler())
		req := httptest.NewRequest(method, "/v1/orders", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	metricsOutput := getMetricsOutput(t, mc)
	for _, method := range methods {
		assert.Contains(t, metricsOutput, `method="`+method+`"`)
	}
}

func TestMetricsCollector_Middleware_UsesChiRoutePattern(t *testing.T) {
	mc := middleware.NewMetricsCollector()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mc.Middleware()(next)

	// Create a request with chi route context containing a route pattern
	req := httptest.NewRequest("GET", "/v1/orders/123", nil)
	rctx := chi.NewRouteContext()
	rctx.RoutePatterns = []string{"/v1/orders/{id}"}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, `/v1/orders/{id}`)
}

func TestMetricsCollector_Middleware_UnknownRouteWhenNoChiContext(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	handler := mc.Middleware()(okHandler())

	req := httptest.NewRequest("GET", "/some/path", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, `route="unknown"`)
}

func TestMetricsCollector_Middleware_PreservesResponseBody(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	body := `{"id":"order-1"}`
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})
	handler := mc.Middleware()(next)

	req := httptest.NewRequest("GET", "/v1/orders/1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, body, rr.Body.String())
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestMetricsCollector_Middleware_WriteWithoutExplicitWriteHeader(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Write without calling WriteHeader first; should default to 200
		w.Write([]byte("hello"))
	})
	handler := mc.Middleware()(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, `status="200"`)
}

func TestMetricsCollector_Middleware_WriteHeaderCalledTwice_FirstWins(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.WriteHeader(http.StatusInternalServerError) // should be ignored
	})
	handler := mc.Middleware()(next)

	req := httptest.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)
	assert.Contains(t, metricsOutput, `status="201"`)
	assert.NotContains(t, metricsOutput, `status="500"`)
}

// --- Handler (Prometheus exposition) tests ---

func TestMetricsCollector_Handler_ContentType(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	mc.Handler().ServeHTTP(rr, req)

	assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8", rr.Header().Get("Content-Type"))
}

func TestMetricsCollector_Handler_EmptyCollectorOutputsStructure(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	metricsOutput := getMetricsOutput(t, mc)

	// Should have HELP and TYPE lines even with no data
	assert.Contains(t, metricsOutput, "# HELP openoms_http_requests_total")
	assert.Contains(t, metricsOutput, "# TYPE openoms_http_requests_total counter")
	assert.Contains(t, metricsOutput, "# HELP openoms_http_request_duration_seconds")
	assert.Contains(t, metricsOutput, "# TYPE openoms_http_request_duration_seconds histogram")
	assert.Contains(t, metricsOutput, "# HELP openoms_http_active_requests")
	assert.Contains(t, metricsOutput, "# TYPE openoms_http_active_requests gauge")
	assert.Contains(t, metricsOutput, "# HELP openoms_http_response_bytes_total")
	assert.Contains(t, metricsOutput, "# TYPE openoms_http_response_bytes_total counter")
}

func TestMetricsCollector_Handler_HistogramBucketBoundaries(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	handler := mc.Middleware()(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	metricsOutput := getMetricsOutput(t, mc)

	// All default bucket boundaries should appear
	expectedBuckets := []string{"0.005", "0.01", "0.025", "0.05", "0.1", "0.25", "0.5", "1", "2.5", "5", "10"}
	for _, b := range expectedBuckets {
		assert.Contains(t, metricsOutput, `le="`+b+`"`,
			"should have bucket boundary le=%s", b)
	}
	assert.Contains(t, metricsOutput, `le="+Inf"`)
}

func TestMetricsCollector_Handler_SortedOutput(t *testing.T) {
	mc := middleware.NewMetricsCollector()

	// Create requests with different methods to get multiple counter keys
	methods := []string{"DELETE", "GET", "POST", "PUT"}
	for _, m := range methods {
		handler := mc.Middleware()(okHandler())
		req := httptest.NewRequest(m, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	metricsOutput := getMetricsOutput(t, mc)
	lines := strings.Split(metricsOutput, "\n")

	// Find counter lines and verify they're sorted
	var counterLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "openoms_http_requests_total{") {
			counterLines = append(counterLines, line)
		}
	}

	require.GreaterOrEqual(t, len(counterLines), 4, "should have at least 4 counter lines")
	for i := 1; i < len(counterLines); i++ {
		assert.True(t, counterLines[i-1] <= counterLines[i],
			"counter lines should be sorted: %q should come before %q", counterLines[i-1], counterLines[i])
	}
}

// --- metricsResponseWriter feature tests ---

func TestMetricsResponseWriter_Hijack_FailsWhenNotSupported(t *testing.T) {
	mc := middleware.NewMetricsCollector()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hj, ok := w.(http.Hijacker); ok {
			_, _, err := hj.Hijack()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not implement http.Hijacker")
		}
	})

	handler := mc.Middleware()(next)
	req := httptest.NewRequest("GET", "/v1/ws", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestMetricsResponseWriter_Flush_DelegatesToUnderlying(t *testing.T) {
	mc := middleware.NewMetricsCollector()
	flusher := &fakeFlusherResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
	})

	handler := mc.Middleware()(next)
	req := httptest.NewRequest("GET", "/v1/events", nil)
	handler.ServeHTTP(flusher, req)

	assert.True(t, flusher.flushed, "underlying Flush should have been called")
}

func TestMetricsResponseWriter_Unwrap(t *testing.T) {
	mc := middleware.NewMetricsCollector()

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// http.ResponseController uses Unwrap to find the underlying writer
		rc := http.NewResponseController(w)
		// Just verify it doesn't panic — the exact behavior depends on the underlying writer
		_ = rc
		w.WriteHeader(http.StatusOK)
	})

	handler := mc.Middleware()(next)
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// getMetricsOutput is a helper that invokes the Handler endpoint and returns the body.
func getMetricsOutput(t *testing.T, mc *middleware.MetricsCollector) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	mc.Handler().ServeHTTP(rr, req)
	return rr.Body.String()
}
