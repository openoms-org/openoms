package middleware

import (
	"encoding/json"
	"net/http"
)

// writeJSONError writes a JSON error response of the form {"error": message}
// with the given status code. It is the shared helper for the single-field
// error responses used across the middleware package. Variants that need a
// different body shape (e.g. plan_guard's two-field {error, message}) or
// additional headers (e.g. ratelimit's Retry-After) deliberately do not use it.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
