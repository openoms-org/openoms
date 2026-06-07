package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// FulfillmentReadService is the OPE-419 tenant-safe operations/fulfillment READ API.
//
// It exposes a read-only (plus two operator action) view over
// the canonical fulfillment model recorded by OPE-414/415/417/418
// (fulfillment_processes/units/steps/blockers, fulfillment_provider_attempts,
// orchestration_outbox/attempts). EVERY method runs inside
// database.WithTenant(ctx, pool, tenantID, ...) so PostgreSQL Row-Level Security
// enforces isolation — a tenant can never read or mutate another tenant's rows.
//
// Unlike FulfillmentService (the gated, best-effort RECORDING service), this
// service is always active: the read endpoints are registered unconditionally and
// simply return empty results when no fulfillment data exists (which is the case
// while FULFILLMENT_PROCESS_ENABLED is off and nothing is being recorded).
type FulfillmentReadService struct {
	pool          *pgxpool.Pool
	fulfillment   *repository.FulfillmentRepository
	attempts      *repository.FulfillmentAttemptRepository
	orchestration *repository.OrchestrationRepository
}

// NewFulfillmentReadService creates a FulfillmentReadService over the app
// (RLS-scoped) pool and the existing tenant-scoped repositories.
func NewFulfillmentReadService(
	pool *pgxpool.Pool,
	fulfillment *repository.FulfillmentRepository,
	attempts *repository.FulfillmentAttemptRepository,
	orchestration *repository.OrchestrationRepository,
) *FulfillmentReadService {
	return &FulfillmentReadService{
		pool:          pool,
		fulfillment:   fulfillment,
		attempts:      attempts,
		orchestration: orchestration,
	}
}

// ErrFulfillmentNotFound is returned when a requested process/blocker/step does not
// exist for the tenant (either truly absent, or hidden by RLS — the two are
// indistinguishable to the tenant, which is the desired behavior).
var ErrFulfillmentNotFound = errors.New("fulfillment resource not found")

// ---- DTOs ----

// ProcessListResult is a paginated list of processes plus the total match count.
type ProcessListResult struct {
	Items []model.FulfillmentProcess `json:"items"`
	Total int                        `json:"total"`
}

// ProcessDetail assembles a process with its units, steps, blockers and provider
// attempts into a single operator-facing response.
type ProcessDetail struct {
	Process  model.FulfillmentProcess   `json:"process"`
	Units    []UnitDetail               `json:"units"`
	Blockers []model.FulfillmentBlocker `json:"blockers"`
	Attempts []model.ProviderAttempt    `json:"provider_attempts"`
}

// UnitDetail is a fulfillment unit together with its ordered steps.
type UnitDetail struct {
	Unit  model.FulfillmentUnit   `json:"unit"`
	Steps []model.FulfillmentStep `json:"steps"`
}

// OperatorBucket names a top-level operations queue. A process lands in exactly one
// bucket (see classifyProcessBucket for the mapping rules).
type OperatorBucket string

const (
	// BucketReady — process is ready to be worked, no health concerns.
	BucketReady OperatorBucket = "ready"
	// BucketProcessing — actively in progress and healthy.
	BucketProcessing OperatorBucket = "processing"
	// BucketStuck — system error / failed health while still progressing (needs ops attention).
	BucketStuck OperatorBucket = "stuck"
	// BucketBlocked — a typed blocker is holding the process (operator/supplier action).
	BucketBlocked OperatorBucket = "blocked"
	// BucketProviderIssue — waiting on an external provider, or an integration capability blocker.
	BucketProviderIssue OperatorBucket = "provider_issue"
	// BucketMissingData — a mapping/data blocker requires operator-supplied input.
	BucketMissingData OperatorBucket = "missing_data"
)

// OperationsSummaryResult is the operator control-tower count per bucket.
type OperationsSummaryResult struct {
	Buckets map[OperatorBucket]int `json:"buckets"`
	Total   int                    `json:"total"`
}

// ExceptionItem is one actionable process in the exceptions feed, carrying its top
// open blocker (when any) for quick triage.
type ExceptionItem struct {
	Process    model.FulfillmentProcess  `json:"process"`
	Bucket     OperatorBucket            `json:"bucket"`
	TopBlocker *model.FulfillmentBlocker `json:"top_blocker,omitempty"`
}

