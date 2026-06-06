package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// OPE-418: fulfillment UNITS + STEPS + typed supplier blockers.
//
// This file extends FulfillmentService (defined in fulfillment_service.go) with
// gated, BEST-EFFORT recording of the fulfillment *units* that make up an order's
// fulfillment process (warehouse, dropship, backorder, supplier purchase) and the
// canonical *steps* within them. Every method here:
//
//   - early-returns when the service is not enabled+wired (recordingReady());
//   - opens its OWN database.WithTenant transaction (never reuses the caller's);
//   - logs-and-continues on ANY error, never propagating it to the caller.
//
// As a result these methods are ADDITIVE and cannot break, block, or roll back the
// primary pick-pack / dropship / purchase-order / warehouse-document operation they
// are hooked into. When FULFILLMENT_PROCESS_ENABLED is OFF every method is a
// complete no-op, so the existing order-status flow is byte-for-byte unchanged.
//
// The data model (fulfillment_processes/units/steps/blockers) is the OPE-414
// tenant-scoped, RLS-enforced schema (migration 000036); no new tables are added.

// SupplierFulfillmentCapability classifies how a dropship/supplier order can be
// transmitted to the supplier, derived from the supplier's existing configuration
// (feed format + linked integration). It governs whether the dropship unit can be
// auto-submitted via API or requires a manual/portal fallback step, and whether a
// typed capability blocker is raised. It is intentionally coarse: the full
// supplier-availability POLICY model (quantities, freshness, lead time, preflight)
// is deferred to an OPE-418 follow-up.
type SupplierFulfillmentCapability string

const (
	// SupplierCapAPI means the supplier has a linked integration with a registered
	// provider — dropship orders can be submitted programmatically.
	SupplierCapAPI SupplierFulfillmentCapability = "api"
	// SupplierCapPortal means the supplier has portal access enabled but no API
	// provider — orders are handed off via the supplier portal (manual step).
	SupplierCapPortal SupplierFulfillmentCapability = "portal"
	// SupplierCapManual means no integration and no portal — the operator must place
	// the order with the supplier out-of-band (manual fallback step).
	SupplierCapManual SupplierFulfillmentCapability = "manual"
	// SupplierCapUnsupported means the supplier is configured with an integration
	// whose format has no registered provider — a capability blocker is raised.
	SupplierCapUnsupported SupplierFulfillmentCapability = "unsupported"
)

// ClassifySupplierCapability derives a supplier's fulfillment capability from its
// existing configuration. It is a pure helper (no I/O): API when a linked
// integration resolves to a registered provider; Unsupported when an integration
// is linked but its format has no provider; Portal when portal access is enabled;
// otherwise Manual. providerRegistered reports whether a supplier provider exists
// for the supplier's (normalized) feed format — callers pass
// integration.HasSupplierProvider so this helper stays dependency-free.
func ClassifySupplierCapability(s *model.Supplier, providerRegistered bool) SupplierFulfillmentCapability {
	if s == nil {
		return SupplierCapManual
	}
	if s.IntegrationID != nil {
		if providerRegistered {
			return SupplierCapAPI
		}
		return SupplierCapUnsupported
	}
	if s.PortalEnabled {
		return SupplierCapPortal
	}
	return SupplierCapManual
}

// EnsureUnit ensures a fulfillment unit of the given type exists on the order's
// process, returning the existing one when present (idempotent per
// (process, unit_type, key) where key is taken from metadata["key"] when set).
// BEST-EFFORT: no-op when disabled/unwired or when no process exists; runs its own
// tenant transaction; logs-and-continues on error. Returns nil when nothing was
// recorded.
func (s *FulfillmentService) EnsureUnit(ctx context.Context, tenantID, orderID uuid.UUID, unitType, dedupeKey string, metadata map[string]any) *model.FulfillmentUnit {
	if !s.recordingReady() {
		return nil
	}
	if !model.IsValidUnitType(unitType) {
		slog.Warn("fulfillment: ensure unit skipped, unknown unit type (best-effort)", "unit_type", unitType)
		return nil
	}

	var unit *model.FulfillmentUnit
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		proc, perr := s.fulfillment.GetProcessByOrder(ctx, tx, orderID)
		if errors.Is(perr, pgx.ErrNoRows) {
			return nil // no process — nothing to attach the unit to
		}
		if perr != nil {
			return fmt.Errorf("lookup process for unit: %w", perr)
		}

		existing, lerr := s.fulfillment.ListUnits(ctx, tx, proc.ID)
		if lerr != nil {
			return fmt.Errorf("list units: %w", lerr)
		}
		for i := range existing {
			if existing[i].UnitType != unitType {
				continue
			}
			if dedupeKey == "" || unitMetaKey(existing[i].Metadata) == dedupeKey {
				unit = &existing[i]
				return nil
			}
		}

		meta := cloneMeta(metadata)
		if dedupeKey != "" {
			meta["key"] = dedupeKey
		}
		out, cerr := s.fulfillment.CreateUnit(ctx, tx, model.FulfillmentUnit{
			TenantID:  tenantID,
			ProcessID: proc.ID,
			UnitType:  unitType,
			Metadata:  meta,
		})
		if cerr != nil {
			return fmt.Errorf("create unit: %w", cerr)
		}
		unit = out
		return nil
	})
	if err != nil {
		slog.Warn("fulfillment: ensure unit failed (best-effort, ignored)",
			"tenant_id", tenantID, "order_id", orderID, "unit_type", unitType, "error", err)
		return nil
	}
	return unit
}

