package middleware_test

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs replaces slog's default handler with a JSON handler that writes to
// the returned buffer, runs the provided function, then restores the original handler.
func captureLogs(fn func()) *bytes.Buffer {
	var buf bytes.Buffer
	original := slog.Default()
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(original)
	fn()
	return &buf
}

func TestLogging_CallsNextHandler(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/v1/orders", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	assert.True(t, called, "next handler should be called")
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Greater(t, buf.Len(), 0, "should have produced log output")
}

func TestLogging_LogsRequestMethod(t *testing.T) {
	handler := middleware.Logging(okHandler())
	req := httptest.NewRequest("POST", "/v1/orders", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"method":"POST"`)
}

func TestLogging_LogsRequestPath(t *testing.T) {
	handler := middleware.Logging(okHandler())
	req := httptest.NewRequest("GET", "/v1/products/123", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"path":"/v1/products/123"`)
}

func TestLogging_LogsStatusCode(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/v1/missing", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"status":404`)
}

func TestLogging_LogsDefaultStatusCode200(t *testing.T) {
	// When the handler doesn't explicitly call WriteHeader, default should be 200
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})
	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"status":200`)
}

func TestLogging_LogsDurationMs(t *testing.T) {
	handler := middleware.Logging(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"duration_ms"`)
}

func TestLogging_LogsRemoteAddr(t *testing.T) {
	handler := middleware.Logging(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"remote_addr":"192.168.1.100:54321"`)
}

func TestLogging_LogsMessage(t *testing.T) {
	handler := middleware.Logging(okHandler())
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	logOutput := buf.String()
	assert.Contains(t, logOutput, `"msg":"http request"`)
}

func TestLogging_LogsVariousStatusCodes(t *testing.T) {
	codes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusInternalServerError,
	}

	for _, code := range codes {
		t.Run(fmt.Sprintf("status_%d", code), func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(code)
			})
			handler := middleware.Logging(next)
			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()

			buf := captureLogs(func() {
				handler.ServeHTTP(rr, req)
			})

			logOutput := buf.String()
			assert.Contains(t, logOutput, fmt.Sprintf(`"status":%d`, code))
		})
	}
}

func TestLogging_LogsVariousMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := middleware.Logging(okHandler())
			req := httptest.NewRequest(method, "/v1/orders", nil)
			rr := httptest.NewRecorder()

			buf := captureLogs(func() {
				handler.ServeHTTP(rr, req)
			})

			logOutput := buf.String()
			assert.Contains(t, logOutput, fmt.Sprintf(`"method":"%s"`, method))
		})
	}
}

func TestLogging_WrittenStatusCodeCaptured(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("accepted"))
	})
	handler := middleware.Logging(next)
	req := httptest.NewRequest("POST", "/v1/shipments", nil)
	rr := httptest.NewRecorder()

	buf := captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusAccepted, rr.Code)
	assert.Contains(t, buf.String(), `"status":202`)
}

func TestLogging_PreservesResponseBody(t *testing.T) {
	body := `{"id":"order-1","status":"new"}`
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})
	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/v1/orders/1", nil)
	rr := httptest.NewRecorder()

	captureLogs(func() {
		handler.ServeHTTP(rr, req)
	})

	assert.Equal(t, body, rr.Body.String())
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

// --- wrappedWriter Hijack/Flush tests ---

// fakeHijackableResponseWriter implements http.ResponseWriter and http.Hijacker.
type fakeHijackableResponseWriter struct {
	http.ResponseWriter
	hijacked bool
}

func (f *fakeHijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.hijacked = true
	return nil, nil, nil
}

func TestLogging_WrappedWriter_HijackDelegatesToUnderlying(t *testing.T) {
	// We need to test that the wrappedWriter's Hijack method works.
	// The Logging middleware wraps the ResponseWriter; if the underlying supports
	// Hijack, the wrapped writer should delegate to it.
	hijackable := &fakeHijackableResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// The w passed here is the wrappedWriter from Logging middleware
		if hj, ok := w.(http.Hijacker); ok {
			_, _, err := hj.Hijack()
			require.NoError(t, err)
			called = true
		}
	})

	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/v1/ws", nil)
	handler.ServeHTTP(hijackable, req)

	assert.True(t, called, "Hijack should have been called through wrappedWriter")
	assert.True(t, hijackable.hijacked, "underlying Hijack should have been called")
}

func TestLogging_WrappedWriter_HijackFailsWhenNotSupported(t *testing.T) {
	// When the underlying ResponseWriter does NOT implement Hijacker,
	// calling Hijack should return an error.
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hj, ok := w.(http.Hijacker); ok {
			_, _, err := hj.Hijack()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "does not implement http.Hijacker")
		}
	})

	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/v1/ws", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

// fakeFlusherResponseWriter implements http.ResponseWriter and http.Flusher.
type fakeFlusherResponseWriter struct {
	http.ResponseWriter
	flushed bool
}

func (f *fakeFlusherResponseWriter) Flush() {
	f.flushed = true
}

func TestLogging_WrappedWriter_FlushDelegatesToUnderlying(t *testing.T) {
	flusher := &fakeFlusherResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
	})

	handler := middleware.Logging(next)
	req := httptest.NewRequest("GET", "/v1/events", nil)
	handler.ServeHTTP(flusher, req)

	assert.True(t, flusher.flushed, "underlying Flush should have been called")
}
