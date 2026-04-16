package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	rtFamilyPrefix = "rt_family:"
	rtTokenPrefix  = "rt_token:"
)

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
	return r.StoreFamily(ctx, family, ttl)
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
	val, err := r.client.Get(ctx, rtTokenPrefix+tokenHash).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	var entry RefreshTokenEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return fmt.Errorf("unmarshal token entry: %w", err)
	}
	entry.Used = true

	updated, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal token entry: %w", err)
	}

	// Preserve remaining TTL
	ttl, err := r.client.TTL(ctx, rtTokenPrefix+tokenHash).Result()
	if err != nil || ttl <= 0 {
		ttl = 30 * 24 * time.Hour // fallback to max refresh TTL
	}

	return r.client.Set(ctx, rtTokenPrefix+tokenHash, updated, ttl).Err()
}

// DeleteToken removes a refresh token entry from Redis.
func (r *RedisRefreshTokenStore) DeleteToken(ctx context.Context, tokenHash string) error {
	return r.client.Del(ctx, rtTokenPrefix+tokenHash).Err()
}

// IsPersistent returns true — Redis survives server restarts.
func (r *RedisRefreshTokenStore) IsPersistent() bool { return true }
