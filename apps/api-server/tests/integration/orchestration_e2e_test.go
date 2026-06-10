//go:build integration

package integration

// Package integration / OPE-423d — end-to-end validation suite for the Phase-14
// fulfillment orchestration (OPE-414..423). These tests drive the FULL chain
// through the real services, workers, repositories and PostgreSQL (RLS-scoped),
// rather than any single layer in isolation, to prove the merged orchestration
// behaves correctly across the realistic operator scenarios.
//
// They MIRROR the existing harness (harness_test.go): appPool + database.WithTenant
// for tenant-scoped ops, superPool as the privileged worker pool, and the shared
// seed helpers (seedTenant / seedOrderWithStatus / seedFulfillmentOrder /
// seedProcessWithStatus / seedBlocker / seedFailedStep / seedUser). Gated behind
// the `integration` build tag + DATABASE_URL.
//
// Scope is HONEST: only the Phase-14 scenarios whose capabilities are MERGED are
// covered here. The scenarios that depend on still-unbuilt runtime capabilities
// are enumerated, with the reason each is deferred, in deferredPhase14Scenarios
// below (and in the OPE-423d report). They are NOT faked.
//
// DEFERRED Phase-14 scenarios (NOT covered here — need capabilities not yet in
// main; tracked as OPE-403 followups / blocked-on-external):
//
//   - missing-phone carrier blocker: needs the carrier capability-profile +
//     order-validation runtime that raises a typed carrier blocker from real
//     shipment preflight. Only the generic blocker plumbing is merged.
//   - unmapped-status-blocks-automation: needs the marketplace status-mapping
//     runtime (external_status_unmapped raised from a live poller), not just the
//     blocker code constant.
//   - dropship accept/reject + supplier tracking: needs the supplier
//     availability policy + a supplier order provider (BTP is verified at the SDK
//     level but the orchestrated accept/reject + tracking loop is a followup).
//   - feed-only manual step: needs the feed-only supplier capability profile that
//     emits a manual preflight step gate.
//   - carrier-no-tracking unsupported: needs the carrier capability profile that
//     marks tracking unsupported and skips await_tracking.
//   - backorder -> PO-receipt linkage: the backorder unit + receive_purchase_order
//     step exist, but there is no PO<->order link wiring to auto-resolve from a
//     real goods-receipt event yet.
//   - label-fail -> retry on a live carrier: needs a live/cert carrier sandbox; the
//     retry mechanics are covered here with seeded steps instead.
//   - webhook == poller parity: needs the live marketplace webhook + poller paths
//     to compare against each other (external integration env).
//   - n8n connector failure path: needs the n8n automation hub (enterprise infra),
//     out of scope for app integration tests.

