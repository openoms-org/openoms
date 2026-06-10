//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// newReadService builds the OPE-419 read service over the RLS-scoped app pool,
// mirroring production wiring (same repos as OPE-414..418). OPE-422 best-effort
// metrics + audit are wired so the operator-action audit + gauge paths are
// exercised end-to-end against the real DB.
func newReadService() *service.FulfillmentReadService {
	return service.NewFulfillmentReadService(
		appPool,
		repository.NewFulfillmentRepository(),
		repository.NewFulfillmentAttemptRepository(),
		repository.NewOrchestrationRepository(),
	).WithMetrics(obsmetrics.NewFulfillmentMetrics()).WithAudit(repository.NewAuditRepository())
}

// seedProcessWithStatus creates a fulfillment process for an order and forces its
// aggregate/health status (the recording defaults to new/ok). Returns the process.
func seedProcessWithStatus(t *testing.T, ctx context.Context, tenantID, orderID uuid.UUID, aggregate, health string) model.FulfillmentProcess {
	t.Helper()
	fRepo := repository.NewFulfillmentRepository()
	var proc *model.FulfillmentProcess
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		p, e := fRepo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantID, OrderID: orderID})
		if e != nil {
			return e
		}
		p, e = fRepo.UpdateProcessStatus(ctx, tx, p.ID, aggregate, health)
		if e != nil {
			return e
		}
		proc = p
		return nil
	}))
	require.NotNil(t, proc)
	return *proc
}

// seedBlocker creates an open blocker on a process (RLS-scoped). Returns it.
func seedBlocker(t *testing.T, ctx context.Context, tenantID, processID uuid.UUID, code string) model.FulfillmentBlocker {
	t.Helper()
	fRepo := repository.NewFulfillmentRepository()
	var b *model.FulfillmentBlocker
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		out, e := fRepo.CreateBlocker(ctx, tx, model.FulfillmentBlocker{
			TenantID:    tenantID,
			ProcessID:   processID,
			Code:        code,
			Description: "seeded blocker",
		})
		if e != nil {
			return e
		}
		b = out
		return nil
	}))
	require.NotNil(t, b)
	return *b
}

// seedFailedStep creates a unit + a failed step on a process and returns the step.
func seedFailedStep(t *testing.T, ctx context.Context, tenantID, processID uuid.UUID) model.FulfillmentStep {
	t.Helper()
	fRepo := repository.NewFulfillmentRepository()
	var step *model.FulfillmentStep
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		unit, e := fRepo.CreateUnit(ctx, tx, model.FulfillmentUnit{
			TenantID:  tenantID,
			ProcessID: processID,
			UnitType:  model.UnitTypeWarehouse,
		})
		if e != nil {
			return e
		}
		s, e := fRepo.CreateStep(ctx, tx, model.FulfillmentStep{
			TenantID: tenantID,
			UnitID:   unit.ID,
			StepKey:  model.StepCreateShipment,
			Status:   model.FulfillmentStatusFailed,
			Attempts: 1,
		})
		if e != nil {
			return e
		}
		step = s
		return nil
	}))
	require.NotNil(t, step)
	return *step
}

