package middleware

import "net/http"

// MaxBodySize wraps r.Body with http.MaxBytesReader to enforce a maximum
// request body size. Returns 413 Request Entity Too Large when exceeded.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
