package middleware_test

import (
	"testing"
	"time"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// unreachableRedisOpts returns Redis client options that fail fast against a
// non-listening port. Uses 127.0.0.1 (not localhost) to avoid DNS lookup,
// MaxRetries=0 and short DialTimeout for fast test execution.
func unreachableRedisOpts() *redis.Options {
	return &redis.Options{
		Addr:        "127.0.0.1:1",
		MaxRetries:  0,
		DialTimeout: 100 * time.Millisecond,
	}
}

func TestRedisTokenBlacklist_NewReturnsNonNil(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer func() { _ = client.Close() }()

	bl := middleware.NewRedisTokenBlacklist(client)
	assert.NotNil(t, bl, "NewRedisTokenBlacklist should return a non-nil instance")
}

func TestRedisTokenBlacklist_ImplementsTokenBlacklistStore(_ *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer func() { _ = client.Close() }()

	bl := middleware.NewRedisTokenBlacklist(client)
	var _ middleware.TokenBlacklistStore = bl
}

func TestRedisTokenBlacklist_Revoke_NegativeTTLIsNoOp(t *testing.T) {
	// With a broken/unreachable client, Revoke with a past expiry should be a no-op
	// (the TTL check returns early before even touching Redis).
	client := redis.NewClient(unreachableRedisOpts())
	defer func() { _ = client.Close() }()

	bl := middleware.NewRedisTokenBlacklist(client)

	// Should not panic even with unreachable Redis when TTL is negative
	assert.NotPanics(t, func() {
		bl.Revoke("expired-token", time.Now().Add(-10*time.Second))
	})
}

func TestRedisTokenBlacklist_Revoke_ZeroTTLIsNoOp(t *testing.T) {
	client := redis.NewClient(unreachableRedisOpts())
	defer func() { _ = client.Close() }()

	bl := middleware.NewRedisTokenBlacklist(client)

	assert.NotPanics(t, func() {
		bl.Revoke("zero-ttl-token", time.Now())
	})
}

func TestRedisTokenBlacklist_IsRevoked_FailsClosedOnError(t *testing.T) {
	// With an unreachable Redis, IsRevoked must fail CLOSED (return true / deny)
	// within the circuit-breaker window, so a token revoked on another instance is
	// never accepted during a transient Redis outage (OPE-465).
	client := redis.NewClient(unreachableRedisOpts())
	defer func() { _ = client.Close() }()

	bl := middleware.NewRedisTokenBlacklist(client)
	result := bl.IsRevoked("any-token")
	assert.True(t, result, "should fail closed (return true / deny) when Redis is unreachable")
}

func TestRedisTokenBlacklist_Composite_FailsClosedForUnknownTokenWhenRedisDown(t *testing.T) {
	// Composite with an unreachable Redis primary and an empty in-memory fallback:
	// a token that was never revoked is still DENIED while Redis is down, because
	// the primary read fails closed within the breaker window (OPE-465).
	client := redis.NewClient(unreachableRedisOpts())
	defer func() { _ = client.Close() }()

	composite := middleware.NewCompositeTokenBlacklist(
		middleware.NewRedisTokenBlacklist(client),
		middleware.NewMemoryTokenBlacklist(),
	)
	assert.True(t, composite.IsRevoked("never-revoked-token"),
		"composite must fail closed when Redis is unreachable and the token is unknown")
}

func TestRedisTokenBlacklist_UsableInComposite(t *testing.T) {
	// Redis blacklist can be used as the primary store in a composite,
	// with in-memory as fallback. When Redis is down, the composite
	// still works through the fallback.
	client := redis.NewClient(unreachableRedisOpts())
	defer func() { _ = client.Close() }()

	redisBL := middleware.NewRedisTokenBlacklist(client)
	memoryBL := middleware.NewMemoryTokenBlacklist()
	composite := middleware.NewCompositeTokenBlacklist(redisBL, memoryBL)

	composite.Revoke("composite-token", time.Now().Add(1*time.Hour))

	// Even though Redis is down, the fallback (memory) should have the token
	assert.True(t, composite.IsRevoked("composite-token"),
		"composite should find token in memory fallback even when Redis is down")
}
