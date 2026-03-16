package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedLock provides Redis SETNX-based distributed locking for workers.
// When multiple API pods run simultaneously (HPA), this ensures only one pod
// executes each worker at a time, preventing duplicate order polling,
// duplicate delayed actions, and duplicate notifications.
type DistributedLock struct {
	client *redis.Client
	prefix string
}

// NewDistributedLock creates a new distributed lock manager.
// If client is nil (Redis unavailable), locks always succeed (single-pod mode).
func NewDistributedLock(client *redis.Client, prefix string) *DistributedLock {
	return &DistributedLock{client: client, prefix: prefix}
}

// Acquire tries to acquire a lock for the given worker name.
// Returns true if the lock was acquired, false if another instance holds it.
// The lock auto-expires after ttl to prevent deadlocks if a pod crashes.
func (d *DistributedLock) Acquire(ctx context.Context, workerName string, ttl time.Duration) (bool, error) {
	if d == nil || d.client == nil {
		return true, nil // No Redis = single-pod mode, always proceed
	}
	key := fmt.Sprintf("%s:worker-lock:%s", d.prefix, workerName)
	return d.client.SetNX(ctx, key, "locked", ttl).Result()
}

// Release releases the lock for the given worker name.
func (d *DistributedLock) Release(ctx context.Context, workerName string) {
	if d == nil || d.client == nil {
		return
	}
	key := fmt.Sprintf("%s:worker-lock:%s", d.prefix, workerName)
	_ = d.client.Del(ctx, key).Err()
}
