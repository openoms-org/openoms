package middleware

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Default circuit-breaker tuning for the Redis blacklist.
//
// On a Redis read error the blacklist fails CLOSED (treats the token as
// revoked / denies it) so a revoked token from another instance is never
// accepted during a transient blip. To avoid a sustained outage locking
// every session out for the whole token TTL, a circuit breaker trips after
// the failures persist past redisBreakerOpenAfter, after which the blacklist
// reverts to fail-OPEN. While the breaker is open, locally-revoked tokens are
// still denied by the in-memory fallback in CompositeTokenBlacklist (populated
// on every Revoke). Any successful read closes the breaker again.
const (
	redisBlacklistReadTimeout = 2 * time.Second
	// redisBreakerOpenAfter is how long Redis must be continuously failing
	// before the breaker opens and reads start failing open again.
	redisBreakerOpenAfter = 30 * time.Second
)

// RedisTokenBlacklist implements TokenBlacklistStore using Redis SET with TTL.
//
// Reads fail CLOSED (deny) on error, bounded by a circuit breaker so a
// sustained outage does not deny every request indefinitely.
type RedisTokenBlacklist struct {
	client          *redis.Client
	readTimeout     time.Duration
	breakerOpenedAt time.Duration

	mu             sync.Mutex
	firstFailureAt time.Time // zero when the last read succeeded
}

// NewRedisTokenBlacklist creates a Redis-backed token blacklist.
func NewRedisTokenBlacklist(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{
		client:          client,
		readTimeout:     redisBlacklistReadTimeout,
		breakerOpenedAt: redisBreakerOpenAfter,
	}
}

// Revoke adds a token hash to the Redis blacklist with a TTL.
func (r *RedisTokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return
	}
	r.client.Set(context.Background(), "blacklist:"+tokenHash, "1", ttl)
}

// IsRevoked checks if a token hash exists in the Redis blacklist.
//
// On a Redis read error it fails CLOSED (returns true, deny) so a token
// revoked on another instance is never accepted during a transient outage.
// A circuit breaker bounds this: once Redis has been continuously failing
// for longer than breakerOpenedAt the method fails OPEN (returns false) to
// avoid locking every session out for the full token TTL. Locally-revoked
// tokens remain denied via the in-memory fallback in the composite store.
func (r *RedisTokenBlacklist) IsRevoked(tokenHash string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), r.readTimeout)
	defer cancel()

	result, err := r.client.Exists(ctx, "blacklist:"+tokenHash).Result()
	if err != nil {
		return r.onReadError(err)
	}
	r.onReadSuccess()
	return result > 0
}

// onReadSuccess closes the breaker after a successful Redis read.
func (r *RedisTokenBlacklist) onReadSuccess() {
	r.mu.Lock()
	r.firstFailureAt = time.Time{}
	r.mu.Unlock()
}

// onReadError records the failure and decides fail-closed vs fail-open.
// Returns true (deny) while inside the breaker window, false (allow) once the
// breaker has opened due to a sustained outage.
func (r *RedisTokenBlacklist) onReadError(err error) bool {
	now := time.Now()

	r.mu.Lock()
	if r.firstFailureAt.IsZero() {
		r.firstFailureAt = now
	}
	breakerOpen := now.Sub(r.firstFailureAt) >= r.breakerOpenedAt
	r.mu.Unlock()

	if breakerOpen {
		// Sustained outage: revert to fail-open so auth is not bricked for the
		// full token TTL. The composite in-memory fallback still denies tokens
		// revoked by this instance.
		slog.Warn("token blacklist Redis read failing; circuit breaker open, failing open",
			"error", err)
		return false
	}

	// Transient error: fail closed (deny) so revoked tokens stay revoked.
	slog.Warn("token blacklist Redis read error; failing closed (deny)", "error", err)
	return true
}
