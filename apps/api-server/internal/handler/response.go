package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/getsentry/sentry-go"
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
		slog.Error("server error response", "status", status, "message", message)
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
