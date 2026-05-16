package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DistributedLock provides Redis-based distributed locking for workers.
// When multiple API pods run simultaneously (HPA), this ensures only one pod
// executes each worker at a time, preventing duplicate order polling,
// duplicate delayed actions, and duplicate notifications.
//
// Lock ownership is verified on release via a Lua script that checks the
// stored UUID matches before deleting — prevents releasing another pod's lock
// if the original lock expired and was re-acquired.
type DistributedLock struct {
	client *redis.Client
	prefix string
}

// releaseScript atomically checks ownership before deleting the lock.
// Returns 1 if the lock was released, 0 if the value didn't match (someone else owns it).
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

var extendScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

// NewDistributedLock creates a new distributed lock manager.
// If client is nil (Redis unavailable), locks always succeed (single-pod mode).
func NewDistributedLock(client *redis.Client, prefix string) *DistributedLock {
	return &DistributedLock{client: client, prefix: prefix}
}

// Acquire tries to acquire a lock for the given worker name.
// Returns the lock token (UUID) if acquired, or empty string if another instance holds it.
// The lock auto-expires after ttl to prevent deadlocks if a pod crashes.
func (d *DistributedLock) Acquire(ctx context.Context, workerName string, ttl time.Duration) (string, error) {
	if d == nil || d.client == nil {
		return "no-redis", nil // No Redis = single-pod mode, always proceed
	}
	if ttl <= 0 {
		return "", fmt.Errorf("distributed lock: TTL must be positive, got %v", ttl)
	}
	key := fmt.Sprintf("%s:worker-lock:%s", d.prefix, workerName)
	token := uuid.New().String()
	ok, err := d.client.SetArgs(ctx, key, token, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
	}).Result()
	if err == redis.Nil {
		return "", nil // Key already exists — another pod holds the lock
	}
	if err != nil {
		return "", err
	}
	if ok != "OK" {
		return "", nil
	}
	return token, nil
}

// Extend renews a lock only if the token still owns it.
func (d *DistributedLock) Extend(ctx context.Context, workerName string, token string, ttl time.Duration) (bool, error) {
	if d == nil || d.client == nil || token == "no-redis" {
		return true, nil
	}
	if token == "" {
		return false, nil
	}
	if ttl <= 0 {
		return false, fmt.Errorf("distributed lock: TTL must be positive, got %v", ttl)
	}
	key := fmt.Sprintf("%s:worker-lock:%s", d.prefix, workerName)
	result, err := extendScript.Run(ctx, d.client, []string{key}, token, ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// Release releases the lock only if the token matches (ownership verification).
// Uses a Lua script for atomic check-and-delete. Uses context.Background()
// because the worker context may already be cancelled during shutdown.
func (d *DistributedLock) Release(workerName string, token string) {
	if d == nil || d.client == nil || token == "" || token == "no-redis" {
		return
	}
	key := fmt.Sprintf("%s:worker-lock:%s", d.prefix, workerName)
	// Use Background context — the worker context may be cancelled during shutdown.
	_ = releaseScript.Run(context.Background(), d.client, []string{key}, token).Err()
}