// RecordUnitTransition updates a unit's status (BEST-EFFORT, own tenant tx,
// log-and-continue). Invalid statuses are skipped. unitID of uuid.Nil is a no-op.
func (s *FulfillmentService) RecordUnitTransition(ctx context.Context, tenantID, unitID uuid.UUID, status string) {
	if !s.recordingReady() || unitID == uuid.Nil {
		return
	}
	if !model.IsValidFulfillmentStatus(status) {
		slog.Warn("fulfillment: unit transition skipped, unknown status (best-effort)", "status", status)
		return
	}
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		_, uerr := s.fulfillment.UpdateUnitStatus(ctx, tx, unitID, status)
		return uerr
	})
	if err != nil {
		slog.Warn("fulfillment: unit transition failed (best-effort, ignored)",
			"tenant_id", tenantID, "unit_id", unitID, "status", status, "error", err)
	}
}

// RecordStep upserts a canonical step on a unit and sets its status (BEST-EFFORT,
// own tenant tx, log-and-continue). The step is keyed by (unit, step_key): an
// existing step is updated (attempts incremented) rather than duplicated, so this
// is idempotent across repeated calls for the same step. Invalid step keys /
// statuses are skipped. Returns nil when nothing was recorded.
func (s *FulfillmentService) RecordStep(ctx context.Context, tenantID, unitID uuid.UUID, stepKey, status string, metadata map[string]any) *model.FulfillmentStep {
	if !s.recordingReady() || unitID == uuid.Nil {
		return nil
	}
	if !model.IsValidStepKey(stepKey) {
		slog.Warn("fulfillment: record step skipped, unknown step key (best-effort)", "step_key", stepKey)
		return nil
	}
	if !model.IsValidFulfillmentStatus(status) {
		slog.Warn("fulfillment: record step skipped, unknown status (best-effort)", "status", status)
		return nil
	}

	var step *model.FulfillmentStep
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		steps, lerr := s.fulfillment.ListSteps(ctx, tx, unitID)
		if lerr != nil {
			return fmt.Errorf("list steps: %w", lerr)
		}
		for i := range steps {
			if steps[i].StepKey == stepKey {
				out, uerr := s.fulfillment.UpdateStepStatus(ctx, tx, steps[i].ID, status, steps[i].Attempts+1)
				if uerr != nil {
					return fmt.Errorf("update step: %w", uerr)
				}
				step = out
				return nil
			}
		}
		out, cerr := s.fulfillment.CreateStep(ctx, tx, model.FulfillmentStep{
			TenantID: tenantID,
			UnitID:   unitID,
			StepKey:  stepKey,
			Status:   status,
			Attempts: 1,
			Metadata: cloneMeta(metadata),
		})
		if cerr != nil {
			return fmt.Errorf("create step: %w", cerr)
		}
		step = out
		return nil
	})
	if err != nil {
		slog.Warn("fulfillment: record step failed (best-effort, ignored)",
			"tenant_id", tenantID, "unit_id", unitID, "step_key", stepKey, "error", err)
		return nil
	}
	return step
}

// CreateSupplierBlocker creates a typed supplier/capability blocker on an order's
// process (BEST-EFFORT, own tenant tx, log-and-continue). The code MUST be one of
// the ADR-424 blocker codes (model.IsValidBlockerCode); unknown codes are skipped.
// When unitID is non-nil the blocker is attached to that unit. No-op when no
// process exists. Returns nil when nothing was recorded.
func (s *FulfillmentService) CreateSupplierBlocker(ctx context.Context, tenantID, orderID uuid.UUID, unitID *uuid.UUID, code, description string) *model.FulfillmentBlocker {
	if !s.recordingReady() {
		return nil
	}
	if !model.IsValidBlockerCode(code) {
		slog.Warn("fulfillment: create supplier blocker skipped, unknown code (best-effort)", "code", code)
		return nil
	}

	var blocker *model.FulfillmentBlocker
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		proc, perr := s.fulfillment.GetProcessByOrder(ctx, tx, orderID)
		if errors.Is(perr, pgx.ErrNoRows) {
			return nil // no process — nothing to attach the blocker to
		}
		if perr != nil {
			return fmt.Errorf("lookup process for supplier blocker: %w", perr)
		}
		out, berr := s.fulfillment.CreateBlocker(ctx, tx, model.FulfillmentBlocker{
			TenantID:    tenantID,
			ProcessID:   proc.ID,
			UnitID:      unitID,
			Code:        code,
			Description: description,
		})
		if berr != nil {
			return berr
		}
		blocker = out
		return nil
	})
	if err != nil {
		slog.Warn("fulfillment: create supplier blocker failed (best-effort, ignored)",
			"tenant_id", tenantID, "order_id", orderID, "code", code, "error", err)
		return nil
	}
	return blocker
}

