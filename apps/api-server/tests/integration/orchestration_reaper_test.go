//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// seedClaimedEvent enqueues an outbox event for a fresh order/process and force-claims
// it with the given claimed_at age via the superuser pool (simulating a worker that
// claimed the row and then crashed before marking it).
func seedClaimedEvent(t *testing.T, ctx context.Context, tenantID uuid.UUID, age time.Duration, attempts, maxAttempts int) model.OrchestrationOutboxEvent {
	t.Helper()
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantID, "Reaper Customer")
	repo := repository.NewOrchestrationRepository()
	var ev *model.OrchestrationOutboxEvent
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, created, err := repo.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
			TenantID:       tenantID,
			ProcessID:      processID,
			EventType:      "reaper.test",
			IdempotencyKey: "reaper.test:" + orderID.String(),
			Payload:        map[string]any{"order_id": orderID.String()},
		})
		ev = created
		return err
	}))
	require.NotNil(t, ev)
	_, err := superPool.Exec(ctx,
		`UPDATE orchestration_outbox
		    SET status='claimed', claimed_at = now() - make_interval(secs => $2),
		        attempts = $3, max_attempts = $4, updated_at = now()
		  WHERE id = $1`,
		ev.ID, age.Seconds(), attempts, maxAttempts)
	require.NoError(t, err)
	ev.Attempts, ev.MaxAttempts = attempts, maxAttempts
	return *ev
}

func TestOrchestrationReaper_RepoMethods(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	repo := repository.NewOrchestrationRepository()

	stale := seedClaimedEvent(t, ctx, tenantA, 30*time.Minute, 0, 5)
	fresh := seedClaimedEvent(t, ctx, tenantA, 1*time.Minute, 0, 5)

	// ListStaleClaimed returns only the stale row.
	got, err := repo.ListStaleClaimed(ctx, superPool, 10*time.Minute, 50)
	require.NoError(t, err)
	ids := map[uuid.UUID]bool{}
	for i := range got {
		ids[got[i].ID] = true
	}
	assert.True(t, ids[stale.ID], "stale claimed row must be listed")
	assert.False(t, ids[fresh.ID], "fresh claimed row must NOT be listed")

	// FailRunningAttempts closes a dangling running attempt for the stale event.
	att, err := repo.StartAttempt(ctx, superPool, tenantA, stale.ID, stale.Attempts+1)
	require.NoError(t, err)
	n, err := repo.FailRunningAttempts(ctx, superPool, stale.ID, "reaped: worker crashed mid-attempt")
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	var status, errMsg string
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status, error FROM orchestration_attempts WHERE id = $1`, att.ID).Scan(&status, &errMsg))
	assert.Equal(t, model.AttemptStatusFailed, status)
	assert.Contains(t, errMsg, "reaped")
}
