package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLoginLockoutStore implements LoginLockoutStore using Redis.
type RedisLoginLockoutStore struct {
	client *redis.Client
}

// NewRedisLoginLockoutStore creates a new Redis-backed lockout store.
func NewRedisLoginLockoutStore(client *redis.Client) *RedisLoginLockoutStore {
	return &RedisLoginLockoutStore{client: client}
}

// IncrFailures atomically increments the failure counter in Redis and sets the expiration window.
func (r *RedisLoginLockoutStore) IncrFailures(ctx context.Context, key string, window time.Duration) (int, error) {
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

// ResetFailures deletes the failure counter key from Redis.
func (r *RedisLoginLockoutStore) ResetFailures(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// GetLockoutRemaining checks whether a lockout key exists in Redis and returns its remaining TTL.
func (r *RedisLoginLockoutStore) GetLockoutRemaining(ctx context.Context, key string) (time.Duration, error) {
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil || exists == 0 { //nolint:nilerr // fail open on Redis error — lockout is best-effort
		return 0, nil // not locked
	}
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 { //nolint:nilerr // fail open on Redis error — lockout is best-effort
		return 0, nil
	}
	return ttl, nil
}

// SetLockout creates a lockout key in Redis with the given duration as TTL.
func (r *RedisLoginLockoutStore) SetLockout(ctx context.Context, key string, duration time.Duration) error {
	return r.client.Set(ctx, key, "locked", duration).Err()
}
