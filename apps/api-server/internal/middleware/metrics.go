package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
)

// Default histogram bucket boundaries in seconds.
var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0}

// MetricsCollector collects HTTP request metrics in a Prometheus-compatible format.
type MetricsCollector struct {
	mu sync.RWMutex

	// Counter: total requests by method, route pattern, status code.
	requestsTotal map[string]*atomic.Int64

	// Histogram: request duration by method and route pattern.
	requestDuration map[string]*histogramData

	// Gauge: currently active (in-flight) requests.
	activeRequests atomic.Int64

	// Counter: total response bytes written.
	responseBytesTotal atomic.Int64
}

type histogramData struct {
	count   atomic.Int64
	sum     atomic.Int64 // stored as nanoseconds
	buckets []*bucket
}

type bucket struct {
	le    float64 // upper bound in seconds
	count atomic.Int64
}

func newHistogramData() *histogramData {
	h := &histogramData{
		buckets: make([]*bucket, len(defaultBuckets)),
	}
	for i, le := range defaultBuckets {
		h.buckets[i] = &bucket{le: le}
	}
	return h
}

// NewMetricsCollector creates a new MetricsCollector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requestsTotal:   make(map[string]*atomic.Int64),
		requestDuration: make(map[string]*histogramData),
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and bytes written.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// Unwrap supports http.ResponseController introduced in Go 1.20.
func (w *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Middleware returns an HTTP middleware that records request metrics.
func (mc *MetricsCollector) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			mc.activeRequests.Add(1)

			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			mc.activeRequests.Add(-1)
			duration := time.Since(start)

			// Use chi route pattern to avoid cardinality explosion.
			route := "unknown"
			if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
				route = rctx.RoutePattern()
			}

			method := r.Method
			status := strconv.Itoa(wrapped.statusCode)

			// Increment request counter.
			counterKey := method + "|" + route + "|" + status
			mc.getOrCreateCounter(counterKey).Add(1)

			// Record duration in histogram.
			histKey := method + "|" + route
			hist := mc.getOrCreateHistogram(histKey)
			durationSec := duration.Seconds()
			hist.sum.Add(duration.Nanoseconds())
			hist.count.Add(1)
			for _, b := range hist.buckets {
				if durationSec <= b.le {
					b.count.Add(1)
				}
			}

			// Accumulate response bytes.
			mc.responseBytesTotal.Add(int64(wrapped.bytesWritten))
		})
	}
}

func (mc *MetricsCollector) getOrCreateCounter(key string) *atomic.Int64 {
	mc.mu.RLock()
	c, ok := mc.requestsTotal[key]
	mc.mu.RUnlock()
	if ok {
		return c
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()
	// Double-check after acquiring write lock.
	if c, ok = mc.requestsTotal[key]; ok {
		return c
	}
	c = &atomic.Int64{}
	mc.requestsTotal[key] = c
	return c
}

func (mc *MetricsCollector) getOrCreateHistogram(key string) *histogramData {
	mc.mu.RLock()
	h, ok := mc.requestDuration[key]
	mc.mu.RUnlock()
	if ok {
		return h
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()
	if h, ok = mc.requestDuration[key]; ok {
		return h
	}
	h = newHistogramData()
	mc.requestDuration[key] = h
	return h
}

// Handler returns an http.HandlerFunc that serves the /metrics endpoint
// in Prometheus text exposition format.
func (mc *MetricsCollector) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		var b strings.Builder

		// --- openoms_http_requests_total (counter) ---
		b.WriteString("# HELP openoms_http_requests_total Total number of HTTP requests.\n")
		b.WriteString("# TYPE openoms_http_requests_total counter\n")

		mc.mu.RLock()
		counterKeys := make([]string, 0, len(mc.requestsTotal))
		for k := range mc.requestsTotal {
			counterKeys = append(counterKeys, k)
		}
		mc.mu.RUnlock()
		sort.Strings(counterKeys)

		for _, key := range counterKeys {
			mc.mu.RLock()
			c, ok := mc.requestsTotal[key]
			mc.mu.RUnlock()
			if !ok {
				continue
			}
			parts := strings.SplitN(key, "|", 3)
			if len(parts) != 3 {
				continue
			}
			method, route, status := parts[0], parts[1], parts[2]
			fmt.Fprintf(&b, "openoms_http_requests_total{method=%q,route=%q,status=%q} %d\n",
				method, route, status, c.Load())
		}

		// --- openoms_http_request_duration_seconds (histogram) ---
		b.WriteString("# HELP openoms_http_request_duration_seconds HTTP request duration in seconds.\n")
		b.WriteString("# TYPE openoms_http_request_duration_seconds histogram\n")

		mc.mu.RLock()
		histKeys := make([]string, 0, len(mc.requestDuration))
		for k := range mc.requestDuration {
			histKeys = append(histKeys, k)
		}
		mc.mu.RUnlock()
		sort.Strings(histKeys)

		for _, key := range histKeys {
			mc.mu.RLock()
			h, ok := mc.requestDuration[key]
			mc.mu.RUnlock()
			if !ok {
				continue
			}
			parts := strings.SplitN(key, "|", 2)
			if len(parts) != 2 {
				continue
			}
			method, route := parts[0], parts[1]

			for _, bkt := range h.buckets {
				fmt.Fprintf(&b, "openoms_http_request_duration_seconds_bucket{method=%q,route=%q,le=%q} %d\n",
					method, route, formatFloat(bkt.le), bkt.count.Load())
			}
			// +Inf bucket equals total count.
			fmt.Fprintf(&b, "openoms_http_request_duration_seconds_bucket{method=%q,route=%q,le=\"+Inf\"} %d\n",
				method, route, h.count.Load())
			fmt.Fprintf(&b, "openoms_http_request_duration_seconds_sum{method=%q,route=%q} %s\n",
				method, route, formatNanosAsSeconds(h.sum.Load()))
			fmt.Fprintf(&b, "openoms_http_request_duration_seconds_count{method=%q,route=%q} %d\n",
				method, route, h.count.Load())
		}

		// --- openoms_http_active_requests (gauge) ---
		b.WriteString("# HELP openoms_http_active_requests Number of currently active HTTP requests.\n")
		b.WriteString("# TYPE openoms_http_active_requests gauge\n")
		fmt.Fprintf(&b, "openoms_http_active_requests %d\n", mc.activeRequests.Load())

		// --- openoms_http_response_bytes_total (counter) ---
		b.WriteString("# HELP openoms_http_response_bytes_total Total bytes written in HTTP responses.\n")
		b.WriteString("# TYPE openoms_http_response_bytes_total counter\n")
		fmt.Fprintf(&b, "openoms_http_response_bytes_total %d\n", mc.responseBytesTotal.Load())

		w.Write([]byte(b.String())) //nolint:errcheck
	}
}

// formatFloat formats a float64 without unnecessary trailing zeros.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatNanosAsSeconds converts nanoseconds to a decimal seconds string.
func formatNanosAsSeconds(ns int64) string {
	sec := float64(ns) / 1e9
	return strconv.FormatFloat(sec, 'f', 6, 64)
}