import (
	"context"
	"fmt"
	"log/slog"
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

// TestE2E_OrderToProcessValidating_FullWorkerChain drives the order.created path
// END TO END through the production wiring: EnsureProcessForOrder (flag ON) creates
// the process AND enqueues the order.created outbox event, then the ACTUAL
// orchestration worker — draining the outbox via the OrchestrationDispatcher with a
// real OrderCreatedHandler registered — advances the process new -> validating and
// marks the event succeeded. (order_fulfillment_test.go invokes the handler
// directly; this asserts the worker+dispatcher chain that runs in production.)
func TestE2E_OrderToProcessValidating_FullWorkerChain(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)

	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,$3)`,
		orderID, tenant, "E2E Validating Customer")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID) })

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	svc := service.NewFulfillmentService(true, fRepo, orchRepo)

	// EnsureProcessForOrder: create the process + enqueue order.created in one tenant tx.
	var procID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		proc, e := svc.EnsureProcessForOrder(ctx, tx, tenant, orderID)
		if e != nil {
			return e
		}
		require.NotNil(t, proc)
		assert.Equal(t, model.ProcessStatusNew, proc.AggregateStatus, "process starts new")
		procID = proc.ID
		return nil
	}))

	// The order.created event is pending before the worker runs.
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, service.EventOrderCreated, events[0].EventType)
		assert.Equal(t, model.OutboxStatusPending, events[0].Status)
		return nil
	}))

	// Run the REAL worker with the order.created handler registered (production wiring:
	// the worker pool is the privileged superPool, exactly like the running service).
	disp := service.NewOrchestrationDispatcher()
	disp.Register(service.EventOrderCreated, service.NewOrderCreatedHandler(superPool, fRepo))
	w := worker.NewOrchestrationWorker(superPool, orchRepo, disp, fRepo, time.Second, slog.Default())
	require.NoError(t, w.Run(ctx))

	// Outcome: event succeeded, process advanced new -> validating, health still ok.
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, model.OutboxStatusSucceeded, events[0].Status, "worker drained order.created")

		proc, e := fRepo.GetProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Equal(t, model.ProcessStatusValidating, proc.AggregateStatus, "worker chain advanced new -> validating")
		assert.Equal(t, model.ProcessHealthOK, proc.HealthStatus)
		return nil
	}))

	// Re-running the worker is a no-op: nothing pending, process untouched.
	require.NoError(t, w.Run(ctx))
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		proc, e := fRepo.GetProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Equal(t, model.ProcessStatusValidating, proc.AggregateStatus, "re-run leaves the process at validating")
		return nil
	}))
}

// TestE2E_BlockerResolveThenStepRetry is the operator-recovery scenario end to end:
// a process carrying an OPEN blocker + a FAILED step is recovered by the two operator
// actions on the read service. ResolveBlocker flips the blocker to resolved (and
// audits it); RetryStep resets the failed step failed -> pending AND — because the
// read service is wired with the orchestration outbox — enqueues an idempotent
// fulfillment.step retry event the worker would consume. We assert the full chain
// via the read model + the outbox, including the cross-tenant guards.
func TestE2E_BlockerResolveThenStepRetry(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderA, _ := seedFulfillmentOrder(t, ctx, tenantA, "Recovery A")
	procA := seedProcessWithStatus(t, ctx, tenantA, orderA, model.ProcessStatusBlocked, model.ProcessHealthActionRequired)
	blockerA := seedBlocker(t, ctx, tenantA, procA.ID, model.BlockerIntegrationCapabilityMissing)
	stepA := seedFailedStep(t, ctx, tenantA, procA.ID)
	actor := seedUser(t, ctx, tenantA)
	const actorIP = "198.51.100.9"

	read := newReadService()
	orchRepo := repository.NewOrchestrationRepository()

	// --- Resolve the blocker (operator action) ---
	resolved, err := read.ResolveBlocker(ctx, tenantA, blockerA.ID, actor, actorIP)
	require.NoError(t, err)
	assert.Equal(t, model.BlockerStatusResolved, resolved.Status)
	require.NotNil(t, resolved.ResolvedAt)
	assertAuditEntry(t, ctx, tenantA, "fulfillment_blocker", blockerA.ID, "fulfillment.blocker.resolved")

	// The detail view reflects the now-resolved blocker.
	detail, err := read.GetProcessDetail(ctx, tenantA, procA.ID)
	require.NoError(t, err)
	require.Len(t, detail.Blockers, 1)
	assert.Equal(t, model.BlockerStatusResolved, detail.Blockers[0].Status, "resolved blocker surfaces as resolved in detail")

	// --- Retry the failed step (operator action) ---
	retried, err := read.RetryStep(ctx, tenantA, stepA.ID, actor, actorIP)
	require.NoError(t, err)
	assert.Equal(t, model.FulfillmentStatusPending, retried.Status, "retry resets failed -> pending")
	assert.Equal(t, stepA.Attempts, retried.Attempts, "attempts preserved (worker increments on its next run)")
	assertAuditEntry(t, ctx, tenantA, "fulfillment_step", stepA.ID, "fulfillment.step.retried")

	// The retry enqueued an idempotent fulfillment.step retry event on the process,
	// keyed on the step + its (preserved) attempt count, ready for the orchestration
	// worker. seedFailedStep records the step with Attempts=1, which RetryStep keeps.
	wantIdem := fmt.Sprintf("%s:retry:%s:%d", service.EventFulfillmentStep, stepA.ID, stepA.Attempts)
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procA.ID)
		if e != nil {
			return e
		}
		var retryEvents []model.OrchestrationOutboxEvent
		for _, ev := range events {
			if ev.EventType == service.EventFulfillmentStep {
				retryEvents = append(retryEvents, ev)
			}
		}
		require.Len(t, retryEvents, 1, "exactly one fulfillment.step retry event enqueued")
		assert.Equal(t, wantIdem, retryEvents[0].IdempotencyKey)
		assert.Equal(t, model.OutboxStatusPending, retryEvents[0].Status)
		require.NotNil(t, retryEvents[0].UnitID, "retry event is unit-scoped")
		return nil
	}))

	// Retrying the now-pending step is rejected (not in a retryable state).
	_, err = read.RetryStep(ctx, tenantA, stepA.ID, actor, actorIP)
	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve, "retrying a non-failed/blocked step yields a ValidationError")

	// --- Cross-tenant guards: tenant B can touch NEITHER A's blocker NOR A's step ---
	_, err = read.ResolveBlocker(ctx, tenantB, blockerA.ID, uuid.New(), actorIP)
	require.ErrorIs(t, err, service.ErrFulfillmentNotFound, "tenant B must NOT resolve tenant A's blocker")
	_, err = read.RetryStep(ctx, tenantB, stepA.ID, uuid.New(), actorIP)
	require.ErrorIs(t, err, service.ErrFulfillmentNotFound, "tenant B must NOT retry tenant A's step")
}

// TestE2E_MixedOrderAllCancelledAggregatesCancelled covers the specific
// all-cancelled aggregation rule (ADR-424): when every unit of a mixed order is
// cancelled, the whole process aggregates to cancelled — NOT completed — driven
// through the live recording service against the real DB.
func TestE2E_MixedOrderAllCancelledAggregatesCancelled(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenant, "All-Cancelled Customer")

	svc := newUnitService()
	fRepo := repository.NewFulfillmentRepository()

	// A mixed order: a warehouse unit + a dropship unit.
	wh := svc.EnsureUnit(ctx, tenant, orderID, model.UnitTypeWarehouse, "", nil)
	require.NotNil(t, wh)
	ds := svc.EnsureUnit(ctx, tenant, orderID, model.UnitTypeDropship, uuid.New().String(), nil)
	require.NotNil(t, ds)

	// One cancelled, one still running -> NOT cancelled yet (running dominates).
	svc.RecordUnitTransition(ctx, tenant, wh.ID, model.FulfillmentStatusCancelled)
	svc.RecordUnitTransition(ctx, tenant, ds.ID, model.FulfillmentStatusRunning)
	proc := svc.AggregateMixedOrder(ctx, tenant, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusInProgress, proc.AggregateStatus, "a running unit keeps the order in progress")

	// Now cancel the dropship unit too -> EVERY unit cancelled -> process cancelled.
	svc.RecordUnitTransition(ctx, tenant, ds.ID, model.FulfillmentStatusCancelled)
	proc = svc.AggregateMixedOrder(ctx, tenant, orderID)
	require.NotNil(t, proc)
	assert.Equal(t, model.ProcessStatusCancelled, proc.AggregateStatus, "all units cancelled -> process cancelled (not completed)")
	assert.Equal(t, model.ProcessHealthOK, proc.HealthStatus)

	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		units, e := fRepo.ListUnits(ctx, tx, processID)
		if e != nil {
			return e
		}
		require.Len(t, units, 2)
		for _, u := range units {
			assert.Equal(t, model.FulfillmentStatusCancelled, u.Status, "both units cancelled")
		}
		return nil
	}))
}

// TestE2E_BackfillThenParityMet chains the OPE-423a backfill into the OPE-423b/c
// parity report: a tenant with several non-terminal, process-less orders (plus
// terminal noise) reports a coverage GAP and a NOT-met verdict; a write-run backfill
// then creates exactly the missing processes; and a fresh parity report shows
// orders_missing_process -> 0, coverage 1.0 and ProcessCoverageMet=true. A dry run
// in between writes NOTHING, proving the gate verdict is driven by real writes only.
func TestE2E_BackfillThenParityMet(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)

	// Eligible: non-terminal, no process yet.
	eligible := []uuid.UUID{
		seedOrderWithStatus(t, ctx, tenant, model.OrderStatusNew),
		seedOrderWithStatus(t, ctx, tenant, model.OrderStatusProcessing),
		seedOrderWithStatus(t, ctx, tenant, model.OrderStatusReadyToShip),
	}
	// Terminal noise: excluded from the non-terminal population entirely.
	seedOrderWithStatus(t, ctx, tenant, model.OrderStatusCompleted)
	seedOrderWithStatus(t, ctx, tenant, model.OrderStatusCancelled)

	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	backfill := service.NewFulfillmentBackfillService(fRepo, orchRepo)
	read := newReadService()

	// --- BEFORE: parity shows the full coverage gap, gate NOT met ---
	before, err := read.ParityReport(ctx, tenant, 0) // 0 -> default threshold
	require.NoError(t, err)
	assert.Equal(t, len(eligible), before.NonTerminalOrders, "only non-terminal orders count (terminal noise excluded)")
	assert.Equal(t, len(eligible), before.OrdersMissingProcess, "every eligible order lacks a process")
	assert.Equal(t, 0, before.FulfillmentProcesses)
	assert.InDelta(t, 0.0, before.ProcessCoverage, 1e-9, "zero coverage before backfill")
	assert.False(t, before.ProcessCoverageMet, "gate NOT met while the gap exists")

	// --- DRY RUN: counts the gap but writes nothing; parity verdict unchanged ---
	var dry service.BackfillReport
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		r, e := backfill.BackfillActiveOrderProcesses(ctx, tx, tenant, service.BackfillOptions{DryRun: true})
		dry = r
		return e
	}))
	assert.Equal(t, len(eligible), dry.NeedingProcess, "dry run counts what a write run would create")
	assert.Equal(t, 0, dry.Created)
	for _, id := range eligible {
		assert.Equal(t, 0, countProcessesForOrder(t, ctx, tenant, id), "dry run creates no process")
	}
	stillBefore, err := read.ParityReport(ctx, tenant, 0)
	require.NoError(t, err)
	assert.Equal(t, len(eligible), stillBefore.OrdersMissingProcess, "dry run leaves the gap unchanged")
	assert.False(t, stillBefore.ProcessCoverageMet)

	// --- WRITE RUN: creates exactly the missing processes (+ one order.created each) ---
	var report service.BackfillReport
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		r, e := backfill.BackfillActiveOrderProcesses(ctx, tx, tenant, service.BackfillOptions{DryRun: false})
		report = r
		return e
	}))
	assert.Equal(t, len(eligible), report.Scanned)
	assert.Equal(t, len(eligible), report.Created)
	assert.Equal(t, 0, report.Errors)
	for _, id := range eligible {
		assert.Equal(t, 1, countProcessesForOrder(t, ctx, tenant, id), "one process per eligible order")
		assert.Equal(t, 1, countOrderCreatedEventsForOrder(t, ctx, tenant, id), "one order.created event per eligible order")
	}

	// --- AFTER: parity gap closed, coverage 1.0, gate MET ---
	after, err := read.ParityReport(ctx, tenant, 0)
	require.NoError(t, err)
	assert.Equal(t, len(eligible), after.NonTerminalOrders, "population unchanged by backfill")
	assert.Equal(t, 0, after.OrdersMissingProcess, "orders_missing_process -> 0 after backfill")
	assert.Equal(t, len(eligible), after.FulfillmentProcesses, "every non-terminal order now has a process")
	assert.InDelta(t, 1.0, after.ProcessCoverage, 1e-9, "full coverage after backfill")
	assert.True(t, after.ProcessCoverageMet, "process_coverage_met=true after backfill closes the gap")
}
