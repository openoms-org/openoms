package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedLock_NilStruct(t *testing.T) {
	var dl *DistributedLock

	token, err := dl.Acquire(context.Background(), "test-worker", time.Second)
	require.NoError(t, err)
	assert.Equal(t, "no-redis", token, "nil lock should return no-redis token")

	// Release should not panic on nil receiver.
	dl.Release("test-worker", token)
}

func TestDistributedLock_NilClient(t *testing.T) {
	dl := NewDistributedLock(nil, "openoms")

	token, err := dl.Acquire(context.Background(), "test-worker", time.Second)
	require.NoError(t, err)
	assert.Equal(t, "no-redis", token, "nil Redis client should return no-redis token")

	// Release should not panic with nil client.
	dl.Release("test-worker", token)
}

func TestDistributedLock_InvalidTTL(t *testing.T) {
	dl := NewDistributedLock(nil, "openoms")
	// nil client returns no-redis regardless of TTL, so this tests the real path
	// only with an actual Redis client. For nil client, TTL check is skipped.
	token, err := dl.Acquire(context.Background(), "test-worker", -1*time.Second)
	// With nil client, this still succeeds (single-pod mode)
	require.NoError(t, err)
	assert.Equal(t, "no-redis", token)
}

func TestNewDistributedLock(t *testing.T) {
	dl := NewDistributedLock(nil, "test-prefix")

	require.NotNil(t, dl)
	assert.Nil(t, dl.client)
	assert.Equal(t, "test-prefix", dl.prefix)
}
