package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// RateLimit provides a simple in-memory per-IP rate limiter.
// It allows maxRequests per window duration. Returns 429 Too Many Requests
// when the limit is exceeded.
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	entries := make(map[string]*rateLimitEntry)

	// Background cleanup of stale entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, entry := range entries {
				if now.After(entry.resetTime) {
					delete(entries, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			mu.Lock()
			now := time.Now()
			entry, exists := entries[ip]
			if !exists || now.After(entry.resetTime) {
				entries[ip] = &rateLimitEntry{count: 1, resetTime: now.Add(window)}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			entry.count++
			if entry.count > maxRequests {
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"too many requests"}`))
				return
			}
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
