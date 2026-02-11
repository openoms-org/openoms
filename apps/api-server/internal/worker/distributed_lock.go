package worker

import (
	"context"
	"time"
)

// DistributedLock provides distributed locking capability.
// When Redis is configured, it uses Redis SETNX for locking.
// When Redis is not available, it falls back to a no-op (always acquires).
type DistributedLock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (acquired bool, err error)
	Unlock(ctx context.Context, key string) error
}

// NoOpLock always acquires the lock (for single-instance deployments).
type NoOpLock struct{}

func (l *NoOpLock) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (l *NoOpLock) Unlock(ctx context.Context, key string) error {
	return nil
}