// IntegrationCapabilitySummaryResult reports automation coverage derived from the
// recorded fulfillment data. It is computed from blocker categories + provider
// attempt outcomes (the inputs available to a tenant-scoped read). Sub-metrics that
// would require cross-referencing the provider Studio capability matrix or supplier
// mappings (which are platform/owner data, not part of the recorded process data)
// are intentionally DEFERRED — see notes in the field comments.
type IntegrationCapabilitySummaryResult struct {
	// AutomationCoverage is the share (0..1) of processes with NO capability/mapping
	// blocker — i.e. that could proceed without an unsupported-capability stop.
	AutomationCoverage float64 `json:"automation_coverage"`
	// ManualInterventionProcesses counts processes carrying an operator-action blocker.
	ManualInterventionProcesses int `json:"manual_intervention_processes"`
	// UnsupportedCapabilityProcesses counts processes blocked by a missing/degraded
	// integration capability.
	UnsupportedCapabilityProcesses int `json:"unsupported_capability_processes"`
	// StaleDataProcesses counts processes blocked by stale channel/supplier data.
	StaleDataProcesses int `json:"stale_data_processes"`
	// MissingMappingProcesses counts processes blocked by an unmapped external status.
	MissingMappingProcesses int `json:"missing_mapping_processes"`
	// FailedProviderAttempts counts recorded provider attempts in a failed state.
	FailedProviderAttempts int `json:"failed_provider_attempts"`
	// TotalProcesses is the denominator used for AutomationCoverage.
	TotalProcesses int `json:"total_processes"`
}

// ---- Reads ----

// ListProcesses returns a paginated, optionally status/health-filtered list of the
// tenant's fulfillment processes (RLS-scoped). Invalid status filter values are
// ignored (they would match nothing); an empty filter returns the newest processes.
func (s *FulfillmentReadService) ListProcesses(ctx context.Context, tenantID uuid.UUID, aggregateStatuses, healthStatuses []string, limit, offset int) (ProcessListResult, error) {
	filter := repository.ProcessListFilter{
		AggregateStatuses: keepValid(aggregateStatuses, model.IsValidAggregateStatus),
		HealthStatuses:    keepValid(healthStatuses, model.IsValidHealthStatus),
		Limit:             limit,
		Offset:            offset,
	}
	var out ProcessListResult
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		items, total, e := s.fulfillment.ListProcessesFiltered(ctx, tx, filter)
		if e != nil {
			return e
		}
		out.Items = items
		out.Total = total
		return nil
	})
	if err != nil {
		return ProcessListResult{}, fmt.Errorf("list processes: %w", err)
	}
	return out, nil
}

