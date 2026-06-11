package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/asyncutil"
)

// RateLimiter defines the interface for rate limiting backends.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimitWith creates rate limiting middleware using the provided RateLimiter.
// On a limiter backend error it FAILS OPEN (allows the request) to preserve
// availability for non-critical routes. Use RateLimitCriticalWith for
// security-critical endpoints that must fail closed instead.
func RateLimitWith(limiter RateLimiter, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimitByIP(limiter, maxRequests, window, false)
}

// RateLimitCriticalWith creates rate limiting middleware that FAILS CLOSED: on a
// limiter backend error it denies the request (503 Service Unavailable) instead
// of allowing it. Use for security-critical unauthenticated endpoints (login,
// 2FA login, token refresh, registration, checkout creation) where availability
// must not silently disable brute-force / abuse protection.
func RateLimitCriticalWith(limiter RateLimiter, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimitByIP(limiter, maxRequests, window, true)
}

// rateLimitByIP is the shared per-IP rate limiting implementation. failClosed
// selects the behavior when the limiter backend returns an error: when true the
// request is denied (503), when false it is allowed (fail open).
func rateLimitByIP(limiter RateLimiter, maxRequests int, window time.Duration, failClosed bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIPForRateLimit(r)
			route := routePatternForRateLimit(r)
			key := fmt.Sprintf("rl:%s:%s:%d", ip, route, maxRequests)
			allowed, err := limiter.Allow(r.Context(), key, maxRequests, window)
			if err != nil {
				if failClosed {
					slog.Error("rate limiter error, failing closed", "error", err, "ip", ip) // #nosec G706 -- ip is logged as a structured field, not interpolated
					writeRateLimitUnavailable(w)
					return
				}
				slog.Error("rate limiter error, failing open", "error", err, "ip", ip) // #nosec G706 -- ip is logged as a structured field, not interpolated
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				writeRateLimitExceeded(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByAuthenticatedUserWith rate limits by authenticated user ID,
// falling back to client IP when auth context has not been populated.
func RateLimitByAuthenticatedUserWith(limiter RateLimiter, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := routePatternForRateLimit(r)
			userID := UserIDFromContext(r.Context())
			identity := clientIPForRateLimit(r)
			identityType := "ip"
			if userID != uuid.Nil {
				identity = userID.String()
				identityType = "user"
			}

			key := fmt.Sprintf("rl:%s:%s:%s:%d", identityType, identity, route, maxRequests)
			allowed, err := limiter.Allow(r.Context(), key, maxRequests, window)
			if err != nil {
				slog.Error("rate limiter error, failing open", "error", err, "identity_type", identityType)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				writeRateLimitExceeded(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIPForRateLimit(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func routePatternForRateLimit(r *http.Request) string {
	// Include route pattern in the key so endpoints with the same limit
	// get separate counters (e.g., login and register both at 10/min).
	route := r.URL.Path
	if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
		route = rctx.RoutePattern()
	}
	return route
}

func writeRateLimitExceeded(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"too many requests"}`))
}

// writeRateLimitUnavailable is the fail-closed response: the rate-limit backend
// is unavailable, so the request is refused rather than admitted unprotected.
func writeRateLimitUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"rate limiter unavailable"}`))
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
	done    chan struct{}
}

// NewMemoryRateLimiter creates a new in-memory rate limiter with background cleanup.
func NewMemoryRateLimiter() *MemoryRateLimiter {
	m := &MemoryRateLimiter{
		entries: make(map[string]*memoryEntry),
		done:    make(chan struct{}),
	}
	asyncutil.SafeGo(func() { m.cleanup() })
	return m
}

// Close stops the background cleanup goroutine.
func (m *MemoryRateLimiter) Close() {
	close(m.done)
}

// Allow checks if a request is within the rate limit for the given key.
func (m *MemoryRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	entry, exists := m.entries[key]
	if !exists || now.After(entry.resetTime) {
		m.entries[key] = &memoryEntry{count: 1, resetTime: now.Add(window)}
		return 1 <= limit, nil
	}

	entry.count++
	return entry.count <= limit, nil
}

func (m *MemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for key, entry := range m.entries {
				if now.After(entry.resetTime) {
					delete(m.entries, key)
				}
			}
			m.mu.Unlock()
		case <-m.done:
			return
		}
	}
}
