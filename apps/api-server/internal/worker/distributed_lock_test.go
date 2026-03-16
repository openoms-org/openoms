package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Nil client (single-pod / no Redis fallback)
// ---------------------------------------------------------------------------

func TestDistributedLock_NilStruct(t *testing.T) {
	var dl *DistributedLock

	acquired, err := dl.Acquire(context.Background(), "test-worker", time.Second)
	require.NoError(t, err)
	assert.True(t, acquired, "nil lock should always succeed")

	// Release should not panic on nil receiver.
	dl.Release(context.Background(), "test-worker")
}

func TestDistributedLock_NilClient(t *testing.T) {
	dl := NewDistributedLock(nil, "openoms")

	acquired, err := dl.Acquire(context.Background(), "test-worker", time.Second)
	require.NoError(t, err)
	assert.True(t, acquired, "nil Redis client should always succeed")

	// Release should not panic with nil client.
	dl.Release(context.Background(), "test-worker")
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewDistributedLock(t *testing.T) {
	dl := NewDistributedLock(nil, "test-prefix")

	require.NotNil(t, dl)
	assert.Nil(t, dl.client)
	assert.Equal(t, "test-prefix", dl.prefix)
}
