package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// DefaultParityCoverageThreshold is the default process-coverage bar the rollout
// parity gate uses to decide it is safe to cut the operations dashboard over from
// the LEGACY (order/shipment-status-derived) heuristic to the PROCESS-BACKED read
// model (OPE-419). At >= 0.99, essentially every non-terminal order is backed by a
// fulfillment process, so the process-backed counts can stand in for the legacy
// ones without losing visibility on in-flight work. Operators can override it per
// request (e.g. demand a full 1.0 before cutover); see ParityReport for the verdict.
const DefaultParityCoverageThreshold = 0.99

// legacyProblemShipmentStatuses are the shipment statuses the LEGACY operations
// dashboard treats as a "problem" shipment. The dashboard exception feed filters on
// status == "failed" (buildShipmentException) and the integration-health view treats
// "error" as a problem; shipment.status is free-text (no DB CHECK), provider-driven,
// so we match BOTH here to stay comparable to what an operator sees today. This is
// the only legacy heuristic input that is honestly approximate — documented as such.
var legacyProblemShipmentStatuses = []string{"failed", "error"}

// ParityReport compares the LEGACY operational counts (derived from order +
// shipment status, the way the current dashboard does it) against the PROCESS-BACKED
// counts (the OPE-419 read model) for a single tenant. It is a READ-ONLY decision
// aid for the rollout parity gate: when ProcessCoverageMet is true, the process-
// backed read model covers essentially all in-flight orders, so the dashboard can be
// safely cut over from the heuristic to process-backed counts (OPE-423c).
//
// LEGACY vs PROCESS-BACKED mapping (kept explicit because this is a decision aid —
// some fields are directly comparable, some are deliberately approximate):
//
//   - NonTerminalOrders  (LEGACY active population) — orders whose status is NOT
//     terminal (completed/cancelled/refunded). This is the population the legacy
//     dashboard spreads across its intake/fulfillment/shipping stages.
//   - FulfillmentProcesses (PROCESS-BACKED population) — total fulfillment processes
//     recorded for the tenant (one per backfilled/created order).
//   - OrdersMissingProcess — non-terminal orders with NO process yet. This is the
//     coverage GAP and must trend to 0 after the OPE-423a backfill. It is the
//     authoritative parity signal (computed directly, not inferred from a diff).
//   - ProcessCoverage — (NonTerminalOrders - OrdersMissingProcess) / NonTerminalOrders,
//     clamped to [0,1]; 1.0 when there are no non-terminal orders (vacuously covered).
//   - LegacyProblemOrders — the legacy "needs attention" grouping computed on the
//     backend the SAME way the dashboard derives it: non-terminal orders that are
//     on_hold OR have a problem (failed/error) shipment. CAVEAT: the dashboard also
//     shows integration-error exceptions, which are NOT order-scoped, so they are
//     excluded here — this is the order-comparable subset.
//   - ProcessBackedExceptions — the process-backed "needs attention" total: the
//     blocked + provider_issue + stuck operator buckets from the OPE-419 operations
//     summary. NOTE: LegacyProblemOrders and ProcessBackedExceptions are counted on
//     DIFFERENT primitives (orders vs processes) and via different signals, so they
//     are NOT expected to be equal even at full coverage — they are surfaced side by
//     side as a sanity check on order of magnitude, not as a strict equality gate.
//
// The ONLY gate-relevant verdict is ProcessCoverageMet (driven by OrdersMissingProcess
// / ProcessCoverage). The two exception counts are advisory context.
type ParityReport struct {
	// NonTerminalOrders is the legacy active order population (coverage denominator).
	NonTerminalOrders int `json:"non_terminal_orders"`
	// FulfillmentProcesses is the total process-backed population for the tenant.
	FulfillmentProcesses int `json:"fulfillment_processes"`
	// OrdersMissingProcess is the count of non-terminal orders with no process yet
	// (the coverage gap; must trend to 0 after backfill).
	OrdersMissingProcess int `json:"orders_missing_process"`
	// ProcessCoverage is the share (0..1) of non-terminal orders that ARE backed by a
	// process. 1.0 when there are no non-terminal orders.
	ProcessCoverage float64 `json:"process_coverage"`
	// LegacyProblemOrders is the legacy "needs attention" order count (on_hold or
	// problem-shipment), the order-comparable subset of the dashboard heuristic.
	LegacyProblemOrders int `json:"legacy_problem_orders"`
	// ProcessBackedExceptions is the process-backed "needs attention" total
	// (blocked + provider_issue + stuck buckets).
	ProcessBackedExceptions int `json:"process_backed_exceptions"`
	// CoverageThreshold is the threshold ProcessCoverageMet was evaluated against.
	CoverageThreshold float64 `json:"coverage_threshold"`
	// ProcessCoverageMet is the parity verdict: true when ProcessCoverage >=
	// CoverageThreshold (i.e. it is safe to cut the dashboard over to process-backed
	// counts). This is the only gate-relevant boolean in the report.
	ProcessCoverageMet bool `json:"process_coverage_met"`
}