// TestFulfillmentReadAPI_CrossTenantIsolation is the isolation proof: tenant A's
// read service must NEVER surface tenant B's processes, blockers, summary counts
// or exceptions, and detail-by-id for B's process must return NotFound to A.
func TestFulfillmentReadAPI_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderA, _ := seedFulfillmentOrder(t, ctx, tenantA, "Reader A") // helper already creates a process; ignore it
	orderB, _ := seedFulfillmentOrder(t, ctx, tenantB, "Reader B")

	// Tenant A: one blocked (with blocker) + one ready process.
	aBlocked := seedProcessWithStatus(t, ctx, tenantA, orderA, model.ProcessStatusBlocked, model.ProcessHealthActionRequired)
	seedBlocker(t, ctx, tenantA, aBlocked.ID, model.BlockerIntegrationCapabilityMissing)
	aReadyOrder, _ := seedFulfillmentOrder(t, ctx, tenantA, "Reader A2")
	_ = seedProcessWithStatus(t, ctx, tenantA, aReadyOrder, model.ProcessStatusReady, model.ProcessHealthOK)

	// Tenant B: a distinct blocked + waiting_external process, each with a blocker.
	bBlocked := seedProcessWithStatus(t, ctx, tenantB, orderB, model.ProcessStatusBlocked, model.ProcessHealthActionRequired)
	seedBlocker(t, ctx, tenantB, bBlocked.ID, model.BlockerExternalStatusUnmapped)
	bWaitOrder, _ := seedFulfillmentOrder(t, ctx, tenantB, "Reader B2")
	bWaiting := seedProcessWithStatus(t, ctx, tenantB, bWaitOrder, model.ProcessStatusWaitingExternal, model.ProcessHealthWarning)
	seedBlocker(t, ctx, tenantB, bWaiting.ID, model.BlockerSupplierAvailabilityStale)

	svc := newReadService()

	// --- List: tenant A only sees its own processes ---
	list, err := svc.ListProcesses(ctx, tenantA, nil, nil, 100, 0)
	require.NoError(t, err)
	for _, p := range list.Items {
		assert.Equal(t, tenantA, p.TenantID, "list must not leak tenant B processes")
		assert.NotEqual(t, bBlocked.ID, p.ID)
		assert.NotEqual(t, bWaiting.ID, p.ID)
	}
	// The seed helper for orderA created an extra process, so A has >= 3 here; B's
	// 3 rows must NOT inflate this. Assert the total equals A's own row count.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment_processes`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, n, list.Total, "list total must equal tenant A's own process count (no B rows)")
		return nil
	}))

	// --- Detail by id: A can read its own, but B's process is NotFound to A ---
	detail, err := svc.GetProcessDetail(ctx, tenantA, aBlocked.ID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, aBlocked.ID, detail.Process.ID)
	require.Len(t, detail.Blockers, 1)
	assert.Equal(t, model.BlockerIntegrationCapabilityMissing, detail.Blockers[0].Code)

	_, err = svc.GetProcessDetail(ctx, tenantA, bBlocked.ID)
	require.ErrorIs(t, err, service.ErrFulfillmentNotFound, "tenant A must NOT read tenant B's process by id")

	// Order-scoped detail for B's order must also be NotFound to A.
	_, err = svc.GetOrderFulfillment(ctx, tenantA, orderB)
	require.ErrorIs(t, err, service.ErrFulfillmentNotFound, "tenant A must NOT read tenant B's order fulfillment")

	// --- Summary: counts only tenant A's processes ---
	summaryA, err := svc.OperationsSummary(ctx, tenantA)
	require.NoError(t, err)
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment_processes`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, n, summaryA.Total, "summary total must equal tenant A's own process count")
		return nil
	}))
	assert.GreaterOrEqual(t, summaryA.Buckets[service.BucketBlocked], 1, "A has at least one blocked process")

	// Tenant B's summary is independent: it must equal B's own row count, not A's,
	// proving the two tenants' counts never bleed into each other.
	summaryB, err := svc.OperationsSummary(ctx, tenantB)
	require.NoError(t, err)
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM fulfillment_processes`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, n, summaryB.Total, "tenant B summary total must equal tenant B's own process count")
		return nil
	}))

	// --- Exceptions: only tenant A's actionable processes, never B's ---
	exA, err := svc.OperationsExceptions(ctx, tenantA, 100)
	require.NoError(t, err)
	require.NotEmpty(t, exA)
	for _, item := range exA {
		assert.Equal(t, tenantA, item.Process.TenantID, "exceptions must not leak tenant B processes")
		assert.NotEqual(t, bBlocked.ID, item.Process.ID)
		assert.NotEqual(t, bWaiting.ID, item.Process.ID)
		if item.TopBlocker != nil {
			assert.Equal(t, tenantA, item.TopBlocker.TenantID, "exception blocker must belong to tenant A")
		}
	}

	// --- Integration capability summary: scoped to tenant A ---
	capA, err := svc.IntegrationCapabilitySummary(ctx, tenantA, 1000)
	require.NoError(t, err)
	assert.Equal(t, summaryA.Total, capA.TotalProcesses, "capability denominator is tenant A's own total")
	assert.GreaterOrEqual(t, capA.UnsupportedCapabilityProcesses, 1, "A's capability blocker is counted")
}

