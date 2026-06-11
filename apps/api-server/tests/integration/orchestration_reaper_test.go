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
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
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

// TestOrchestrationReaper_EndToEnd: a stale claimed event is requeued by the reap
// pass with a backoff (interrupted attempt counts), then processed once due; a fresh
// claim is untouched; an exhausted stale claim fails permanently with a blocker.
func TestOrchestrationReaper_EndToEnd(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	repo := repository.NewOrchestrationRepository()
	fRepo := repository.NewFulfillmentRepository()

	disp := service.NewOrchestrationDispatcher()
	disp.Register("reaper.test", fakeOrchHandler(func(model.OrchestrationOutboxEvent) error {
		return nil // ack — we only care about the reap + reprocess mechanics
	}))
	w := worker.NewOrchestrationWorker(superPool, repo, disp, fRepo, time.Second, nil)

	// (a) stale claim, attempts available -> requeued pending with backoff.
	stale := seedClaimedEvent(t, ctx, tenantA, 30*time.Minute, 0, 5)
	// (b) fresh claim -> untouched by the reaper.
	fresh := seedClaimedEvent(t, ctx, tenantA, 1*time.Minute, 0, 5)
	// (c) stale claim with attempts exhausted (attempts+1 >= max) -> permanent + blocker.
	exhausted := seedClaimedEvent(t, ctx, tenantA, 30*time.Minute, 4, 5)

	require.NoError(t, w.Run(ctx))

	// (a) The reap requeues with nextRetryAt in the future, so the SAME tick's
	// ClaimDue must NOT pick it up: pending, interrupted attempt counted, claim cleared.
	requeued, err := repo.GetEvent(ctx, superPool, stale.ID)
	require.NoError(t, err)
	assert.Equal(t, model.OutboxStatusPending, requeued.Status, "stale claim must be requeued to pending")
	assert.Equal(t, 1, requeued.Attempts, "the interrupted attempt counts toward max_attempts")
	assert.Nil(t, requeued.ClaimedAt, "the stale claim must be cleared")
	assert.Contains(t, requeued.LastError, "reaped")

	// (b) Fresh claim untouched.
	var status string
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status FROM orchestration_outbox WHERE id = $1`, fresh.ID).Scan(&status))
	assert.Equal(t, model.OutboxStatusClaimed, status, "fresh claim must not be reaped")

	// (c) Exhausted stale claim fails permanently.
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status FROM orchestration_outbox WHERE id = $1`, exhausted.ID).Scan(&status))
	assert.Equal(t, model.OutboxStatusFailed, status, "exhausted stale claim must fail permanently")

	// The exhausted event must have opened a blocker on its process.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		blockers, err := fRepo.ListBlockers(ctx, tx, exhausted.ProcessID)
		if err != nil {
			return err
		}
		require.Len(t, blockers, 1, "exhausted reap must open a blocker")
		assert.Contains(t, blockers[0].Description, "reaped")
		return nil
	}))

	// (a, continued) Once due again, the requeued event processes normally to succeeded.
	_, err = superPool.Exec(ctx,
		`UPDATE orchestration_outbox SET next_attempt_at = now() WHERE id = $1`, stale.ID)
	require.NoError(t, err)
	require.NoError(t, w.Run(ctx))
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT status FROM orchestration_outbox WHERE id = $1`, stale.ID).Scan(&status))
	assert.Equal(t, model.OutboxStatusSucceeded, status, "requeued stale claim must process to succeeded once due")
}
