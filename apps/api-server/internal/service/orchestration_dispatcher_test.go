package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type stubHandler struct{ err error }

func (h stubHandler) Handle(context.Context, model.OrchestrationOutboxEvent) error { return h.err }

func TestOrchestrationDispatcher(t *testing.T) {
	d := NewOrchestrationDispatcher()

	// Unregistered event type -> permanent failure (must not loop forever).
	err := d.Dispatch(context.Background(), model.OrchestrationOutboxEvent{EventType: "nope"})
	require.Error(t, err)
	assert.True(t, model.IsPermanent(err), "unregistered event type must be a permanent failure")

	// Registered handler success.
	d.Register("ok", stubHandler{err: nil})
	assert.NoError(t, d.Dispatch(context.Background(), model.OrchestrationOutboxEvent{EventType: "ok"}))

	// Registered handler error propagates (retryable).
	boom := errors.New("boom")
	d.Register("boom", stubHandler{err: boom})
	err = d.Dispatch(context.Background(), model.OrchestrationOutboxEvent{EventType: "boom"})
	assert.ErrorIs(t, err, boom)
	assert.False(t, model.IsPermanent(err))
}