// TestFulfillmentReadAPI_ResolveBlockerAndRetryStep covers the two operator
// actions end-to-end against a real DB, plus their cross-tenant guards.
func TestFulfillmentReadAPI_ResolveBlockerAndRetryStep(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderA, _ := seedFulfillmentOrder(t, ctx, tenantA, "Action A")
	procA := seedProcessWithStatus(t, ctx, tenantA, orderA, model.ProcessStatusBlocked, model.ProcessHealthActionRequired)
	blockerA := seedBlocker(t, ctx, tenantA, procA.ID, model.BlockerIntegrationCapabilityMissing)
	stepA := seedFailedStep(t, ctx, tenantA, procA.ID)

	// Seed a real user as the operator: audit_log.user_id has a FK to users, so a
	// non-existent actor would (correctly, best-effort) be dropped without writing an
	// audit row. We assert the audit row IS written for a valid operator.
	actor := seedUser(t, ctx, tenantA)
	const actorIP = "203.0.113.7"
	svc := newReadService()

	// --- ResolveBlocker happy path ---
	resolved, err := svc.ResolveBlocker(ctx, tenantA, blockerA.ID, actor, actorIP)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, model.BlockerStatusResolved, resolved.Status)
	require.NotNil(t, resolved.ResolvedAt)

	// OPE-422: the operator action wrote a best-effort tenant audit entry.
	assertAuditEntry(t, ctx, tenantA, "fulfillment_blocker", blockerA.ID, "fulfillment.blocker.resolved")

	// Resolving is reflected in the detail's open-blocker-free view.
	detail, err := svc.GetProcessDetail(ctx, tenantA, procA.ID)
	require.NoError(t, err)
	require.Len(t, detail.Blockers, 1)
	assert.Equal(t, model.BlockerStatusResolved, detail.Blockers[0].Status)

	// Tenant B cannot resolve tenant A's blocker.
	_, err = svc.ResolveBlocker(ctx, tenantB, blockerA.ID, uuid.New(), actorIP)
	require.ErrorIs(t, err, service.ErrFulfillmentNotFound, "tenant B must NOT resolve tenant A's blocker")

	// --- RetryStep happy path: failed -> pending ---
	retried, err := svc.RetryStep(ctx, tenantA, stepA.ID, actor, actorIP)
	require.NoError(t, err)
	require.NotNil(t, retried)
	assert.Equal(t, model.FulfillmentStatusPending, retried.Status)
	assert.Equal(t, stepA.Attempts, retried.Attempts, "attempts preserved on retry")

	// OPE-422: the retry action wrote a best-effort tenant audit entry.
	assertAuditEntry(t, ctx, tenantA, "fulfillment_step", stepA.ID, "fulfillment.step.retried")

	// Retrying a now-pending step is not retryable -> ValidationError.
	_, err = svc.RetryStep(ctx, tenantA, stepA.ID, actor, actorIP)
	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve, "retrying a non-failed/blocked step yields a ValidationError")

	// Tenant B cannot retry tenant A's step.
	_, err = svc.RetryStep(ctx, tenantB, stepA.ID, uuid.New(), actorIP)
	require.ErrorIs(t, err, service.ErrFulfillmentNotFound, "tenant B must NOT retry tenant A's step")
}

// assertAuditEntry verifies a best-effort operator-action audit row was written for
// the given entity with the expected action (OPE-422), scoped to the tenant (RLS).
func assertAuditEntry(t *testing.T, ctx context.Context, tenantID uuid.UUID, entityType string, entityID uuid.UUID, action string) {
	t.Helper()
	auditRepo := repository.NewAuditRepository()
	var entries []model.AuditLogEntry
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		e, err := auditRepo.ListByEntity(ctx, tx, entityType, entityID)
		entries = e
		return err
	}))
	found := false
	for _, e := range entries {
		if e.Action == action {
			found = true
			break
		}
	}
	assert.Truef(t, found, "expected audit entry %q for %s %s", action, entityType, entityID)
}
