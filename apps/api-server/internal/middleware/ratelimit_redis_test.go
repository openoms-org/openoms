package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// unreachableRedisClient returns a Redis client that fails fast against a
// non-listening port. Uses 127.0.0.1 (not localhost) to avoid DNS lookup,
// DialTimeout + MaxRetries=0 for fast test execution.
func unreachableRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		MaxRetries:  0,
		DialTimeout: 100 * time.Millisecond,
	})
}

func TestRedisRateLimiter_NewReturnsNonNil(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	assert.NotNil(t, limiter, "NewRedisRateLimiter should return a non-nil instance")
}

func TestRedisRateLimiter_ImplementsRateLimiterInterface(_ *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	var _ middleware.RateLimiter = limiter
}

func TestRedisRateLimiter_Allow_ReturnsErrorOnUnreachableRedis(t *testing.T) {
	client := unreachableRedisClient()
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	allowed, err := limiter.Allow(context.Background(), "test-key", 10, time.Minute)
	assert.Error(t, err, "should return error when Redis is unreachable")
	assert.False(t, allowed, "should return false when error occurs")
}

func TestRedisRateLimiter_Allow_ContextCancelled(t *testing.T) {
	client := unreachableRedisClient()
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	allowed, err := limiter.Allow(ctx, "test-key", 10, time.Minute)
	assert.Error(t, err, "should return error with cancelled context")
	assert.False(t, allowed)
}

func TestRedisRateLimiter_UsableInRateLimitWithMiddleware_FailsOpen(t *testing.T) {
	// When Redis is down, the RateLimitWith middleware should fail open
	// because the limiter returns an error.
	client := unreachableRedisClient()
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	handler := middleware.RateLimitWith(limiter, 1, time.Minute)(okHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code,
		"when Redis rate limiter errors, middleware should fail open and allow the request")
}

func TestRedisRateLimiter_Allow_ConnectionErrorType(t *testing.T) {
	// Verify the error is a dial/connection error, not a Lua script error.
	client := unreachableRedisClient()
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	_, err := limiter.Allow(context.Background(), "192.168.1.1", 100, 5*time.Minute)
	assert.Error(t, err)
	// The error could be "connection refused" or "i/o timeout" depending on the OS;
	// the key invariant is that it's a dial-level error, not a Lua script error.
	assert.Contains(t, err.Error(), "dial",
		"error should be a dial-level error, not a Lua script error")
}

func TestRedisRateLimiter_MultipleCallsAllReturnErrors(t *testing.T) {
	client := unreachableRedisClient()
	defer client.Close()

	limiter := middleware.NewRedisRateLimiter(client)
	ctx := context.Background()

	for i := range 5 {
		allowed, err := limiter.Allow(ctx, "key", 10, time.Minute)
		assert.Error(t, err, "call %d should return error", i+1)
		assert.False(t, allowed, "call %d should return false on error", i+1)
	}
}
