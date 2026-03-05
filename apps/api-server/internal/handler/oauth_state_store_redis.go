package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const oauthStateKeyPrefix = "oauth_state:"

// RedisOAuthStateStore stores OAuth state in Redis with automatic TTL expiration.
// Safe for multi-pod deployments.
type RedisOAuthStateStore struct {
	client *redis.Client
}

// NewRedisOAuthStateStore creates a new Redis-backed OAuth state store.
func NewRedisOAuthStateStore(client *redis.Client) *RedisOAuthStateStore {
	return &RedisOAuthStateStore{client: client}
}

// Save stores the OAuth state data in Redis with the given TTL.
func (r *RedisOAuthStateStore) Save(ctx context.Context, state string, data *OAuthState, ttl time.Duration) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, oauthStateKeyPrefix+state, val, ttl).Err()
}

// getAndDeleteScript atomically fetches and deletes a key, preventing replay attacks.
var getAndDeleteScript = redis.NewScript(`
	local val = redis.call('GET', KEYS[1])
	if val then redis.call('DEL', KEYS[1]) end
	return val
`)

// Load retrieves and atomically deletes the OAuth state data for the given state key.
func (r *RedisOAuthStateStore) Load(ctx context.Context, state string) (*OAuthState, error) {
	key := oauthStateKeyPrefix + state

	result, err := getAndDeleteScript.Run(ctx, r.client, []string{key}).Result()
	if err == redis.Nil || result == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var data OAuthState
	if err := json.Unmarshal([]byte(result.(string)), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
