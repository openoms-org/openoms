# NemoClaw Phase 1: Orchestrator Core — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working orchestrator service that receives Linear tickets via webhooks, queues them in Redis, and tracks task state — the foundation for all subsequent phases.

**Architecture:** New Go service at `apps/orchestrator/` following existing api-server patterns (chi router, caarlos0/env config, slog logging, Redis via go-redis/v9). Receives Linear webhooks, deduplicates tasks in Redis, runs a polling fallback, exposes health endpoint.

**Tech Stack:** Go 1.25, chi/v5, go-redis/v9, caarlos0/env/v11, slog, testify

**Spec:** `docs/superpowers/specs/2026-03-17-nemoclaw-orchestrator-design.md` (sections 1-5, 9.1, 9.14, 9.15, 9.17, 19, 21)

**Phase scope:** Linear intake → Redis task queue → scheduler event loop → health endpoint. NO agent dispatch, NO GitHub integration, NO OpenRouter calls — those are Phase 2+.

---

## File Structure

```
apps/orchestrator/
├── cmd/orchestrator/
│   └── main.go                    # Entry point (config → Redis → router → server)
├── internal/
│   ├── config/
│   │   └── config.go              # Env-based config (LINEAR_*, REDIS_URL, PORT)
│   │   └── config_test.go
│   ├── model/
│   │   └── task.go                # Task model + state machine constants
│   │   └── task_test.go
│   ├── store/
│   │   └── redis.go               # Redis-backed task store (CRUD + queue ops)
│   │   └── redis_test.go
│   ├── linear/
│   │   └── webhook.go             # Linear webhook handler (signature verification)
│   │   └── webhook_test.go
│   │   └── client.go              # Linear GraphQL API client (poll + update)
│   │   └── client_test.go
│   ├── scheduler/
│   │   └── scheduler.go           # Event loop: poll Linear, process queue
│   │   └── scheduler_test.go
│   └── handler/
│       └── health.go              # GET /health
│       └── health_test.go
├── go.mod
└── go.sum
```

**Total: 7 source files + 7 test files + main.go + go.mod**

---

## Chunk 1: Project Scaffolding + Config + Health

### Task 1: Project scaffolding

**Files:**
- Create: `apps/orchestrator/go.mod`
- Create: `apps/orchestrator/cmd/orchestrator/main.go`
- Modify: `go.work` (add `./apps/orchestrator`)

- [ ] **Step 1: Create go.mod**

```bash
mkdir -p apps/orchestrator/cmd/orchestrator
mkdir -p apps/orchestrator/internal/{config,model,store,linear,scheduler,handler}
cd apps/orchestrator && go mod init github.com/openoms-org/openoms/apps/orchestrator
```

- [ ] **Step 2: Add dependencies**

```bash
cd apps/orchestrator
go get github.com/go-chi/chi/v5@latest
go get github.com/redis/go-redis/v9@latest
go get github.com/caarlos0/env/v11@latest
go get github.com/stretchr/testify@latest
```

- [ ] **Step 3: Add to go.work**

Add `./apps/orchestrator` to the `use` block in `/Users/rafs/praca/OpenOMS/go.work`:

```go
use (
	./apps/api-server
	./apps/orchestrator
	./packages/allegro-go-sdk
	// ... rest unchanged
)
```

- [ ] **Step 4: Create minimal main.go**

File: `apps/orchestrator/cmd/orchestrator/main.go`

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/config"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/handler"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Logger
	logLevel := slog.LevelInfo
	if cfg.Env == "development" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	slog.Info("starting NemoClaw orchestrator", "port", cfg.Port, "env", cfg.Env)

	// Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(redisOpts)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	slog.Info("connected to Redis")

	taskStore := store.New(rdb)

	// Router
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	healthHandler := handler.NewHealthHandler(rdb, taskStore)
	r.Get("/health", healthHandler.Health)

	// Server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	slog.Info("server stopped gracefully")
	return nil
}
```

- [ ] **Step 5: Verify it compiles**

```bash
cd /Users/rafs/praca/OpenOMS/apps/orchestrator
go build ./cmd/orchestrator/
```

Expected: compilation errors about missing packages (config, handler, store) — that's correct, we build them next.

- [ ] **Step 6: Commit scaffolding**

```bash
git add apps/orchestrator/go.mod apps/orchestrator/go.sum apps/orchestrator/cmd/ go.work
git commit -m "feat(orchestrator): scaffold orchestrator service

Add new Go service at apps/orchestrator with chi router,
Redis connection, and graceful shutdown. Phase 1 of NemoClaw."
```

---

### Task 2: Config

**Files:**
- Create: `apps/orchestrator/internal/config/config.go`
- Create: `apps/orchestrator/internal/config/config_test.go`

- [ ] **Step 1: Write config test**

File: `apps/orchestrator/internal/config/config_test.go`

```go
package config_test

