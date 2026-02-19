package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter defines the interface for rate limiting backends.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimitWith creates rate limiting middleware using the provided RateLimiter.
func RateLimitWith(limiter RateLimiter, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			allowed, err := limiter.Allow(r.Context(), ip, maxRequests, window)
			if err != nil {
				// Fail open on error
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"too many requests"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit is the legacy constructor using in-memory backend.
// Kept for backward compatibility with existing router.go calls.
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimitWith(NewMemoryRateLimiter(), maxRequests, window)
}

// --- Memory implementation ---

type memoryEntry struct {
	count     int
	resetTime time.Time
}

// MemoryRateLimiter is an in-memory rate limiter for development/single-instance use.
type MemoryRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*memoryEntry
}

// NewMemoryRateLimiter creates a new in-memory rate limiter with background cleanup.
func NewMemoryRateLimiter() *MemoryRateLimiter {
	m := &MemoryRateLimiter{entries: make(map[string]*memoryEntry)}
	go m.cleanup()
	return m
}

// Allow checks if a request is within the rate limit for the given key.
func (m *MemoryRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	entry, exists := m.entries[key]
	if !exists || now.After(entry.resetTime) {
		m.entries[key] = &memoryEntry{count: 1, resetTime: now.Add(window)}
		return true, nil
	}

	entry.count++
	return entry.count <= limit, nil
}

func (m *MemoryRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		m.mu.Lock()
		now := time.Now()
		for key, entry := range m.entries {
			if now.After(entry.resetTime) {
				delete(m.entries, key)
			}
		}
		m.mu.Unlock()
	}
}
