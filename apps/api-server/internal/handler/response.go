package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	if status >= 500 {
		slog.Error("server error response", "status", status, "message", message) //nolint:gosec
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("failed to encode JSON error response", "error", err)
	}
}

// writeServerError logs the underlying error, reports it to Sentry, and returns a generic message to the client.
func writeServerError(w http.ResponseWriter, message string, err error) {
	slog.Error(message, "error", err)
	sentry.CaptureException(err)
	writeError(w, http.StatusInternalServerError, message)
}

// writeCSVHeaders sets standard headers for CSV file downloads and writes the UTF-8 BOM for Excel compatibility.
func writeCSVHeaders(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
}

// decodeJSON decodes the JSON request body into dst. On failure it writes a
// 400 "invalid request body" response and returns false; the caller should
// simply return when this returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// parseIDParam parses the named chi URL parameter as a UUID. On failure it
// writes a 400 "invalid <resource> id" response and returns ok=false; the
// caller should simply return when ok is false.
func parseIDParam(w http.ResponseWriter, r *http.Request, param, resource string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+resource+" id")
		return uuid.Nil, false
	}
	return id, true
}

// queryInt parses an integer query parameter, returning def when absent or invalid.
func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// queryFloat parses a float query parameter, returning def when absent or invalid.
func queryFloat(r *http.Request, key string, def float64) float64 {
	if v := r.URL.Query().Get(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func isValidationError(err error) bool {
	var ve *service.ValidationError
	return errors.As(err, &ve)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
