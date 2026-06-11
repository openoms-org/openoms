package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

// --- MemoryRateLimiter.Allow tests ---

func TestMemoryRateLimiter_Allow_FirstRequestAllowed(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	allowed, err := limiter.Allow(context.Background(), "user1", 5, time.Minute)
	assert.NoError(t, err)
	assert.True(t, allowed, "first request should be allowed")
}

func TestMemoryRateLimiter_Allow_UpToLimitAllowed(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 10

	for i := 1; i <= limit; i++ {
		allowed, err := limiter.Allow(ctx, "user1", limit, time.Minute)
		assert.NoError(t, err)
		assert.True(t, allowed, "request %d of %d should be allowed", i, limit)
	}
}

func TestMemoryRateLimiter_Allow_ExceedLimitBlocked(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 3

	// Use up the limit
	for range limit {
		allowed, err := limiter.Allow(ctx, "user1", limit, time.Minute)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// Request limit+1 should be blocked
	allowed, err := limiter.Allow(ctx, "user1", limit, time.Minute)
	assert.NoError(t, err)
	assert.False(t, allowed, "request exceeding limit should be blocked")
}

func TestMemoryRateLimiter_Allow_MultipleExceedAllBlocked(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 2

	for range limit {
		_, _ = limiter.Allow(ctx, "key", limit, time.Minute)
	}

	// All subsequent requests should be blocked
	for i := range 5 {
		allowed, err := limiter.Allow(ctx, "key", limit, time.Minute)
		assert.NoError(t, err)
		assert.False(t, allowed, "request %d after limit should be blocked", i+1)
	}
}

func TestMemoryRateLimiter_Allow_DifferentKeysIndependent(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 1

	// Exhaust key1
	allowed, _ := limiter.Allow(ctx, "key1", limit, time.Minute)
	assert.True(t, allowed)

	blocked, _ := limiter.Allow(ctx, "key1", limit, time.Minute)
	assert.False(t, blocked, "key1 should be rate limited")

	// key2 should still be allowed
	allowed2, err := limiter.Allow(ctx, "key2", limit, time.Minute)
	assert.NoError(t, err)
	assert.True(t, allowed2, "key2 should be independent from key1")
}

func TestMemoryRateLimiter_Allow_ManyKeysIndependent(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 2

	keys := []string{"192.168.1.1", "10.0.0.1", "172.16.0.1", "8.8.8.8", "1.1.1.1"}
	for _, key := range keys {
		for i := range limit {
			allowed, err := limiter.Allow(ctx, key, limit, time.Minute)
			assert.NoError(t, err)
			assert.True(t, allowed, "key %s request %d should be allowed", key, i+1)
		}
		blocked, _ := limiter.Allow(ctx, key, limit, time.Minute)
		assert.False(t, blocked, "key %s should be blocked after limit", key)
	}
}

func TestMemoryRateLimiter_Allow_WindowExpiry(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 1
	window := 50 * time.Millisecond

	// Use up the limit
	allowed, _ := limiter.Allow(ctx, "user1", limit, window)
	assert.True(t, allowed)

	blocked, _ := limiter.Allow(ctx, "user1", limit, window)
	assert.False(t, blocked, "should be blocked within window")

	// Wait for the window to expire
	time.Sleep(60 * time.Millisecond)

	// After window expiry, should be allowed again
	allowed2, err := limiter.Allow(ctx, "user1", limit, window)
	assert.NoError(t, err)
	assert.True(t, allowed2, "should be allowed after window expiry")
}

func TestMemoryRateLimiter_Allow_WindowExpiryResetsCount(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 3
	window := 50 * time.Millisecond

	// Use all requests in window
	for range limit {
		_, _ = limiter.Allow(ctx, "user1", limit, window)
	}
	blocked, _ := limiter.Allow(ctx, "user1", limit, window)
	assert.False(t, blocked)

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	// Full limit should be available again
	for i := range limit {
		allowed, err := limiter.Allow(ctx, "user1", limit, window)
		assert.NoError(t, err)
		assert.True(t, allowed, "request %d after window reset should be allowed", i+1)
	}
}

func TestMemoryRateLimiter_Allow_LimitOfOne(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "strict", 1, time.Minute)
	assert.NoError(t, err)
	assert.True(t, allowed, "first request with limit=1 should be allowed")

	blocked, err := limiter.Allow(ctx, "strict", 1, time.Minute)
	assert.NoError(t, err)
	assert.False(t, blocked, "second request with limit=1 should be blocked")
}

