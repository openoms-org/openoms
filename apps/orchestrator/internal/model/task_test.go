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
		{model.StateAssigned, model.StateQueued, true},
		{model.StateDeveloping, model.StatePRCreated, true},
		{model.StateDeveloping, model.StateLocalFix, true},
		{model.StateLocalFix, model.StateDeveloping, true},
		{model.StateLocalFix, model.StateEscalated, true},
		{model.StateEscalated, model.StateAssigned, true},
		{model.StateEscalated, model.StateDeveloping, true},
		{model.StateEscalated, model.StateBlocked, true},
		{model.StatePRCreated, model.StateCIRunning, true},
		{model.StateCIRunning, model.StateSecurityReview, true},
		{model.StateCIRunning, model.StateCIFixing, true},
		{model.StateCIFixing, model.StateCIRunning, true},
		{model.StateCIFixing, model.StateBlocked, true},
		{model.StateSecurityReview, model.StateMerging, true},
		{model.StateSecurityReview, model.StateSecFixing, true},
		{model.StateSecFixing, model.StateCIRunning, true},
		{model.StateMerging, model.StateDone, true},
		{model.StateBlocked, model.StateQueued, true},
		{model.StateBlocked, model.StateCancelled, true},

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

func TestTask_TransitionTo_Valid(t *testing.T) {
	task := model.NewTask("OPE-1", "Test", "desc")
	err := task.TransitionTo(model.StateDecomposing)
	assert.NoError(t, err)
	assert.Equal(t, model.StateDecomposing, task.State)
}

func TestTask_TransitionTo_Invalid(t *testing.T) {
	task := model.NewTask("OPE-1", "Test", "desc")
	err := task.TransitionTo(model.StateDone)
	assert.Error(t, err)

	var transErr *model.InvalidTransitionError
	assert.ErrorAs(t, err, &transErr)
	assert.Equal(t, model.StateQueued, transErr.From)
	assert.Equal(t, model.StateDone, transErr.To)
}
