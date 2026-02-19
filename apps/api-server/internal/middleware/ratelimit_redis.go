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

func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	count, err := r.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		r.client.Expire(ctx, redisKey, window)
	}

	return count <= int64(limit), nil
}