import (
	"testing"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Set only required vars
	t.Setenv("LINEAR_API_KEY", "lin_api_test")
	t.Setenv("LINEAR_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "lin_api_test", cfg.LinearAPIKey)
	assert.Equal(t, "whsec_test", cfg.LinearWebhookSecret)
	assert.Equal(t, "OPE", cfg.LinearTeamID)
	assert.Equal(t, 5, cfg.MaxCIRetries)
	assert.Equal(t, 300, cfg.PollIntervalSeconds)
	assert.Equal(t, float64(280), cfg.BudgetMonthlyLimit)
}

func TestLoad_MissingRequired(t *testing.T) {
	// Clear all env
	t.Setenv("LINEAR_API_KEY", "")
	t.Setenv("LINEAR_WEBHOOK_SECRET", "")

	_, err := config.Load()
	assert.Error(t, err)
}

func TestConfig_Validate(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "lin_api_test")
	t.Setenv("LINEAR_WEBHOOK_SECRET", "whsec_test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("MAX_CI_RETRIES", "0")

	cfg, err := config.Load()
	require.NoError(t, err)

	err = cfg.Validate()
	assert.Error(t, err, "MaxCIRetries=0 should fail validation")
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd /Users/rafs/praca/OpenOMS/apps/orchestrator
go test ./internal/config/ -v
```

Expected: FAIL — package config does not exist yet.

- [ ] **Step 3: Implement config**

File: `apps/orchestrator/internal/config/config.go`

```go
// Package config loads runtime configuration for the orchestrator from environment variables.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port string `env:"PORT" envDefault:"9090"`
	Env  string `env:"ENV" envDefault:"development"`

	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379"`

	LinearAPIKey        string `env:"LINEAR_API_KEY,required"`
	LinearWebhookSecret string `env:"LINEAR_WEBHOOK_SECRET,required"`
	LinearTeamID        string `env:"LINEAR_TEAM_ID" envDefault:"OPE"`

	OpenRouterAPIKey string `env:"OPENROUTER_API_KEY"`
	GitHubToken      string `env:"GITHUB_TOKEN"`
	GitHubRepo       string `env:"GITHUB_REPO" envDefault:"openoms-org/openoms"`

	TelegramBotToken string `env:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID   string `env:"TELEGRAM_CHAT_ID"`

	MaxCIRetries        int     `env:"MAX_CI_RETRIES" envDefault:"5"`
	PollIntervalSeconds int     `env:"POLL_INTERVAL_SECONDS" envDefault:"300"`
	BudgetMonthlyLimit  float64 `env:"BUDGET_MONTHLY_LIMIT" envDefault:"280"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.MaxCIRetries < 1 || c.MaxCIRetries > 10 {
		return fmt.Errorf("MAX_CI_RETRIES must be between 1 and 10, got %d", c.MaxCIRetries)
	}
	if c.PollIntervalSeconds < 30 {
		return fmt.Errorf("POLL_INTERVAL_SECONDS must be >= 30, got %d", c.PollIntervalSeconds)
	}
	if c.BudgetMonthlyLimit <= 0 {
		return fmt.Errorf("BUDGET_MONTHLY_LIMIT must be positive, got %f", c.BudgetMonthlyLimit)
	}
	return nil
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
cd /Users/rafs/praca/OpenOMS/apps/orchestrator
go test ./internal/config/ -v
```

Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/orchestrator/internal/config/
git commit -m "feat(orchestrator): add env-based config with validation"
```

---

### Task 3: Task model + state machine

**Files:**
- Create: `apps/orchestrator/internal/model/task.go`
- Create: `apps/orchestrator/internal/model/task_test.go`

- [ ] **Step 1: Write model test**

File: `apps/orchestrator/internal/model/task_test.go`

```go
package model_test

import (
	"testing"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestTaskState_ValidTransitions(t *testing.T) {
	tests := []struct {
		from model.TaskState
		to   model.TaskState
		ok   bool
	}{
		{model.StateQueued, model.StateDecomposing, true},
		{model.StateDecomposing, model.StateAssigned, true},
		{model.StateDecomposing, model.StateSplit, true},
		{model.StateDecomposing, model.StateBlocked, true},
		{model.StateAssigned, model.StateDeveloping, true},
		{model.StateAssigned, model.StateQueued, true}, // dispatch fail → re-queue
		{model.StateDeveloping, model.StatePRCreated, true},
		{model.StateDeveloping, model.StateLocalFix, true},
		{model.StateLocalFix, model.StateDeveloping, true},   // fixed
		{model.StateLocalFix, model.StateEscalated, true},    // can't fix
		{model.StateEscalated, model.StateAssigned, true},    // PO re-assigns
		{model.StateEscalated, model.StateDeveloping, true},  // CEO guides
		{model.StateEscalated, model.StateBlocked, true},     // too complex
		{model.StatePRCreated, model.StateCIRunning, true},
		{model.StateCIRunning, model.StateSecurityReview, true},
		{model.StateCIRunning, model.StateCIFixing, true},
		{model.StateCIFixing, model.StateCIRunning, true},   // fixed → CI re-run
		{model.StateCIFixing, model.StateBlocked, true},     // 5 retries
		{model.StateSecurityReview, model.StateMerging, true},
		{model.StateSecurityReview, model.StateSecFixing, true},
		{model.StateSecFixing, model.StateCIRunning, true},  // CI must re-run after sec fix
		{model.StateMerging, model.StateDone, true},
		{model.StateBlocked, model.StateQueued, true},       // human re-opens
		{model.StateBlocked, model.StateCancelled, true},    // human cancels

		// Invalid transitions
		{model.StateQueued, model.StateDone, false},
		{model.StateDone, model.StateQueued, false},
		{model.StateCIRunning, model.StateDeveloping, false},
		{model.StateDecomposing, model.StateDone, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + " → " + string(tt.to)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.ok, model.IsValidTransition(tt.from, tt.to), name)
		})
	}
}

func TestNewTask(t *testing.T) {
	task := model.NewTask("OPE-142", "Add bulk export", "Implement CSV export for orders")
	assert.Equal(t, "OPE-142", task.LinearIssueID)
	assert.Equal(t, model.StateQueued, task.State)
	assert.False(t, task.CreatedAt.IsZero())
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/model/ -v
```

- [ ] **Step 3: Implement model**

File: `apps/orchestrator/internal/model/task.go`

```go
// Package model defines domain models for the orchestrator.
package model

import "time"

type TaskState string

const (
	StateQueued          TaskState = "queued"
	StateDecomposing     TaskState = "decomposing"
	StateSplit           TaskState = "split"
	StateAssigned        TaskState = "assigned"
	StateDeveloping      TaskState = "developing"
	StateLocalFix        TaskState = "local_fix"
	StateEscalated       TaskState = "escalated"
	StatePRCreated       TaskState = "pr_created"
	StateCIRunning       TaskState = "ci_running"
	StateCIFixing        TaskState = "ci_fixing"
	StateSecurityReview  TaskState = "security_review"
	StateSecFixing       TaskState = "sec_fixing"
	StateMerging         TaskState = "merging"
	StateDone            TaskState = "done"
	StateBlocked         TaskState = "blocked"
	StateCancelled       TaskState = "cancelled"
)

type AgentRole string

const (
	RolePO             AgentRole = "po"
	RoleBackendDev     AgentRole = "backend-dev"
	RoleFrontendDev    AgentRole = "frontend-dev"
	RoleIntegrationDev AgentRole = "integration-dev"
	RoleDevOps         AgentRole = "devops"
	RoleQA             AgentRole = "qa"
	RoleSecurityReview AgentRole = "security-reviewer"
	RoleCEO            AgentRole = "ceo-architect"
)

type Task struct {
	LinearIssueID string    `json:"linear_issue_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	State         TaskState `json:"state"`
	AssignedAgent AgentRole `json:"assigned_agent,omitempty"`
	PRNumber      int       `json:"pr_number,omitempty"`
	BranchName    string    `json:"branch_name,omitempty"`
	CIRetryCount  int       `json:"ci_retry_count"`
	ParentTaskID  string    `json:"parent_task_id,omitempty"`
	Priority      int       `json:"priority"` // 0=urgent, 1=high, 2=medium, 3=low
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewTask(linearIssueID, title, description string) *Task {
	now := time.Now().UTC()
	return &Task{
		LinearIssueID: linearIssueID,
		Title:         title,
		Description:   description,
		State:         StateQueued,
		Priority:      2, // medium default
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// validTransitions defines all allowed state transitions per the spec (Section 19).
var validTransitions = map[TaskState][]TaskState{
	StateQueued:         {StateDecomposing},
	StateDecomposing:    {StateAssigned, StateSplit, StateBlocked},
	StateSplit:          {}, // parent stays in split; sub-tasks have own lifecycle
	StateAssigned:       {StateDeveloping, StateQueued}, // queued = dispatch fail re-queue
	StateDeveloping:     {StatePRCreated, StateLocalFix},
	StateLocalFix:       {StateDeveloping, StateEscalated},
	StateEscalated:      {StateAssigned, StateDeveloping, StateBlocked},
	StatePRCreated:      {StateCIRunning},
	StateCIRunning:      {StateSecurityReview, StateCIFixing},
	StateCIFixing:       {StateCIRunning, StateBlocked},
	StateSecurityReview: {StateMerging, StateSecFixing},
	StateSecFixing:      {StateCIRunning}, // CI must re-run after security fix
	StateMerging:        {StateDone},
	StateBlocked:        {StateQueued, StateCancelled},
	StateDone:           {},
	StateCancelled:      {},
}

func IsValidTransition(from, to TaskState) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

func (t *Task) TransitionTo(state TaskState) error {
	if !IsValidTransition(t.State, state) {
		return &InvalidTransitionError{From: t.State, To: state}
	}
	t.State = state
	t.UpdatedAt = time.Now().UTC()
	return nil
}

type InvalidTransitionError struct {
	From TaskState
	To   TaskState
}

func (e *InvalidTransitionError) Error() string {
	return "invalid state transition: " + string(e.From) + " → " + string(e.To)
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/model/ -v
```

Expected: all transition tests + NewTask test PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/orchestrator/internal/model/
git commit -m "feat(orchestrator): add task model with state machine transitions

Implements full lifecycle: queued → decomposing → assigned → developing →
pr_created → ci_running → security_review → merging → done.
All transitions match spec Section 19."
```

---

### Task 4: Redis task store

**Files:**
- Create: `apps/orchestrator/internal/store/redis.go`
- Create: `apps/orchestrator/internal/store/redis_test.go`

- [ ] **Step 1: Write store test**

File: `apps/orchestrator/internal/store/redis_test.go`

```go
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
		url = "redis://localhost:6379/1" // DB 1 for tests
	}
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)
	rdb := redis.NewClient(opts)

	ctx := context.Background()
	require.NoError(t, rdb.Ping(ctx).Err(), "Redis must be running for store tests")

	// Clean test DB
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
	assert.Error(t, err) // queued → done is invalid
}

func TestTaskStore_ListByState(t *testing.T) {
	s := setupRedis(t)
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, model.NewTask("OPE-1", "Task 1", "desc")))
	require.NoError(t, s.Create(ctx, model.NewTask("OPE-2", "Task 2", "desc")))
	require.NoError(t, s.Create(ctx, model.NewTask("OPE-3", "Task 3", "desc")))

	// Move OPE-2 to decomposing
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
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/store/ -v
```

Expected: FAIL — package doesn't exist.

- [ ] **Step 3: Implement store**

File: `apps/orchestrator/internal/store/redis.go`

```go
// Package store provides Redis-backed storage for orchestrator tasks.
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

func taskKey(id string) string  { return taskKeyPrefix + id }
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
			continue // task may have been deleted between SMembers and Get
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
```

- [ ] **Step 4: Run test — expect PASS (requires Redis running)**

```bash
go test ./internal/store/ -v
```

Expected: all 6 tests PASS. If Redis not running: `redis-server &` first.

- [ ] **Step 5: Commit**

```bash
git add apps/orchestrator/internal/store/
git commit -m "feat(orchestrator): add Redis-backed task store

Supports create (with deduplication), get, update state (with
transition validation), list by state, delete, queue depth.
Uses Redis sets for O(1) state-based lookups."
```

---

### Task 5: Health handler

**Files:**
- Create: `apps/orchestrator/internal/handler/health.go`
- Create: `apps/orchestrator/internal/handler/health_test.go`

- [ ] **Step 1: Write health test**

File: `apps/orchestrator/internal/handler/health_test.go`

```go
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/handler"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

func setupHealthTest(t *testing.T) (*handler.HealthHandler, *redis.Client) {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/1"
	}
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)
	rdb := redis.NewClient(opts)
	require.NoError(t, rdb.Ping(context.Background()).Err())
	rdb.FlushDB(context.Background())
	t.Cleanup(func() {
		rdb.FlushDB(context.Background())
		rdb.Close()
	})

	taskStore := store.New(rdb)
	return handler.NewHealthHandler(rdb, taskStore), rdb
}

func TestHealth_Healthy(t *testing.T) {
	h, _ := setupHealthTest(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Health(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "healthy", resp["status"])
}
```

- [ ] **Step 2: Implement health handler**

File: `apps/orchestrator/internal/handler/health.go`

```go
// Package handler provides HTTP handlers for the orchestrator.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

type HealthHandler struct {
	rdb       *redis.Client
	taskStore *store.TaskStore
}

func NewHealthHandler(rdb *redis.Client, taskStore *store.TaskStore) *HealthHandler {
	return &HealthHandler{rdb: rdb, taskStore: taskStore}
}

type HealthResponse struct {
	Status     string         `json:"status"`
	Checks     HealthChecks   `json:"checks"`
	QueueDepth int            `json:"queue_depth"`
}

type HealthChecks struct {
	Redis string `json:"redis"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := HealthResponse{Status: "healthy"}

	// Check Redis
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		resp.Status = "unhealthy"
		resp.Checks.Redis = "disconnected"
	} else {
		resp.Checks.Redis = "connected"
	}

	// Queue depth
	depth, err := h.taskStore.QueueDepth(ctx)
	if err != nil {
		resp.Status = "degraded"
	} else {
		resp.QueueDepth = depth
	}

	status := http.StatusOK
	if resp.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}
```

- [ ] **Step 3: Run test — expect PASS**

```bash
go test ./internal/handler/ -v
```

- [ ] **Step 4: Commit**

```bash
git add apps/orchestrator/internal/handler/
git commit -m "feat(orchestrator): add health endpoint with Redis + queue status"
```

---

## Chunk 2: Linear Webhook + Polling + Scheduler

### Task 6: Linear webhook handler

**Files:**
- Create: `apps/orchestrator/internal/linear/webhook.go`
- Create: `apps/orchestrator/internal/linear/webhook_test.go`

- [ ] **Step 1: Write webhook test**

File: `apps/orchestrator/internal/linear/webhook_test.go`

```go
package linear_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

const testWebhookSecret = "whsec_test_secret_123"

func setupWebhookTest(t *testing.T) (*linear.WebhookHandler, *store.TaskStore) {
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

	taskStore := store.New(rdb)
	wh := linear.NewWebhookHandler(testWebhookSecret, taskStore)
	return wh, taskStore
}

func signPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestWebhook_IssueCreate(t *testing.T) {
	wh, taskStore := setupWebhookTest(t)

	payload := map[string]interface{}{
		"action": "create",
		"type":   "Issue",
		"data": map[string]interface{}{
			"id":          "issue-uuid-123",
			"identifier":  "OPE-142",
			"title":       "Add bulk export",
			"description": "Implement CSV export for orders",
			"priority":    2,
			"state":       map[string]interface{}{"name": "Todo"},
		},
	}
	body, _ := json.Marshal(payload)
	sig := signPayload(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Linear-Signature", sig)

	w := httptest.NewRecorder()
	wh.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify task created in store
	task, err := taskStore.Get(context.Background(), "OPE-142")
	require.NoError(t, err)
	assert.Equal(t, "Add bulk export", task.Title)
}

func TestWebhook_InvalidSignature(t *testing.T) {
	wh, _ := setupWebhookTest(t)

	body := []byte(`{"action":"create","type":"Issue","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Linear-Signature", "invalid")

	w := httptest.NewRecorder()
	wh.HandleWebhook(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestWebhook_DuplicateIssue(t *testing.T) {
	wh, _ := setupWebhookTest(t)

	payload := map[string]interface{}{
		"action": "create",
		"type":   "Issue",
		"data": map[string]interface{}{
			"identifier":  "OPE-142",
			"title":       "Dup",
			"description": "",
			"priority":    2,
			"state":       map[string]interface{}{"name": "Todo"},
		},
	}
	body, _ := json.Marshal(payload)
	sig := signPayload(testWebhookSecret, body)

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req1.Header.Set("Linear-Signature", sig)
	w1 := httptest.NewRecorder()
	wh.HandleWebhook(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request (duplicate) — should still 200 but not create duplicate
	req2 := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req2.Header.Set("Linear-Signature", sig)
	w2 := httptest.NewRecorder()
	wh.HandleWebhook(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code) // idempotent
}

func TestWebhook_IgnoresNonIssue(t *testing.T) {
	wh, _ := setupWebhookTest(t)

	payload := map[string]interface{}{
		"action": "create",
		"type":   "Comment",
		"data":   map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	sig := signPayload(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/linear", bytes.NewReader(body))
	req.Header.Set("Linear-Signature", sig)
	w := httptest.NewRecorder()
	wh.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // ack but ignore
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/linear/ -v
```

- [ ] **Step 3: Implement webhook handler**

File: `apps/orchestrator/internal/linear/webhook.go`

```go
// Package linear handles Linear webhook intake and API communication.
package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/model"
	"github.com/openoms-org/openoms/apps/orchestrator/internal/store"
)

type WebhookHandler struct {
	secret    string
	taskStore *store.TaskStore
}

func NewWebhookHandler(secret string, taskStore *store.TaskStore) *WebhookHandler {
	return &WebhookHandler{secret: secret, taskStore: taskStore}
}

type webhookPayload struct {
	Action string          `json:"action"`
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data"`
}

type issueData struct {
	ID          string    `json:"id"`
	Identifier  string    `json:"identifier"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    int       `json:"priority"`
	State       stateData `json:"state"`
	Labels      []label   `json:"labels"`
}

type stateData struct {
	Name string `json:"name"`
}

type label struct {
	Name string `json:"name"`
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB max
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	// Verify signature
	sig := r.Header.Get("Linear-Signature")
	if !h.verifySignature(body, sig) {
		slog.Warn("webhook signature verification failed")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("unmarshal webhook payload", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Only process Issue events
	if payload.Type != "Issue" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var issue issueData
	if err := json.Unmarshal(payload.Data, &issue); err != nil {
		slog.Error("unmarshal issue data", "error", err)
		http.Error(w, "invalid issue data", http.StatusBadRequest)
		return
	}

	// Skip tickets with "manual" label
	for _, l := range issue.Labels {
		if l.Name == "manual" {
			slog.Info("skipping manual ticket", "identifier", issue.Identifier)
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	switch payload.Action {
	case "create":
		h.handleCreate(r.Context(), w, &issue)
	case "update":
		h.handleUpdate(r.Context(), w, &issue)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (h *WebhookHandler) handleCreate(ctx context.Context, w http.ResponseWriter, issue *issueData) {
	// Only enqueue if state is "Todo" or create was explicit
	if issue.State.Name != "Todo" && issue.State.Name != "" {
		// Created in Backlog or other state — ignore until moved to Todo
		if issue.State.Name != "Todo" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	task := model.NewTask(issue.Identifier, issue.Title, issue.Description)
	task.Priority = issue.Priority

	err := h.taskStore.Create(ctx, task)
	if errors.Is(err, store.ErrTaskExists) {
		slog.Debug("task already exists, skipping", "identifier", issue.Identifier)
		w.WriteHeader(http.StatusOK)
		return
	}
	if err != nil {
		slog.Error("create task from webhook", "error", err, "identifier", issue.Identifier)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("task enqueued from webhook", "identifier", issue.Identifier, "title", issue.Title)
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleUpdate(ctx context.Context, w http.ResponseWriter, issue *issueData) {
	// If status changed to "Todo" and task doesn't exist → enqueue
	if issue.State.Name == "Todo" {
		exists, err := h.taskStore.Exists(ctx, issue.Identifier)
		if err != nil {
			slog.Error("check task exists", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !exists {
			task := model.NewTask(issue.Identifier, issue.Title, issue.Description)
			task.Priority = issue.Priority
			if err := h.taskStore.Create(ctx, task); err != nil && !errors.Is(err, store.ErrTaskExists) {
				slog.Error("create task from update webhook", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			slog.Info("task enqueued from status update", "identifier", issue.Identifier)
		}
	}

	// If status changed to "Cancelled" → delete task
	if issue.State.Name == "Cancelled" {
		_ = h.taskStore.Delete(ctx, issue.Identifier)
		slog.Info("task cancelled via Linear", "identifier", issue.Identifier)
	}

	// If Blocked → Todo (human re-opens after escalation)
	if issue.State.Name == "Todo" {
		task, err := h.taskStore.Get(ctx, issue.Identifier)
		if err == nil && task.State == model.StateBlocked {
			_ = h.taskStore.UpdateState(ctx, issue.Identifier, model.StateQueued)
			slog.Info("blocked task re-queued from Linear", "identifier", issue.Identifier)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) verifySignature(body []byte, signature string) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
```

Wait — the `verifySignature` return type is wrong (string vs bool). Let me fix:

```go
func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/linear/ -v
```

Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/orchestrator/internal/linear/webhook.go apps/orchestrator/internal/linear/webhook_test.go
git commit -m "feat(orchestrator): add Linear webhook handler with HMAC verification

Handles issue.create and issue.update events. Deduplicates tasks,
skips 'manual' labeled tickets, re-queues blocked tasks on status change."
```

---

### Task 7: Linear API client (polling + status updates)

**Files:**
- Create: `apps/orchestrator/internal/linear/client.go`
- Create: `apps/orchestrator/internal/linear/client_test.go`

- [ ] **Step 1: Write client test**

File: `apps/orchestrator/internal/linear/client_test.go`

```go
package linear_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_FetchTodoIssues(t *testing.T) {
	// Mock Linear GraphQL API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "lin_api_test")

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"issues": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"id":          "uuid-1",
							"identifier":  "OPE-150",
							"title":       "Test issue",
							"description": "Test desc",
							"priority":    2,
							"state":       map[string]interface{}{"name": "Todo"},
							"labels":      map[string]interface{}{"nodes": []interface{}{}},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := linear.NewClient("lin_api_test", "OPE", linear.WithBaseURL(server.URL))
	issues, err := client.FetchTodoIssues(context.Background())
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.Equal(t, "OPE-150", issues[0].Identifier)
}

func TestClient_UpdateIssueState(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"issueUpdate": map[string]interface{}{
					"success": true,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := linear.NewClient("lin_api_test", "OPE", linear.WithBaseURL(server.URL))
	err := client.UpdateIssueState(context.Background(), "issue-uuid", "state-uuid-inprogress")
	require.NoError(t, err)

	// Verify GraphQL mutation was sent
	query, ok := receivedBody["query"].(string)
	require.True(t, ok)
	assert.Contains(t, query, "issueUpdate")
}

func TestClient_AddComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"commentCreate": map[string]interface{}{"success": true},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := linear.NewClient("lin_api_test", "OPE", linear.WithBaseURL(server.URL))
	err := client.AddComment(context.Background(), "issue-uuid", "PR created: https://github.com/...")
	require.NoError(t, err)
}
```

- [ ] **Step 2: Implement client**

File: `apps/orchestrator/internal/linear/client.go`

```go
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.linear.app/graphql"

type Client struct {
	apiKey  string
	teamID  string
	baseURL string
	http    *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

func NewClient(apiKey, teamID string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		teamID:  teamID,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Issue struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
}

func (c *Client) FetchTodoIssues(ctx context.Context) ([]Issue, error) {
	query := `query($teamId: String!) {
		issues(filter: {
			team: { key: { eq: $teamId } }
			state: { name: { eq: "Todo" } }
		}, first: 50) {
			nodes {
				id
				identifier
				title
				description
				priority
				state { name }
				labels { nodes { name } }
			}
		}
	}`

	var result struct {
		Data struct {
			Issues struct {
				Nodes []Issue `json:"nodes"`
			} `json:"issues"`
		} `json:"data"`
	}

	if err := c.graphql(ctx, query, map[string]interface{}{"teamId": c.teamID}, &result); err != nil {
		return nil, fmt.Errorf("fetch todo issues: %w", err)
	}
	return result.Data.Issues.Nodes, nil
}

func (c *Client) UpdateIssueState(ctx context.Context, issueID, stateID string) error {
	query := `mutation($id: String!, $stateId: String!) {
		issueUpdate(id: $id, input: { stateId: $stateId }) {
			success
		}
	}`

	var result struct {
		Data struct {
			IssueUpdate struct {
				Success bool `json:"success"`
			} `json:"issueUpdate"`
		} `json:"data"`
	}

	if err := c.graphql(ctx, query, map[string]interface{}{"id": issueID, "stateId": stateID}, &result); err != nil {
		return fmt.Errorf("update issue state: %w", err)
	}
	return nil
}

func (c *Client) AddComment(ctx context.Context, issueID, body string) error {
	query := `mutation($issueId: String!, $body: String!) {
		commentCreate(input: { issueId: $issueId, body: $body }) {
			success
		}
	}`

	var result struct {
		Data struct {
			CommentCreate struct {
				Success bool `json:"success"`
			} `json:"commentCreate"`
		} `json:"data"`
	}

	if err := c.graphql(ctx, query, map[string]interface{}{"issueId": issueID, "body": body}, &result); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

func (c *Client) graphql(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("linear API error: status %d, body: %s", resp.StatusCode, respBody)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Run test — expect PASS**

```bash
go test ./internal/linear/ -v
```

Expected: 7 tests PASS (4 webhook + 3 client).

- [ ] **Step 4: Commit**

```bash
git add apps/orchestrator/internal/linear/client.go apps/orchestrator/internal/linear/client_test.go
git commit -m "feat(orchestrator): add Linear GraphQL API client

Supports FetchTodoIssues (polling), UpdateIssueState, AddComment.
Uses functional options pattern for test mock injection."
```

---

### Task 8: Scheduler (event loop + polling fallback)

**Files:**
- Create: `apps/orchestrator/internal/scheduler/scheduler.go`
- Create: `apps/orchestrator/internal/scheduler/scheduler_test.go`

- [ ] **Step 1: Write scheduler test**

File: `apps/orchestrator/internal/scheduler/scheduler_test.go`

```go
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

	// Both tasks should be in queue
	queued, err := taskStore.ListByState(context.Background(), model.StateQueued)
	require.NoError(t, err)
	assert.Len(t, queued, 2)
}

func TestScheduler_PollSkipsDuplicates(t *testing.T) {
	taskStore, _ := setupSchedulerTest(t)

	// Pre-create task
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

	// Only 1 new task added (OPE-200 was duplicate)
	queued, err := taskStore.ListByState(context.Background(), model.StateQueued)
	require.NoError(t, err)
	assert.Len(t, queued, 2) // 1 existing + 1 new
}

func TestScheduler_PollSkipsManualLabeled(t *testing.T) {
	taskStore, _ := setupSchedulerTest(t)

	mock := &mockLinearClient{
		issues: []linear.Issue{
			{
				Identifier: "OPE-300",
				Title:      "Manual task",
				Priority:   2,
			},
		},
	}
	// Set the label on the mock issue
	mock.issues[0].Labels.Nodes = []struct {
		Name string `json:"name"`
	}{{Name: "manual"}}

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
		// OK — scheduler stopped
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop after cancel")
	}
}
```

- [ ] **Step 2: Implement scheduler**

File: `apps/orchestrator/internal/scheduler/scheduler.go`

```go
// Package scheduler implements the orchestrator's event loop and polling.
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

	// Initial poll on startup
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
		// Skip manual-labeled
		if hasLabel(issue, "manual") {
			continue
		}

		// Deduplication: skip if already in store
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
				continue // race condition with webhook — safe to skip
			}
			slog.Error("create task during poll", "error", err, "identifier", issue.Identifier)
			continue
		}
		enqueued++
	}

	if enqueued > 0 {
		slog.Info("poll enqueued tasks", "count", enqueued)
	}
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
```

- [ ] **Step 3: Run test — expect PASS**

```bash
go test ./internal/scheduler/ -v
```

Expected: 4 tests PASS.

- [ ] **Step 4: Commit**

```bash
git add apps/orchestrator/internal/scheduler/
git commit -m "feat(orchestrator): add scheduler with polling fallback

Event loop polls Linear every 5min (configurable), deduplicates
with Redis, skips manual-labeled tickets. Graceful shutdown via context."
```

---

### Task 9: Wire everything together in main.go

**Files:**
- Modify: `apps/orchestrator/cmd/orchestrator/main.go`

- [ ] **Step 1: Update main.go to wire all components**

The main.go from Task 1 Step 4 needs to be updated to include Linear webhook handler and scheduler. Update the `run()` function to add:

```go
// After taskStore creation, add:

// Linear client (for polling)
linearClient := linear.NewClient(cfg.LinearAPIKey, cfg.LinearTeamID)

// Webhook handler
webhookHandler := linear.NewWebhookHandler(cfg.LinearWebhookSecret, taskStore)

// Scheduler
pollInterval := time.Duration(cfg.PollIntervalSeconds) * time.Second
sched := scheduler.New(taskStore, linearClient, scheduler.WithPollInterval(pollInterval))

// Routes
r.Get("/health", healthHandler.Health)
r.Post("/api/webhooks/linear", webhookHandler.HandleWebhook)

// Start scheduler in background
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go sched.Run(ctx)
```

And add the imports:

```go
"github.com/openoms-org/openoms/apps/orchestrator/internal/linear"
"github.com/openoms-org/openoms/apps/orchestrator/internal/scheduler"
```

- [ ] **Step 2: Verify it compiles and runs**

```bash
cd /Users/rafs/praca/OpenOMS/apps/orchestrator
go build ./cmd/orchestrator/
```

Expected: binary compiles successfully.

- [ ] **Step 3: Run all tests**

```bash
go test ./... -v
```

Expected: all tests PASS (config: 3, model: 2, store: 6, handler: 1, linear: 7, scheduler: 4 = **23 tests total**).

- [ ] **Step 4: Commit**

```bash
git add apps/orchestrator/cmd/orchestrator/main.go
git commit -m "feat(orchestrator): wire all components in main

Connects config, Redis, Linear webhook, scheduler polling,
and health endpoint. Graceful shutdown on SIGINT/SIGTERM."
```

---

### Task 10: Docker + local dev setup

**Files:**
- Create: `apps/orchestrator/Dockerfile`
- Create: `apps/orchestrator/.env.example`

- [ ] **Step 1: Create Dockerfile**

File: `apps/orchestrator/Dockerfile`

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.work go.work.sum ./
COPY apps/orchestrator/go.mod apps/orchestrator/go.sum ./apps/orchestrator/
RUN cd apps/orchestrator && go mod download
COPY apps/orchestrator/ ./apps/orchestrator/
RUN cd apps/orchestrator && \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" -o /orchestrator ./cmd/orchestrator

FROM gcr.io/distroless/static-debian12
COPY --from=builder /orchestrator /orchestrator
EXPOSE 9090
USER nonroot:nonroot
ENTRYPOINT ["/orchestrator"]
```

- [ ] **Step 2: Create .env.example**

File: `apps/orchestrator/.env.example`

```bash
# Required
LINEAR_API_KEY=lin_api_xxx
LINEAR_WEBHOOK_SECRET=whsec_xxx
REDIS_URL=redis://localhost:6379

# Optional (defaults shown)
PORT=9090
ENV=development
LINEAR_TEAM_ID=OPE
MAX_CI_RETRIES=5
POLL_INTERVAL_SECONDS=300
BUDGET_MONTHLY_LIMIT=280

# Phase 2+ (not needed yet)
OPENROUTER_API_KEY=
GITHUB_TOKEN=
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
```

- [ ] **Step 3: Commit**

```bash
git add apps/orchestrator/Dockerfile apps/orchestrator/.env.example
git commit -m "feat(orchestrator): add Dockerfile and env example

Multi-stage build matching api-server pattern. Distroless runtime
image with nonroot user."
```

---

## Summary

**Phase 1 delivers:**
- New `apps/orchestrator/` Go service with 7 packages
- Linear webhook intake with HMAC signature verification
- Redis-backed task store with deduplication
- Full task state machine (16 states, all transitions from spec Section 19)
- Linear polling fallback (configurable interval)
- Health endpoint with Redis + queue status
- 23+ unit/integration tests
- Docker build ready

**Phase 1 does NOT include:**
- Agent dispatch (Phase 2)
- OpenRouter API calls (Phase 2)
- GitHub PR/branch management (Phase 3)
- Telegram notifications (Phase 4)
- Helm charts / NemoClaw config (Phase 5)

**Next:** Phase 2 plan — OpenRouter integration + PO agent + Dev agent dispatch.