// GetProcessDetail returns a process with its units (+ steps), blockers and provider
// attempts assembled into one response (RLS-scoped). Returns ErrFulfillmentNotFound
// when the process does not exist for the tenant.
func (s *FulfillmentReadService) GetProcessDetail(ctx context.Context, tenantID, processID uuid.UUID) (*ProcessDetail, error) {
	var detail *ProcessDetail
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		proc, e := s.fulfillment.GetProcess(ctx, tx, processID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrFulfillmentNotFound
		}
		if e != nil {
			return e
		}
		d, e := s.assembleDetail(ctx, tx, proc)
		if e != nil {
			return e
		}
		detail = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// GetOrderFulfillment returns the assembled fulfillment detail for an order
// (RLS-scoped). Returns ErrFulfillmentNotFound when the order has no process.
func (s *FulfillmentReadService) GetOrderFulfillment(ctx context.Context, tenantID, orderID uuid.UUID) (*ProcessDetail, error) {
	var detail *ProcessDetail
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		proc, e := s.fulfillment.GetProcessByOrder(ctx, tx, orderID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrFulfillmentNotFound
		}
		if e != nil {
			return e
		}
		d, e := s.assembleDetail(ctx, tx, proc)
		if e != nil {
			return e
		}
		detail = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// assembleDetail loads a process's units (+ steps), blockers and provider attempts
// inside an already-tenant-scoped tx.
func (s *FulfillmentReadService) assembleDetail(ctx context.Context, tx pgx.Tx, proc *model.FulfillmentProcess) (*ProcessDetail, error) {
	units, err := s.fulfillment.ListUnits(ctx, tx, proc.ID)
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}
	unitDetails := make([]UnitDetail, 0, len(units))
	for i := range units {
		steps, err := s.fulfillment.ListSteps(ctx, tx, units[i].ID)
		if err != nil {
			return nil, fmt.Errorf("list steps: %w", err)
		}
		unitDetails = append(unitDetails, UnitDetail{Unit: units[i], Steps: steps})
	}

	blockers, err := s.fulfillment.ListBlockers(ctx, tx, proc.ID)
	if err != nil {
		return nil, fmt.Errorf("list blockers: %w", err)
	}

	var attempts []model.ProviderAttempt
	if s.attempts != nil {
		attempts, err = s.attempts.ListByProcess(ctx, tx, proc.ID)
		if err != nil {
			return nil, fmt.Errorf("list provider attempts: %w", err)
		}
	} else {
		attempts = []model.ProviderAttempt{}
	}

	return &ProcessDetail{
		Process:  *proc,
		Units:    unitDetails,
		Blockers: blockers,
		Attempts: attempts,
	}, nil
}

// OperationsSummary returns the operator control-tower counts (one bucket per
// process), derived from each process's aggregate_status + health_status (RLS-scoped).
//
// Mapping (a process is placed in exactly the first matching bucket):
//   - aggregate=blocked                              -> blocked
//   - aggregate=waiting_external                     -> provider_issue
//   - health=system_error (and not already above)    -> stuck
//   - aggregate=in_progress (healthy)                -> processing
//   - aggregate in {new,validating,ready}            -> ready
//   - aggregate in {completed,cancelled}             -> (terminal; not counted)
//
// The missing_data bucket cannot be derived from the (status,health) counts alone —
// it depends on the blocker CATEGORY — so it is surfaced from the blocked set in
// OperationsExceptions, and the summary's blocked bucket carries blocked processes
// regardless of blocker category. Terminal processes are excluded from the buckets
// but included in Total for context.
func (s *FulfillmentReadService) OperationsSummary(ctx context.Context, tenantID uuid.UUID) (OperationsSummaryResult, error) {
	out := OperationsSummaryResult{Buckets: map[OperatorBucket]int{
		BucketReady: 0, BucketProcessing: 0, BucketStuck: 0,
		BucketBlocked: 0, BucketProviderIssue: 0, BucketMissingData: 0,
	}}
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		counts, e := s.fulfillment.CountProcessesByStatus(ctx, tx)
		if e != nil {
			return e
		}
		for _, c := range counts {
			out.Total += c.Count
			if b, ok := classifyProcessBucket(c.AggregateStatus, c.HealthStatus); ok {
				out.Buckets[b] += c.Count
			}
		}
		return nil
	})
	if err != nil {
		return OperationsSummaryResult{}, fmt.Errorf("operations summary: %w", err)
	}
	return out, nil
}

// classifyProcessBucket maps a process's (aggregate_status, health_status) to its
// operator bucket. The bool is false for terminal/uncounted processes. It is a PURE
// function so it can be unit-tested without a DB. missing_data is refined per-blocker
// in OperationsExceptions (it cannot be told apart from blocked here).
func classifyProcessBucket(aggregate, health string) (OperatorBucket, bool) {
	switch aggregate {
	case model.ProcessStatusBlocked:
		return BucketBlocked, true
	case model.ProcessStatusWaitingExternal:
		return BucketProviderIssue, true
	case model.ProcessStatusCompleted, model.ProcessStatusCancelled:
		return "", false
	}
	if health == model.ProcessHealthSystemError {
		return BucketStuck, true
	}
	switch aggregate {
	case model.ProcessStatusInProgress:
		return BucketProcessing, true
	case model.ProcessStatusNew, model.ProcessStatusValidating, model.ProcessStatusReady:
		return BucketReady, true
	}
	return "", false
}

