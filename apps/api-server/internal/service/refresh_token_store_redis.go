package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rtFamilyPrefix = "rt_family:"
	rtTokenPrefix  = "rt_token:"
)

const consumeRefreshEntryScript = `
local val = redis.call("GET", KEYS[1])
if not val then
	return {0, ""}
end

local entry = cjson.decode(val)
if entry["used"] == true then
	return {2, val}
end

entry["used"] = true
local updated = cjson.encode(entry)
local ttl = redis.call("PTTL", KEYS[1])
if ttl > 0 then
	redis.call("SET", KEYS[1], updated, "PX", ttl)
else
	redis.call("SET", KEYS[1], updated)
end
return {1, val}
`

const updateRefreshFamilyScript = `
if redis.call("EXISTS", KEYS[1]) == 0 then
	return 0
end

redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
return 1
`

// RedisRefreshTokenStore implements RefreshTokenStore using Redis.
// Safe for multi-pod deployments.
type RedisRefreshTokenStore struct {
	client *redis.Client
}

// NewRedisRefreshTokenStore creates a new Redis-backed refresh token store.
func NewRedisRefreshTokenStore(client *redis.Client) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{client: client}
}

// StoreFamily persists a token family in Redis with the given TTL.
func (r *RedisRefreshTokenStore) StoreFamily(ctx context.Context, family *RefreshTokenFamily, ttl time.Duration) error {
	val, err := json.Marshal(family)
	if err != nil {
		return fmt.Errorf("marshal family: %w", err)
	}
	return r.client.Set(ctx, rtFamilyPrefix+family.FamilyID, val, ttl).Err()
}

// GetFamily retrieves a token family from Redis. Returns nil if not found.
func (r *RedisRefreshTokenStore) GetFamily(ctx context.Context, familyID string) (*RefreshTokenFamily, error) {
	val, err := r.client.Get(ctx, rtFamilyPrefix+familyID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var family RefreshTokenFamily
	if err := json.Unmarshal([]byte(val), &family); err != nil {
		return nil, fmt.Errorf("unmarshal family: %w", err)
	}
	return &family, nil
}

// UpdateFamily overwrites an existing token family in Redis with a new TTL.
func (r *RedisRefreshTokenStore) UpdateFamily(ctx context.Context, family *RefreshTokenFamily, ttl time.Duration) error {
	val, err := json.Marshal(family)
	if err != nil {
		return fmt.Errorf("marshal family: %w", err)
	}
	ttlMillis := ttl.Milliseconds()
	if ttlMillis <= 0 {
		return fmt.Errorf("refresh token family ttl must be positive")
	}
	res, err := r.client.Eval(ctx, updateRefreshFamilyScript, []string{rtFamilyPrefix + family.FamilyID}, val, ttlMillis).Result()
	if err != nil {
		return err
	}
	updated, err := redisLuaInt(res)
	if err != nil {
		return fmt.Errorf("parse update family response: %w", err)
	}
	if updated == 0 {
		return errRefreshTokenFamilyNotFound
	}
	return nil
}

// DeleteFamily removes a token family from Redis.
func (r *RedisRefreshTokenStore) DeleteFamily(ctx context.Context, familyID string) error {
	return r.client.Del(ctx, rtFamilyPrefix+familyID).Err()
}

// StoreToken persists a refresh token entry in Redis with the given TTL.
func (r *RedisRefreshTokenStore) StoreToken(ctx context.Context, tokenHash string, entry *RefreshTokenEntry, ttl time.Duration) error {
	val, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal token entry: %w", err)
	}
	return r.client.Set(ctx, rtTokenPrefix+tokenHash, val, ttl).Err()
}

// GetToken retrieves a refresh token entry from Redis. Returns nil if not found.
func (r *RedisRefreshTokenStore) GetToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error) {
	val, err := r.client.Get(ctx, rtTokenPrefix+tokenHash).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entry RefreshTokenEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, fmt.Errorf("unmarshal token entry: %w", err)
	}
	return &entry, nil
}

// MarkTokenUsed sets the Used flag to true for the given token hash in Redis.
func (r *RedisRefreshTokenStore) MarkTokenUsed(ctx context.Context, tokenHash string) error {
	_, _, err := r.ConsumeToken(ctx, tokenHash)
	return err
}

// ConsumeToken atomically marks an unused token as used and returns its entry.
func (r *RedisRefreshTokenStore) ConsumeToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, bool, error) {
	res, err := r.client.Eval(ctx, consumeRefreshEntryScript, []string{rtTokenPrefix + tokenHash}).Result()
	if err != nil {
		return nil, false, err
	}

	values, ok := res.([]any)
	if !ok || len(values) != 2 {
		return nil, false, fmt.Errorf("unexpected consume token response: %T", res)
	}

	status, err := redisLuaInt(values[0])
	if err != nil {
		return nil, false, fmt.Errorf("parse consume token status: %w", err)
	}
	if status == 0 {
		return nil, false, nil
	}

	val, err := redisLuaString(values[1])
	if err != nil {
		return nil, false, fmt.Errorf("parse consume token entry: %w", err)
	}
	var entry RefreshTokenEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, false, fmt.Errorf("unmarshal token entry: %w", err)
	}

	switch status {
	case 1:
		return &entry, true, nil
	case 2:
		return &entry, false, nil
	default:
		return nil, false, fmt.Errorf("unexpected consume token status: %d", status)
	}
}

// DeleteToken removes a refresh token entry from Redis.
func (r *RedisRefreshTokenStore) DeleteToken(ctx context.Context, tokenHash string) error {
	return r.client.Del(ctx, rtTokenPrefix+tokenHash).Err()
}

// IsPersistent returns true — Redis survives server restarts.
func (r *RedisRefreshTokenStore) IsPersistent() bool { return true }

func redisLuaInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case string:
		return strconv.Atoi(n)
	case []byte:
		return strconv.Atoi(string(n))
	default:
		return 0, fmt.Errorf("unexpected integer type %T", v)
	}
}

func redisLuaString(v any) (string, error) {
	switch s := v.(type) {
	case string:
		return s, nil
	case []byte:
		return string(s), nil
	default:
		return "", fmt.Errorf("unexpected string type %T", v)
	}
}
