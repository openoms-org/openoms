package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

func setupRedis(t *testing.T) *store.TaskStore {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/1"
	}
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)
	rdb := redis.NewClient(opts)

	ctx := context.Background()
	require.NoError(t, rdb.Ping(ctx).Err(), "Redis must be running for store tests")

	rdb.FlushDB(ctx)
	t.Cleanup(func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	})

	return store.New(rdb)
}

func TestTaskStore_CreateAndGet(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	task := model.NewTask("OPE-142", "Bulk export", "Add CSV export")
	err := s.Create(ctx, task)
	require.NoError(t, err)

	got, err := s.Get(ctx, "OPE-142")
	require.NoError(t, err)
	assert.Equal(t, "OPE-142", got.LinearIssueID)
	assert.Equal(t, model.StateQueued, got.State)
	assert.Equal(t, "Bulk export", got.Title)
}

func TestTaskStore_Create_Duplicate(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	task := model.NewTask("OPE-142", "Bulk export", "desc")
	require.NoError(t, s.Create(ctx, task))

	err := s.Create(ctx, task)
	assert.ErrorIs(t, err, store.ErrTaskExists)
}

func TestTaskStore_UpdateState(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	task := model.NewTask("OPE-142", "Bulk export", "desc")
	require.NoError(t, s.Create(ctx, task))

	err := s.UpdateState(ctx, "OPE-142", model.StateDecomposing)
	require.NoError(t, err)

	got, err := s.Get(ctx, "OPE-142")
	require.NoError(t, err)
	assert.Equal(t, model.StateDecomposing, got.State)
}

func TestTaskStore_UpdateState_InvalidTransition(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	task := model.NewTask("OPE-142", "Bulk export", "desc")
	require.NoError(t, s.Create(ctx, task))

	err := s.UpdateState(ctx, "OPE-142", model.StateDone)
	assert.Error(t, err)
}

func TestTaskStore_ListByState(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, model.NewTask("OPE-1", "Task 1", "desc")))
	require.NoError(t, s.Create(ctx, model.NewTask("OPE-2", "Task 2", "desc")))
	require.NoError(t, s.Create(ctx, model.NewTask("OPE-3", "Task 3", "desc")))

	require.NoError(t, s.UpdateState(ctx, "OPE-2", model.StateDecomposing))

	queued, err := s.ListByState(ctx, model.StateQueued)
	require.NoError(t, err)
	assert.Len(t, queued, 2)

	decomposing, err := s.ListByState(ctx, model.StateDecomposing)
	require.NoError(t, err)
	assert.Len(t, decomposing, 1)
	assert.Equal(t, "OPE-2", decomposing[0].LinearIssueID)
}

func TestTaskStore_Delete(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, model.NewTask("OPE-142", "Task", "desc")))
	require.NoError(t, s.Delete(ctx, "OPE-142"))

	_, err := s.Get(ctx, "OPE-142")
	assert.ErrorIs(t, err, store.ErrTaskNotFound)
}

func TestTaskStore_QueueDepth(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, model.NewTask("OPE-1", "T1", "d")))
	require.NoError(t, s.Create(ctx, model.NewTask("OPE-2", "T2", "d")))

	depth, err := s.QueueDepth(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, depth)
}
