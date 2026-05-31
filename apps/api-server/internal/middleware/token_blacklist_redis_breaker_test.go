package middleware

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The Redis token blacklist must fail CLOSED (deny) on a read error so a token
// revoked on another instance is never accepted during a transient Redis blip,
// but a circuit breaker reverts to fail-open after a sustained outage so auth is
// not bricked for the full token TTL. These tests exercise that logic directly
// (no Redis required).

var errBlacklistRead = errors.New("redis read failed")

func TestRedisTokenBlacklist_OnReadError_FailsClosedWithinWindow(t *testing.T) {
	bl := &RedisTokenBlacklist{breakerOpenedAt: time.Hour}

	denied := bl.onReadError(errBlacklistRead)

	assert.True(t, denied, "a transient read error must fail closed (deny the token)")
	assert.False(t, bl.firstFailureAt.IsZero(), "first failure time must be recorded")
}

func TestRedisTokenBlacklist_OnReadError_FailsOpenAfterSustainedOutage(t *testing.T) {
	bl := &RedisTokenBlacklist{breakerOpenedAt: 30 * time.Second}
	// Simulate Redis having been continuously failing for longer than the window.
	bl.firstFailureAt = time.Now().Add(-time.Minute)

	denied := bl.onReadError(errBlacklistRead)

	assert.False(t, denied, "a sustained outage must open the breaker and fail open")
}

func TestRedisTokenBlacklist_OnReadSuccess_ClosesBreaker(t *testing.T) {
	bl := &RedisTokenBlacklist{breakerOpenedAt: 30 * time.Second}
	bl.onReadError(errBlacklistRead)
	assert.False(t, bl.firstFailureAt.IsZero(), "precondition: failure recorded")

	bl.onReadSuccess()

	assert.True(t, bl.firstFailureAt.IsZero(), "a successful read must reset the failure window")
}