func TestMemoryRateLimiter_Allow_ErrorIsAlwaysNil(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()

	// The in-memory implementation should never return an error
	for range 20 {
		_, err := limiter.Allow(ctx, "any", 5, time.Minute)
		assert.NoError(t, err, "in-memory limiter should never return errors")
	}
}

// --- Concurrency safety ---

func TestMemoryRateLimiter_Allow_ConcurrentAccess(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 100
	goroutines := 50
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	var allowedCount atomic.Int64

	for range goroutines {
		wg.Go(func() {
			for range requestsPerGoroutine {
				allowed, err := limiter.Allow(ctx, "shared-key", limit, time.Minute)
				assert.NoError(t, err)
				if allowed {
					allowedCount.Add(1)
				}
			}
		})
	}

	wg.Wait()

	totalRequests := goroutines * requestsPerGoroutine // 500
	allowed := allowedCount.Load()
	// Exactly `limit` requests should be allowed
	assert.Equal(t, int64(limit), allowed,
		"exactly %d of %d concurrent requests should be allowed", limit, totalRequests)
}

func TestMemoryRateLimiter_Allow_ConcurrentDifferentKeys(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	limit := 5
	numKeys := 20

	var wg sync.WaitGroup

	for k := range numKeys {
		wg.Add(1)
		go func(keyIdx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", keyIdx)
			for i := range limit {
				allowed, err := limiter.Allow(ctx, key, limit, time.Minute)
				assert.NoError(t, err)
				assert.True(t, allowed, "request %d for %s should be allowed", i+1, key)
			}
			// Next request for this key should be blocked
			blocked, err := limiter.Allow(ctx, key, limit, time.Minute)
			assert.NoError(t, err)
			assert.False(t, blocked, "%s should be blocked after limit", key)
		}(k)
	}

	wg.Wait()
}

// --- RateLimitWith middleware integration tests ---

func testOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
}

func TestRateLimitWith_AllowedRequestsPassThrough(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 5, time.Minute)(testOKHandler())

	for i := range 5 {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.RemoteAddr = "203.0.113.50:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}
}

func TestRateLimitWith_ExceededReturns429(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 2, time.Minute)(testOKHandler())

	ip := "198.51.100.1:54321"
	// Exhaust limit
	for range 2 {
		req := httptest.NewRequest("POST", "/v1/auth/login", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Third request should be rate limited
	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	req.RemoteAddr = ip
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, "60", rr.Header().Get("Retry-After"))
	assert.JSONEq(t, `{"error":"too many requests"}`, rr.Body.String())
}

func TestRateLimitWith_ChangingSpoofedXForwardedForDoesNotBypassUntrustedPeer(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.42.0.0/16")}
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.TrustedRealIP(trusted)(
		middleware.RateLimitWith(limiter, 2, time.Minute)(testOKHandler()),
	)

	spoofedIPs := []string{"198.51.100.10", "198.51.100.11", "198.51.100.12"}
	for i, spoofedIP := range spoofedIPs {
		req := httptest.NewRequest("POST", "/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.50:12345"
		req.Header.Set("X-Forwarded-For", spoofedIP)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if i < 2 {
			assert.Equal(t, http.StatusOK, rr.Code, "request %d should be allowed", i+1)
			continue
		}
		assert.Equal(t, http.StatusTooManyRequests, rr.Code, "spoofed X-Forwarded-For must not reset the limiter key")
	}
}

func TestRateLimitWith_DifferentIPsAreIndependent(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 1, time.Minute)(testOKHandler())

	ips := []string{"10.0.0.1:1111", "10.0.0.2:2222", "10.0.0.3:3333"}
	for _, ip := range ips {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "first request from %s should pass", ip)
	}
}

