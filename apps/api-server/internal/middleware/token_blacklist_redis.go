package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisTokenBlacklist implements TokenBlacklistStore using Redis SET with TTL.
type RedisTokenBlacklist struct {
	client *redis.Client
}

// NewRedisTokenBlacklist creates a Redis-backed token blacklist.
func NewRedisTokenBlacklist(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{client: client}
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
func (r *RedisTokenBlacklist) IsRevoked(tokenHash string) bool {
	result, err := r.client.Exists(context.Background(), "blacklist:"+tokenHash).Result()
	if err != nil {
		return false // fail open
	}
	return result > 0
}
