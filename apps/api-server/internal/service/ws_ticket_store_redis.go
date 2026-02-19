package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisWSTicketStore implements WSTicketStore using Redis with automatic TTL expiration.
// Safe for multi-pod deployments.
type RedisWSTicketStore struct {
	client *redis.Client
}

// NewRedisWSTicketStore creates a new Redis-backed WebSocket ticket store.
func NewRedisWSTicketStore(client *redis.Client) *RedisWSTicketStore {
	return &RedisWSTicketStore{client: client}
}

// Store saves a ticket with associated data and TTL in Redis.
func (r *RedisWSTicketStore) Store(ctx context.Context, key string, data WSTicketData, ttl time.Duration) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, val, ttl).Err()
}

// Consume retrieves and atomically deletes a ticket from Redis. Returns nil if not found.
func (r *RedisWSTicketStore) Consume(ctx context.Context, key string) (*WSTicketData, error) {
	// GET + DEL atomically via Lua script
	script := redis.NewScript(`
		local val = redis.call('GET', KEYS[1])
		if val then
			redis.call('DEL', KEYS[1])
		end
		return val
	`)
	result, err := script.Run(ctx, r.client, []string{key}).Result()
	if err == redis.Nil || result == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var data WSTicketData
	if err := json.Unmarshal([]byte(result.(string)), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
