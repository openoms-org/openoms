package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements RateLimiter using Redis INCR + EXPIRE.
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a Redis-backed rate limiter.
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// Allow checks if a request is within the rate limit using a Lua script for atomicity.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	script := redis.NewScript(`
		local count = redis.call('INCR', KEYS[1])
		if count == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		return count
	`)

	count, err := script.Run(ctx, r.client, []string{redisKey}, int(window.Seconds())).Int64()
	if err != nil {
		return false, err
	}
	return count <= int64(limit), nil
}
