package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrTaskExists   = errors.New("task already exists")
)

const (
	taskKeyPrefix  = "nemoclaw:task:"
	stateKeyPrefix = "nemoclaw:state:"
	allTasksKey    = "nemoclaw:tasks"
)

type TaskStore struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *TaskStore {
	return &TaskStore{rdb: rdb}
}

func taskKey(id string) string          { return taskKeyPrefix + id }
func stateKey(s model.TaskState) string { return stateKeyPrefix + string(s) }

func (s *TaskStore) Create(ctx context.Context, task *model.Task) error {
	key := taskKey(task.LinearIssueID)

	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("check task exists: %w", err)
	}
	if exists > 0 {
		return ErrTaskExists
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, key, data, 0)
	pipe.SAdd(ctx, allTasksKey, task.LinearIssueID)
	pipe.SAdd(ctx, stateKey(task.State), task.LinearIssueID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (s *TaskStore) Get(ctx context.Context, linearIssueID string) (*model.Task, error) {
	data, err := s.rdb.Get(ctx, taskKey(linearIssueID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	var task model.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}

func (s *TaskStore) UpdateState(ctx context.Context, linearIssueID string, newState model.TaskState) error {
	task, err := s.Get(ctx, linearIssueID)
	if err != nil {
		return err
	}

	oldState := task.State
	if err := task.TransitionTo(newState); err != nil {
		return err
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, taskKey(linearIssueID), data, 0)
	pipe.SRem(ctx, stateKey(oldState), linearIssueID)
	pipe.SAdd(ctx, stateKey(newState), linearIssueID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("update task state: %w", err)
	}
	return nil
}

func (s *TaskStore) ListByState(ctx context.Context, state model.TaskState) ([]*model.Task, error) {
	ids, err := s.rdb.SMembers(ctx, stateKey(state)).Result()
	if err != nil {
		return nil, fmt.Errorf("list by state: %w", err)
	}

	tasks := make([]*model.Task, 0, len(ids))
	for _, id := range ids {
		task, err := s.Get(ctx, id)
		if err != nil {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *TaskStore) Delete(ctx context.Context, linearIssueID string) error {
	task, err := s.Get(ctx, linearIssueID)
	if err != nil {
		return err
	}

	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, taskKey(linearIssueID))
	pipe.SRem(ctx, allTasksKey, linearIssueID)
	pipe.SRem(ctx, stateKey(task.State), linearIssueID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

func (s *TaskStore) QueueDepth(ctx context.Context) (int, error) {
	count, err := s.rdb.SCard(ctx, stateKey(model.StateQueued)).Result()
	if err != nil {
		return 0, fmt.Errorf("queue depth: %w", err)
	}
	return int(count), nil
}

func (s *TaskStore) Exists(ctx context.Context, linearIssueID string) (bool, error) {
	n, err := s.rdb.Exists(ctx, taskKey(linearIssueID)).Result()
	if err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}
	return n > 0, nil
}
