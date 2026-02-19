package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisWSTicketStore struct {
	client *redis.Client
}

func NewRedisWSTicketStore(client *redis.Client) *RedisWSTicketStore {
	return &RedisWSTicketStore{client: client}
}

func (r *RedisWSTicketStore) Store(ctx context.Context, key string, data WSTicketData, ttl time.Duration) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, val, ttl).Err()
}

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
