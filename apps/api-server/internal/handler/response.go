package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("failed to encode JSON error response", "error", err)
	}
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
