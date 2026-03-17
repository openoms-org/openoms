package model

import "time"

type TaskState string

const (
	StateQueued         TaskState = "queued"
	StateDecomposing    TaskState = "decomposing"
	StateSplit          TaskState = "split"
	StateAssigned       TaskState = "assigned"
	StateDeveloping     TaskState = "developing"
	StateLocalFix       TaskState = "local_fix"
	StateEscalated      TaskState = "escalated"
	StatePRCreated      TaskState = "pr_created"
	StateCIRunning      TaskState = "ci_running"
	StateCIFixing       TaskState = "ci_fixing"
	StateSecurityReview TaskState = "security_review"
	StateSecFixing      TaskState = "sec_fixing"
	StateMerging        TaskState = "merging"
	StateDone           TaskState = "done"
	StateBlocked        TaskState = "blocked"
	StateCancelled      TaskState = "cancelled"
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
	Priority      int       `json:"priority"`
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
		Priority:      2,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

var validTransitions = map[TaskState][]TaskState{
	StateQueued:         {StateDecomposing},
	StateDecomposing:    {StateAssigned, StateSplit, StateBlocked},
	StateSplit:          {},
	StateAssigned:       {StateDeveloping, StateQueued},
	StateDeveloping:     {StatePRCreated, StateLocalFix},
	StateLocalFix:       {StateDeveloping, StateEscalated},
	StateEscalated:      {StateAssigned, StateDeveloping, StateBlocked},
	StatePRCreated:      {StateCIRunning},
	StateCIRunning:      {StateSecurityReview, StateCIFixing},
	StateCIFixing:       {StateCIRunning, StateBlocked},
	StateSecurityReview: {StateMerging, StateSecFixing},
	StateSecFixing:      {StateCIRunning},
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
