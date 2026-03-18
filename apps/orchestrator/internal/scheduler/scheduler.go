package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

type LinearPoller interface {
	FetchTodoIssues(ctx context.Context) ([]linear.Issue, error)
}

type Scheduler struct {
	taskStore    *store.TaskStore
	linearClient LinearPoller
	pollInterval time.Duration
}

type Option func(*Scheduler)

func WithPollInterval(d time.Duration) Option {
	return func(s *Scheduler) { s.pollInterval = d }
}

func New(taskStore *store.TaskStore, linearClient LinearPoller, opts ...Option) *Scheduler {
	s := &Scheduler{
		taskStore:    taskStore,
		linearClient: linearClient,
		pollInterval: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Scheduler) Run(ctx context.Context) {
	slog.Info("scheduler started", "poll_interval", s.pollInterval)
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	if err := s.Poll(ctx); err != nil {
		slog.Error("initial poll failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopping")
			return
		case <-ticker.C:
			if err := s.Poll(ctx); err != nil {
				slog.Error("poll failed", "error", err)
			}
		}
	}
}

func (s *Scheduler) Poll(ctx context.Context) error {
	issues, err := s.linearClient.FetchTodoIssues(ctx)
	if err != nil {
		return err
	}

	enqueued := 0
	for _, issue := range issues {
		if hasLabel(issue, "manual") {
			continue
		}

		exists, err := s.taskStore.Exists(ctx, issue.Identifier)
		if err != nil {
			slog.Error("check exists during poll", "error", err, "identifier", issue.Identifier)
			continue
		}
		if exists {
			continue
		}

		task := model.NewTask(issue.Identifier, issue.Title, issue.Description)
		task.Priority = issue.Priority
		if err := s.taskStore.Create(ctx, task); err != nil {
			if errors.Is(err, store.ErrTaskExists) {
				continue
			}
			slog.Error("create task during poll", "error", err, "identifier", issue.Identifier)
			continue
		}
		enqueued++
	}

	slog.Info("poll complete", "total_issues", len(issues), "enqueued", enqueued)
	return nil
}

func hasLabel(issue linear.Issue, name string) bool {
	for _, l := range issue.Labels.Nodes {
		if l.Name == name {
			return true
		}
	}
	return false
}