// AggregateProcessState derives a process's aggregate status + health from its
// units, following ADR-424 precedence:
//
//   - any unit blocked            -> aggregate=blocked,           health=action_required
//   - any unit failed             -> aggregate=in_progress,       health=system_error
//   - any unit waiting_external   -> aggregate=waiting_external,  health=warning
//   - any unit running            -> aggregate=in_progress,       health=ok
//   - all units cancelled         -> aggregate=cancelled,         health=ok
//   - all units terminal          -> aggregate=completed,         health=ok
//     (succeeded/skipped, with at least one not cancelled)
//   - all units pending/ready     -> aggregate=ready,             health=ok
//   - no units                    -> aggregate=new,               health=ok
//
// It is a PURE function over the unit list so it can be unit-tested without a DB.
func AggregateProcessState(units []model.FulfillmentUnit) (aggregate, health string) {
	if len(units) == 0 {
		return model.ProcessStatusNew, model.ProcessHealthOK
	}

	var blocked, failed, waiting, running, succeeded, cancelled, pendingReady int
	for _, u := range units {
		switch u.Status {
		case model.FulfillmentStatusBlocked:
			blocked++
		case model.FulfillmentStatusFailed:
			failed++
		case model.FulfillmentStatusWaitingExternal:
			waiting++
		case model.FulfillmentStatusRunning:
			running++
		case model.FulfillmentStatusSucceeded, model.FulfillmentStatusSkipped:
			succeeded++
		case model.FulfillmentStatusCancelled:
			cancelled++
		default: // pending, ready
			pendingReady++
		}
	}

	switch {
	case blocked > 0:
		return model.ProcessStatusBlocked, model.ProcessHealthActionRequired
	case failed > 0:
		return model.ProcessStatusInProgress, model.ProcessHealthSystemError
	case waiting > 0:
		return model.ProcessStatusWaitingExternal, model.ProcessHealthWarning
	case running > 0:
		return model.ProcessStatusInProgress, model.ProcessHealthOK
	case cancelled == len(units):
		// Every unit cancelled -> the whole process is cancelled, not completed.
		return model.ProcessStatusCancelled, model.ProcessHealthOK
	case succeeded+cancelled == len(units):
		// All units terminal and at least one succeeded/skipped.
		return model.ProcessStatusCompleted, model.ProcessHealthOK
	default:
		return model.ProcessStatusReady, model.ProcessHealthOK
	}
}

// AggregateMixedOrder recomputes an order's aggregate process state from its units
// and persists it (BEST-EFFORT, own tenant tx, log-and-continue). It is the
// mixed-order helper: after any unit transition a caller may invoke this to keep
// the parent process status consistent with its (possibly mixed warehouse +
// dropship + backorder) units. No-op when no process exists. Returns the updated
// process, or nil.
func (s *FulfillmentService) AggregateMixedOrder(ctx context.Context, tenantID, orderID uuid.UUID) *model.FulfillmentProcess {
	if !s.recordingReady() {
		return nil
	}

	var proc *model.FulfillmentProcess
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		p, perr := s.fulfillment.GetProcessByOrder(ctx, tx, orderID)
		if errors.Is(perr, pgx.ErrNoRows) {
			return nil
		}
		if perr != nil {
			return fmt.Errorf("lookup process for aggregation: %w", perr)
		}
		units, lerr := s.fulfillment.ListUnits(ctx, tx, p.ID)
		if lerr != nil {
			return fmt.Errorf("list units for aggregation: %w", lerr)
		}
		aggregate, health := AggregateProcessState(units)
		updated, uerr := s.fulfillment.UpdateProcessStatus(ctx, tx, p.ID, aggregate, health)
		if uerr != nil {
			return fmt.Errorf("update process status: %w", uerr)
		}
		proc = updated
		return nil
	})
	if err != nil {
		slog.Warn("fulfillment: aggregate mixed order failed (best-effort, ignored)",
			"tenant_id", tenantID, "order_id", orderID, "error", err)
		return nil
	}
	return proc
}

// cloneMeta returns a shallow copy of m, or an empty map when m is nil, so callers
// never mutate the caller-provided map (and we can safely inject keys like "key").
func cloneMeta(m map[string]any) map[string]any {
	out := make(map[string]any, len(m)+1)
	maps.Copy(out, m)
	return out
}

// unitMetaKey extracts the dedupe "key" from a unit's metadata (set by EnsureUnit).
// Returns "" when absent or not a string. Metadata round-trips through JSONB, so it
// is decoded as map[string]any.
func unitMetaKey(meta any) string {
	m, ok := meta.(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := m["key"].(string); ok {
		return v
	}
	return ""
}
