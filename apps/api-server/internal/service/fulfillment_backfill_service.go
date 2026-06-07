package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// defaultBackfillBatchSize bounds how many still-missing orders a single
// BackfillActiveOrderProcesses call drains for one tenant. The backfill is
// resumable: re-running finishes whatever remains, so a modest batch keeps each
// run's transaction short while progress accumulates across runs.
const defaultBackfillBatchSize = 200

// backfillProcessRepo is the slice of the fulfillment repository the backfill
// service needs: find eligible orders, look up an existing process, create one.
// Narrowed to an interface so the service is unit-testable with a stub.
type backfillProcessRepo interface {
	ListOrderIDsMissingProcess(ctx context.Context, tx pgx.Tx, terminalStatuses []string, limit int) ([]uuid.UUID, error)
	GetProcessByOrder(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.FulfillmentProcess, error)
	CreateProcess(ctx context.Context, tx pgx.Tx, p model.FulfillmentProcess) (*model.FulfillmentProcess, error)
}

// backfillEventRepo is the slice of the orchestration repository the backfill
// service needs: idempotently enqueue the initial order.created outbox event.
type backfillEventRepo interface {
	EnqueueEvent(ctx context.Context, q repository.Querier, e model.OrchestrationOutboxEvent) (bool, *model.OrchestrationOutboxEvent, error)
}

// FulfillmentBackfillService backfills fulfillment PROCESSES for existing
// non-terminal orders that do not have one yet (OPE-423a), so a deployment can move
// from legacy scattered order state to process-backed orchestration. It deliberately
// does NOT depend on FulfillmentService's FULFILLMENT_PROCESS_ENABLED gate: the
// whole point of the backfill is to create processes for historical orders even on a
// deployment where live process creation was previously off. It mirrors
// FulfillmentService.EnsureProcessForOrder exactly — same idempotent
// GetProcessByOrder -> CreateProcess path, same order.created event, same
// idempotency key — so backfilled and live-created processes are byte-for-byte
// identical and a backfill can never create a duplicate process or a second event.
//
// The whole worker is gated by FULFILLMENT_BACKFILL_ENABLED (the worker is a no-op
// when off); writing additionally requires the DryRun option to be false (driven by
// FULFILLMENT_BACKFILL_DRY_RUN, default true). This service implements the data path
// only — the two flags live in the worker/config layer.
type FulfillmentBackfillService struct {
	fulfillment   backfillProcessRepo
	orchestration backfillEventRepo
}

// NewFulfillmentBackfillService creates a FulfillmentBackfillService.
func NewFulfillmentBackfillService(fulfillment *repository.FulfillmentRepository, orchestration *repository.OrchestrationRepository) *FulfillmentBackfillService {
	return &FulfillmentBackfillService{fulfillment: fulfillment, orchestration: orchestration}
}

// BackfillOptions configures a single backfill pass for one tenant.
type BackfillOptions struct {
	// DryRun, when true (the default safe mode), only COUNTS the eligible orders —
	// it performs NO writes. Set false to actually create processes + events.
	DryRun bool
	// BatchSize caps how many still-missing orders this call drains. <=0 uses the
	// default. The call is resumable: re-run to finish the rest.
	BatchSize int
}

// BackfillReport is the per-tenant outcome of a backfill pass.
type BackfillReport struct {
	// Scanned is the number of eligible (non-terminal, process-less) orders this
	// pass examined (bounded by BatchSize).
	Scanned int
	// NeedingProcess is how many of the scanned orders still need a process. In a
	// dry run this is the count of what a write run WOULD create.
	NeedingProcess int
	// Created is how many processes (+ order.created events) this pass actually
	// created. Always 0 in a dry run.
	Created int
	// Skipped is how many scanned orders already had a process by the time the
	// write was attempted (idempotent no-op — never re-created).
	Skipped int
	// Errors is how many orders failed to backfill (per-order failures are isolated;
	// the pass continues).
	Errors int
}

// BackfillActiveOrderProcesses backfills fulfillment processes for one tenant's
// non-terminal, process-less orders. It MUST be called inside a
// database.WithTenant transaction (tx is RLS-scoped to the tenant): the eligibility
// query, the existence check and every write run in that tenant's context.
//
// Behaviour:
//   - It selects up to BatchSize still-missing orders (oldest first).
//   - DryRun: increments NeedingProcess per eligible order and writes nothing.
//   - Write run: for each order it mirrors EnsureProcessForOrder — re-checks for an
//     existing process (idempotent: counts it as Skipped if present), otherwise
//     creates the process and enqueues the order.created event with the SAME
//     idempotency key live creation uses. A per-order failure is counted in Errors
//     and does not abort the pass.
//
// Resumability: because the eligibility query selects only orders that still lack a
// process, re-running drains the rest and a fully-backfilled tenant returns a report
// with NeedingProcess == 0 (worker idle). No backfill ledger is needed.
func (s *FulfillmentBackfillService) BackfillActiveOrderProcesses(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, opts BackfillOptions) (BackfillReport, error) {
	batch := opts.BatchSize
	if batch <= 0 {
		batch = defaultBackfillBatchSize
	}

	var report BackfillReport
	orderIDs, err := s.fulfillment.ListOrderIDsMissingProcess(ctx, tx, model.TerminalOrderStatuses, batch)
	if err != nil {
		return report, fmt.Errorf("list orders missing process: %w", err)
	}
	report.Scanned = len(orderIDs)

	for _, orderID := range orderIDs {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		report.NeedingProcess++
		if opts.DryRun {
			continue
		}
		created, err := s.ensureProcess(ctx, tx, tenantID, orderID)
		switch {
		case err != nil:
			report.Errors++
		case created:
			report.Created++
		default:
			report.Skipped++
		}
	}
	return report, nil
}

// ensureProcess mirrors FulfillmentService.EnsureProcessForOrder for a single
// order, but UNCONDITIONALLY (no enabled gate). It returns created=true when it
// inserted a new process, created=false when one already existed (idempotent
// no-op). The CreateProcess + EnqueueEvent pair runs inside the caller's tenant tx,
// so it commits atomically with the rest of the batch.
func (s *FulfillmentBackfillService) ensureProcess(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID) (bool, error) {
	// Idempotent: never create a second process for an order that already has one.
	if _, err := s.fulfillment.GetProcessByOrder(ctx, tx, orderID); err == nil {
		return false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("lookup fulfillment process for order: %w", err)
	}

	proc, err := s.fulfillment.CreateProcess(ctx, tx, model.FulfillmentProcess{
		TenantID: tenantID,
		OrderID:  orderID,
	})
	if err != nil {
		return false, fmt.Errorf("create fulfillment process: %w", err)
	}

	// Same event + idempotency key as live creation (EnsureProcessForOrder), so the
	// outbox dedups even if a live order.created event was somehow already enqueued.
	if _, _, err := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
		TenantID:       tenantID,
		ProcessID:      proc.ID,
		EventType:      EventOrderCreated,
		IdempotencyKey: EventOrderCreated + ":" + orderID.String(),
		Payload:        map[string]string{"order_id": orderID.String()},
	}); err != nil {
		return false, fmt.Errorf("enqueue order.created event: %w", err)
	}
	return true, nil
}