// OperationsExceptions returns the actionable processes — blocked, waiting_external,
// or unhealthy/stuck — each annotated with its bucket and top open blocker
// (RLS-scoped). limit caps the number of processes scanned (default 100). For
// blocked processes the bucket is refined from the top blocker's category:
// capability -> provider_issue, mapping -> missing_data, otherwise -> blocked.
func (s *FulfillmentReadService) OperationsExceptions(ctx context.Context, tenantID uuid.UUID, limit int) ([]ExceptionItem, error) {
	limit = clampScan(limit, 100, 500)
	statuses := []string{
		model.ProcessStatusBlocked,
		model.ProcessStatusWaitingExternal,
		model.ProcessStatusInProgress, // may be "stuck" when health=system_error
	}
	var items []ExceptionItem
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		procs, e := s.fulfillment.ListProcessesByAggregateStatuses(ctx, tx, statuses, limit)
		if e != nil {
			return e
		}
		items = make([]ExceptionItem, 0, len(procs))
		for i := range procs {
			p := procs[i]
			// in_progress is only an exception when unhealthy (stuck); skip the rest.
			if p.AggregateStatus == model.ProcessStatusInProgress && p.HealthStatus != model.ProcessHealthSystemError {
				continue
			}
			open, e := s.fulfillment.ListOpenBlockersByProcess(ctx, tx, p.ID)
			if e != nil {
				return e
			}
			var top *model.FulfillmentBlocker
			if len(open) > 0 {
				top = &open[0]
			}
			items = append(items, ExceptionItem{
				Process:    p,
				Bucket:     exceptionBucket(p, top),
				TopBlocker: top,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("operations exceptions: %w", err)
	}
	return items, nil
}

// exceptionBucket refines an exception process's bucket using its top open blocker
// category when available. Pure helper.
func exceptionBucket(p model.FulfillmentProcess, top *model.FulfillmentBlocker) OperatorBucket {
	if p.AggregateStatus == model.ProcessStatusInProgress {
		return BucketStuck
	}
	if p.AggregateStatus == model.ProcessStatusWaitingExternal {
		return BucketProviderIssue
	}
	// Blocked: refine by category.
	if top != nil {
		switch top.Category {
		case "capability":
			return BucketProviderIssue
		case "mapping":
			return BucketMissingData
		case "operator":
			return BucketMissingData
		}
	}
	return BucketBlocked
}

// IntegrationCapabilitySummary derives automation-coverage metrics for the tenant
// from the recorded blockers + provider attempts (RLS-scoped). See the result type
// for which sub-metrics are DEFERRED (those needing the provider Studio capability
// matrix / supplier mappings, which are not part of the recorded process data).
func (s *FulfillmentReadService) IntegrationCapabilitySummary(ctx context.Context, tenantID uuid.UUID, scanLimit int) (IntegrationCapabilitySummaryResult, error) {
	var out IntegrationCapabilitySummaryResult
	scanLimit = clampScan(scanLimit, 1000, 5000)
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Total processes (for the coverage denominator).
		counts, e := s.fulfillment.CountProcessesByStatus(ctx, tx)
		if e != nil {
			return e
		}
		for _, c := range counts {
			out.TotalProcesses += c.Count
		}

		// Scan currently-actionable processes' open blockers to classify capability gaps.
		statuses := []string{model.ProcessStatusBlocked, model.ProcessStatusWaitingExternal}
		procs, e := s.fulfillment.ListProcessesByAggregateStatuses(ctx, tx, statuses, scanLimit)
		if e != nil {
			return e
		}
		blockedWithCapGap := 0
		for i := range procs {
			open, e := s.fulfillment.ListOpenBlockersByProcess(ctx, tx, procs[i].ID)
			if e != nil {
				return e
			}
			// Counters are PER-PROCESS: a process with several blockers of the same
			// kind is counted once (the result fields are named "...Processes").
			var hasCap, hasMapping, hasOperator, hasStale bool
			for j := range open {
				switch open[j].Category {
				case "capability":
					hasCap = true
				case "mapping":
					hasMapping = true
				case "operator":
					hasOperator = true
				case "supplier", "integration":
					// stale data shows up via specific codes below.
				}
				switch open[j].Code {
				case model.BlockerChannelStockStale, model.BlockerSupplierAvailabilityStale:
					hasStale = true
				}
			}
			if hasCap {
				out.UnsupportedCapabilityProcesses++
			}
			if hasMapping {
				out.MissingMappingProcesses++
			}
			if hasOperator {
				out.ManualInterventionProcesses++
			}
			if hasStale {
				out.StaleDataProcesses++
			}
			if hasCap || hasMapping {
				blockedWithCapGap++
			}
		}

		if out.TotalProcesses > 0 {
			out.AutomationCoverage = float64(out.TotalProcesses-blockedWithCapGap) / float64(out.TotalProcesses)
		}
		return nil
	})
	if err != nil {
		return IntegrationCapabilitySummaryResult{}, fmt.Errorf("integration capability summary: %w", err)
	}
	return out, nil
}