func TestRateLimitWith_RemoteAddrWithoutPort(t *testing.T) {
	// If RemoteAddr has no port (unlikely but handled), it should still work
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitWith(limiter, 1, time.Minute)(testOKHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Second request from same "IP" should be blocked
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.0.2.1"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rr2.Code)
}

// --- Failing-open behavior (limiter returns error) ---

type errorLimiter struct{}

func (e *errorLimiter) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, error) {
	return false, fmt.Errorf("redis connection refused")
}

func TestRateLimitWith_LimiterErrorFailsOpen(t *testing.T) {
	handler := middleware.RateLimitWith(&errorLimiter{}, 1, time.Minute)(testOKHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// When the limiter returns an error, the middleware should fail open (allow request)
	assert.Equal(t, http.StatusOK, rr.Code, "should fail open when limiter errors")
}

// --- Failing-closed behavior (critical routes) ---

func TestRateLimitCriticalWith_LimiterErrorFailsClosed(t *testing.T) {
	handler := middleware.RateLimitCriticalWith(&errorLimiter{}, 10, time.Minute)(testOKHandler())

	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// On a limiter backend error a security-critical route must DENY the request.
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code, "should fail closed when limiter errors")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(t, "60", rr.Header().Get("Retry-After"))
	assert.JSONEq(t, `{"error":"rate limiter unavailable"}`, rr.Body.String())
}

func TestRateLimitCriticalWith_AllowedRequestsPassThrough(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitCriticalWith(limiter, 5, time.Minute)(testOKHandler())

	for i := range 5 {
		req := httptest.NewRequest("POST", "/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.50:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}
}

func TestRateLimitCriticalWith_ExceededReturns429(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitCriticalWith(limiter, 1, time.Minute)(testOKHandler())

	ip := "198.51.100.7:54321"
	req := httptest.NewRequest("POST", "/v1/auth/login", nil)
	req.RemoteAddr = ip
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Exceeding the limit (not a backend error) still returns the normal 429,
	// not the fail-closed 503.
	req = httptest.NewRequest("POST", "/v1/auth/login", nil)
	req.RemoteAddr = ip
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.JSONEq(t, `{"error":"too many requests"}`, rr.Body.String())
}

func TestRateLimitByAuthenticatedUserWith_UsesUserIdentityAcrossIPs(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitByAuthenticatedUserWith(limiter, 1, time.Minute)(testOKHandler())
	userID := uuid.New()

	req := httptest.NewRequest("PATCH", "/v1/users/me/password", nil)
	req.RemoteAddr = "198.51.100.10:1000"
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req = httptest.NewRequest("PATCH", "/v1/users/me/password", nil)
	req.RemoteAddr = "198.51.100.11:1001"
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestRateLimitByAuthenticatedUserWith_FallsBackToIPWithoutUser(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	handler := middleware.RateLimitByAuthenticatedUserWith(limiter, 1, time.Minute)(testOKHandler())

	req := httptest.NewRequest("PATCH", "/v1/users/me/password", nil)
	req.RemoteAddr = "203.0.113.20:1000"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req = httptest.NewRequest("PATCH", "/v1/users/me/password", nil)
	req.RemoteAddr = "203.0.113.20:1001"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// --- RateLimitWith with in-memory limiter (end-to-end) ---

func TestRateLimitWith_MemoryLimiterEndToEnd(t *testing.T) {
	handler := middleware.RateLimitWith(middleware.NewMemoryRateLimiter(), 2, time.Minute)(testOKHandler())

	ip := "203.0.113.99:9999"

	// First two should pass
	for range 2 {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Third should be blocked
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

// --- RateLimiter interface compliance ---

func TestMemoryRateLimiter_ImplementsInterface(_ *testing.T) {
	var _ middleware.RateLimiter = middleware.NewMemoryRateLimiter()
}
