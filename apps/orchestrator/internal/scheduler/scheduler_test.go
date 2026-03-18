package scheduler_test

import (
	"context"
	"os"
	"testing"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/scheduler"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

func setupSchedulerTest(t *testing.T) (*store.TaskStore, *redisclient.Client) {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/1"
	}
	opts, err := redisclient.ParseURL(url)
	require.NoError(t, err)
	rdb := redisclient.NewClient(opts)
	require.NoError(t, rdb.Ping(context.Background()).Err())
	rdb.FlushDB(context.Background())
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		rdb.Close()
	})
	return store.New(rdb), rdb
}

type mockLinearClient struct {
	issues []linear.Issue
	err    error
}

func (m *mockLinearClient) FetchTodoIssues(_ context.Context) ([]linear.Issue, error) {
	return m.issues, m.err
}

func TestScheduler_PollEnqueuesNewTasks(t *testing.T) {
	taskStore, _ := setupSchedulerTest(t)

	mock := &mockLinearClient{
		issues: []linear.Issue{
			{Identifier: "OPE-200", Title: "New task", Description: "desc", Priority: 2},
			{Identifier: "OPE-201", Title: "Another task", Description: "desc2", Priority: 1},
		},
	}

	s := scheduler.New(taskStore, mock)
	err := s.Poll(context.Background())
	require.NoError(t, err)

	queued, err := taskStore.ListByState(context.Background(), model.StateQueued)
	require.NoError(t, err)
	assert.Len(t, queued, 2)
}

func TestScheduler_PollSkipsDuplicates(t *testing.T) {
	taskStore, _ := setupSchedulerTest(t)

	require.NoError(t, taskStore.Create(context.Background(),
		model.NewTask("OPE-200", "Existing", "already in queue")))

	mock := &mockLinearClient{
		issues: []linear.Issue{
			{Identifier: "OPE-200", Title: "Existing", Priority: 2},
			{Identifier: "OPE-201", Title: "New one", Priority: 1},
		},
	}

	s := scheduler.New(taskStore, mock)
	require.NoError(t, s.Poll(context.Background()))

	queued, err := taskStore.ListByState(context.Background(), model.StateQueued)
	require.NoError(t, err)
	assert.Len(t, queued, 2) // 1 existing + 1 new
}

func TestScheduler_PollSkipsManualLabeled(t *testing.T) {
	taskStore, _ := setupSchedulerTest(t)

	issue := linear.Issue{
		Identifier: "OPE-300",
		Title:      "Manual task",
		Priority:   2,
	}
	issue.Labels.Nodes = []struct {
		Name string `json:"name"`
	}{{Name: "manual"}}

	mock := &mockLinearClient{
		issues: []linear.Issue{issue},
	}

	s := scheduler.New(taskStore, mock)
	require.NoError(t, s.Poll(context.Background()))

	queued, err := taskStore.ListByState(context.Background(), model.StateQueued)
	require.NoError(t, err)
	assert.Len(t, queued, 0)
}

func TestScheduler_RunStopsOnCancel(t *testing.T) {
	taskStore, _ := setupSchedulerTest(t)
	mock := &mockLinearClient{}

	s := scheduler.New(taskStore, mock, scheduler.WithPollInterval(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after cancel")
	}
}