// ---- Actions ----

// ResolveBlocker marks a blocker resolved (status=resolved, resolved_at=now) for the
// tenant (RLS-scoped write). Returns ErrFulfillmentNotFound when the blocker does not
// exist for the tenant.
func (s *FulfillmentReadService) ResolveBlocker(ctx context.Context, tenantID, blockerID uuid.UUID) (*model.FulfillmentBlocker, error) {
	var out *model.FulfillmentBlocker
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, e := s.fulfillment.GetBlocker(ctx, tx, blockerID); errors.Is(e, pgx.ErrNoRows) {
			return ErrFulfillmentNotFound
		} else if e != nil {
			return e
		}
		b, e := s.fulfillment.ResolveBlocker(ctx, tx, blockerID, model.BlockerStatusResolved)
		if e != nil {
			return e
		}
		out = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RetryStep resets a failed/blocked step toward pending for re-execution (RLS-scoped
// write) and, when an orchestration outbox is wired, enqueues an idempotent
// fulfillment.step retry event so the orchestration worker re-attempts the side
// effect. The step's attempts counter is preserved (the worker increments on its
// next run). Returns ErrFulfillmentNotFound when the step does not exist for the
// tenant, and a ValidationError when the step is not in a retryable state.
//
// NOTE: actual re-execution depends on the orchestration worker (OPE-415) consuming
// the enqueued event. When no outbox is wired, the step state is reset and the
// worker dependency is recorded in the returned step's metadata-free response — the
// caller (and operator) is responsible for re-driving it.
func (s *FulfillmentReadService) RetryStep(ctx context.Context, tenantID, stepID uuid.UUID) (*model.FulfillmentStep, error) {
	var out *model.FulfillmentStep
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		step, e := s.fulfillment.GetStep(ctx, tx, stepID)
		if errors.Is(e, pgx.ErrNoRows) {
			return ErrFulfillmentNotFound
		}
		if e != nil {
			return e
		}
		if step.Status != model.FulfillmentStatusFailed && step.Status != model.FulfillmentStatusBlocked {
			return NewValidationError(fmt.Errorf("step in status %q is not retryable (must be failed or blocked)", step.Status))
		}

		// Reset the step toward pending; keep the attempts count so history is preserved.
		reset, e := s.fulfillment.UpdateStepStatus(ctx, tx, step.ID, model.FulfillmentStatusPending, step.Attempts)
		if e != nil {
			return e
		}

		// Best-effort idempotent re-attempt enqueue (worker consumes it, OPE-415).
		if s.orchestration != nil {
			unit, e := s.fulfillment.GetUnit(ctx, tx, step.UnitID)
			if e != nil {
				return fmt.Errorf("lookup unit for step retry: %w", e)
			}
			unitID := unit.ID
			idem := EventFulfillmentStep + ":retry:" + step.ID.String() + ":" + fmt.Sprint(step.Attempts)
			if _, _, e := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
				TenantID:       tenantID,
				ProcessID:      unit.ProcessID,
				UnitID:         &unitID,
				EventType:      EventFulfillmentStep,
				IdempotencyKey: idem,
				Payload: map[string]any{
					"step_key": step.StepKey,
					"step_id":  step.ID.String(),
					"unit_id":  step.UnitID.String(),
					"reason":   "operator_retry",
				},
			}); e != nil {
				return fmt.Errorf("enqueue step retry event: %w", e)
			}
		}

		out = reset
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// clampScan bounds a caller-supplied scan/limit param: non-positive falls back to
// def, and anything above upper is capped at upper (so a client cannot request an
// unbounded scan).
func clampScan(v, def, upper int) int {
	if v <= 0 {
		return def
	}
	if v > upper {
		return upper
	}
	return v
}

// keepValid returns the subset of values for which valid(v) is true. Used to drop
// unknown status filter values before they reach the repository.
func keepValid(values []string, valid func(string) bool) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if valid(v) {
			out = append(out, v)
		}
	}
	return out
}