// parityInputs is the raw, DB-sourced set of counts the parity math is derived from.
// Splitting it out keeps computeParity a PURE function (no DB), so the coverage /
// verdict / edge-case logic is unit-tested without a database.
type parityInputs struct {
	nonTerminalOrders    int
	fulfillmentProcesses int
	ordersMissingProcess int
	legacyProblemOrders  int
	processedExceptions  int
}

// ParityReport builds the tenant-scoped legacy-vs-process parity report (RLS-scoped:
// every read runs inside database.WithTenant so a tenant can never count another
// tenant's orders or processes). It is READ-ONLY — it writes NOTHING. threshold is
// the caller-supplied coverage bar; a non-positive value falls back to
// DefaultParityCoverageThreshold and any value above 1 is clamped to 1.
func (s *FulfillmentReadService) ParityReport(ctx context.Context, tenantID uuid.UUID, threshold float64) (ParityReport, error) {
	var in parityInputs
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var e error
		if in.nonTerminalOrders, e = s.fulfillment.CountNonTerminalOrders(ctx, tx, model.TerminalOrderStatuses); e != nil {
			return e
		}
		if in.ordersMissingProcess, e = s.fulfillment.CountOrdersMissingProcess(ctx, tx, model.TerminalOrderStatuses); e != nil {
			return e
		}
		if in.legacyProblemOrders, e = s.fulfillment.CountLegacyProblemOrders(ctx, tx, model.TerminalOrderStatuses, legacyProblemShipmentStatuses); e != nil {
			return e
		}

		// Process-backed population + exceptions, derived from the OPE-419 status counts
		// so the parity report and the operations summary use the SAME source of truth.
		counts, e := s.fulfillment.CountProcessesByStatus(ctx, tx)
		if e != nil {
			return e
		}
		for _, c := range counts {
			in.fulfillmentProcesses += c.Count
			if isProcessBackedException(c.AggregateStatus, c.HealthStatus) {
				in.processedExceptions += c.Count
			}
		}
		return nil
	})
	if err != nil {
		return ParityReport{}, fmt.Errorf("parity report: %w", err)
	}
	return computeParity(in, threshold), nil
}

// isProcessBackedException reports whether a process in (aggregate, health) lands in
// one of the "needs attention" operator buckets (blocked / provider_issue / stuck).
// It reuses classifyProcessBucket so the parity exception total stays consistent with
// the OPE-419 operations summary buckets. Pure helper.
func isProcessBackedException(aggregate, health string) bool {
	b, ok := classifyProcessBucket(aggregate, health)
	if !ok {
		return false
	}
	switch b {
	case BucketBlocked, BucketProviderIssue, BucketStuck:
		return true
	default:
		return false
	}
}

// computeParity turns the raw DB-sourced counts into the report: it normalises the
// threshold, computes coverage and the verdict. PURE (no DB) so all the math +
// edge cases (zero orders, threshold clamping, full/partial coverage) are unit-
// tested without a database.
//
// Coverage rule: with zero non-terminal orders coverage is 1.0 (there is nothing to
// cover, so the gate is vacuously met) — this avoids a 0/0 that would otherwise
// block cutover on an empty/quiet tenant.
func computeParity(in parityInputs, threshold float64) ParityReport {
	threshold = normaliseThreshold(threshold)

	coverage := 1.0
	if in.nonTerminalOrders > 0 {
		// Defensive: missing can never exceed the population, but never report < 0.
		covered := max(in.nonTerminalOrders-in.ordersMissingProcess, 0)
		coverage = float64(covered) / float64(in.nonTerminalOrders)
	}

	return ParityReport{
		NonTerminalOrders:       in.nonTerminalOrders,
		FulfillmentProcesses:    in.fulfillmentProcesses,
		OrdersMissingProcess:    in.ordersMissingProcess,
		ProcessCoverage:         coverage,
		LegacyProblemOrders:     in.legacyProblemOrders,
		ProcessBackedExceptions: in.processedExceptions,
		CoverageThreshold:       threshold,
		ProcessCoverageMet:      coverage >= threshold,
	}
}

// normaliseThreshold bounds a caller-supplied coverage threshold: non-positive falls
// back to the default, anything above 1 is clamped to 1 (you can never require more
// than full coverage).
func normaliseThreshold(threshold float64) float64 {
	if threshold <= 0 {
		return DefaultParityCoverageThreshold
	}
	if threshold > 1 {
		return 1
	}
	return threshold
}
