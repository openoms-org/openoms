package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// EventOrderCreated is the orchestration event type emitted when an order's
// fulfillment process is first created. It is the entry point of the fulfillment
// state machine and is consumed by OrderCreatedHandler.
const EventOrderCreated = "order.created"

// EventFulfillmentStep is the orchestration event type emitted when a provider
// side effect maps onto a canonical fulfillment step (OPE-417). It is an
// observability/audit signal carried through the same transactional outbox.
const EventFulfillmentStep = "fulfillment.step"

// FulfillmentService bridges order/shipment side effects to the fulfillment +
// orchestration foundation (OPE-414/415/416/417). On order creation it ensures a
// fulfillment process exists and enqueues the initial orchestration event. From
// OPE-417 it also records provider attempts, emits fulfillment-step events, and
// creates typed blockers when a carrier call fails — all as BEST-EFFORT side
// effects: these methods never propagate an error to their caller, so they cannot
// break or roll back the primary shipment/label/tracking operation. It is gated by
// a config flag and is a complete no-op when disabled, so wiring it in carries
// zero behavior change until enabled.
type FulfillmentService struct {
	enabled       bool
	pool          *pgxpool.Pool
	fulfillment   *repository.FulfillmentRepository
	orchestration *repository.OrchestrationRepository
	attempts      *repository.FulfillmentAttemptRepository
	// metrics is an OPTIONAL, best-effort observability handle (OPE-422). It is
	// nil-receiver safe, so every record call is a no-op when it is not wired and
	// can never affect or slow the primary recording operation. Metrics are emitted
	// only AFTER the underlying DB write succeeds, using already-validated bounded
	// values (operation/status/category) — never an unbounded identifier.
	metrics *obsmetrics.FulfillmentMetrics
}

// NewFulfillmentService creates a FulfillmentService. When enabled is false every
// method is a no-op (returns nil, nil), so callers can wire it unconditionally.
// The pool and attempt repository are optional (nil-safe): the best-effort
// recording methods become no-ops when they are not configured.
func NewFulfillmentService(enabled bool, fulfillment *repository.FulfillmentRepository, orchestration *repository.OrchestrationRepository) *FulfillmentService {
	return &FulfillmentService{
		enabled:       enabled,
		fulfillment:   fulfillment,
		orchestration: orchestration,
	}
}

// WithRecording wires the pool + attempt repository required by the best-effort
// provider-attempt recording methods (OPE-417). Returns the service for chaining.
// Without it, RecordProviderAttempt/EmitFulfillmentStep/CreateCarrierBlocker stay
// no-ops even when the service is enabled.
func (s *FulfillmentService) WithRecording(pool *pgxpool.Pool, attempts *repository.FulfillmentAttemptRepository) *FulfillmentService {
	if s == nil {
		return nil
	}
	s.pool = pool
	s.attempts = attempts
	return s
}

// WithMetrics wires the OPTIONAL fulfillment metrics collector (OPE-422) for
// additive, best-effort observability. Returns the service for chaining. Safe to
// omit: the collector is nil-receiver safe, so every metric call is a no-op when
// it is not wired and can never affect or slow the primary recording operation.
func (s *FulfillmentService) WithMetrics(m *obsmetrics.FulfillmentMetrics) *FulfillmentService {
	if s == nil {
		return nil
	}
	s.metrics = m
	return s
}

// Enabled reports whether fulfillment process creation is active.
func (s *FulfillmentService) Enabled() bool { return s != nil && s.enabled }

// recordingReady reports whether best-effort recording can run: the service must
// be enabled AND have a pool + attempt repository wired.
func (s *FulfillmentService) recordingReady() bool {
	return s.Enabled() && s.pool != nil && s.attempts != nil
}

// EnsureProcessForOrder creates the fulfillment process for an order and enqueues
// the initial "order.created" orchestration event, running inside the caller's
// tenant transaction (tx) so it commits atomically with the order. It is a no-op
// when the service is disabled, and idempotent: if a process already exists for
// the order it is returned unchanged and no duplicate event is enqueued (the
// outbox dedups on idempotency key regardless).
func (s *FulfillmentService) EnsureProcessForOrder(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID) (*model.FulfillmentProcess, error) {
	if !s.Enabled() {
		return nil, nil
	}

	// Idempotent: reuse an existing process rather than creating a second one.
	if existing, err := s.fulfillment.GetProcessByOrder(ctx, tx, orderID); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup fulfillment process for order: %w", err)
	}

	proc, err := s.fulfillment.CreateProcess(ctx, tx, model.FulfillmentProcess{
		TenantID: tenantID,
		OrderID:  orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("create fulfillment process: %w", err)
	}

	if _, _, err := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
		TenantID:       tenantID,
		ProcessID:      proc.ID,
		EventType:      EventOrderCreated,
		IdempotencyKey: EventOrderCreated + ":" + orderID.String(),
		Payload:        map[string]string{"order_id": orderID.String()},
	}); err != nil {
		return nil, fmt.Errorf("enqueue order.created event: %w", err)
	}

	return proc, nil
}

// EnsureProcessForOrderUnconditional creates/reuses the order's fulfillment process WITHOUT
// the FULFILLMENT_PROCESS_ENABLED gate, for callers gated by their own flag (OPE-421
// external-workflow). Idempotent (GetProcessByOrder first). Unlike EnsureProcessForOrder it
// does NOT enqueue an order.created event — it only guarantees the outbox process_id FK has
// a process to anchor to.
func (s *FulfillmentService) EnsureProcessForOrderUnconditional(ctx context.Context, tx pgx.Tx, tenantID, orderID uuid.UUID) (*model.FulfillmentProcess, error) {
	if existing, err := s.fulfillment.GetProcessByOrder(ctx, tx, orderID); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup process: %w", err)
	}
	return s.fulfillment.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantID, OrderID: orderID})
}

// ProviderAttemptInput describes a carrier/provider attempt to record. OrderID is
// optional and used only to correlate the attempt to an existing fulfillment
// process; CorrelationID (typically the shipment id) is always recorded so the
// attempt is traceable even before a process exists.
type ProviderAttemptInput struct {
	TenantID          uuid.UUID
	OrderID           uuid.UUID // optional; uuid.Nil to skip process lookup
	Provider          string
	Operation         string
	Status            string
	RequestID         string
	CorrelationID     string
	RawProviderStatus string
	ResultRedacted    any
	PayloadHash       string
	ErrorCode         string
}

// HashPayload returns a stable sha256 hex digest for use as ProviderAttempt
// PayloadHash. It is a pure helper (no secret material is retained) — callers pass
// a non-sensitive identity string (e.g. external id + format) to support label
// download idempotency intent without storing raw payloads.
func HashPayload(parts ...string) string {
	h := sha256.New()
	for i, p := range parts {
		if i > 0 {
			_, _ = h.Write([]byte{0})
		}
		_, _ = h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// RecordProviderAttempt records a provider attempt as a BEST-EFFORT side effect.
// It is a complete no-op when the service is disabled or recording is not wired,
// runs its own WithTenant transaction, and on ANY error it logs and returns nil —
// never propagating to the caller. Returns the persisted attempt (or nil).
func (s *FulfillmentService) RecordProviderAttempt(ctx context.Context, in ProviderAttemptInput) *model.ProviderAttempt {
	if !s.recordingReady() {
		return nil
	}

	var attempt *model.ProviderAttempt
	err := database.WithTenant(ctx, s.pool, in.TenantID, func(tx pgx.Tx) error {
		var processID *uuid.UUID
		if in.OrderID != uuid.Nil {
			if proc, perr := s.fulfillment.GetProcessByOrder(ctx, tx, in.OrderID); perr == nil {
				processID = &proc.ID
			} else if !errors.Is(perr, pgx.ErrNoRows) {
				return fmt.Errorf("lookup process for attempt: %w", perr)
			}
		}
		out, rerr := s.attempts.RecordAttempt(ctx, tx, model.ProviderAttempt{
			TenantID:          in.TenantID,
			ProcessID:         processID,
			Provider:          in.Provider,
			Operation:         in.Operation,
			Status:            in.Status,
			RequestID:         in.RequestID,
			CorrelationID:     in.CorrelationID,
			RawProviderStatus: in.RawProviderStatus,
			ResultRedacted:    in.ResultRedacted,
			PayloadHash:       in.PayloadHash,
			ErrorCode:         in.ErrorCode,
		})
		if rerr != nil {
			return rerr
		}
		attempt = out
		return nil
	})
	if err != nil {
		slog.Warn("fulfillment: record provider attempt failed (best-effort, ignored)",
			"tenant_id", in.TenantID, "provider", in.Provider, "operation", in.Operation, "error", err)
		return nil
	}
	// Best-effort metric AFTER the DB write succeeds. Operation/status are bounded
	// enums (mirroring the DB CHECK constraint); the collector coerces any unexpected
	// value to "other", so cardinality is bounded regardless. Never an identifier.
	s.metrics.RecordProviderAttempt(in.Operation, in.Status)
	return attempt
}

// EmitFulfillmentStep enqueues a fulfillment-step orchestration event for an order
// as a BEST-EFFORT side effect. It is a no-op when disabled / not wired or when no
// fulfillment process exists for the order, runs its own WithTenant transaction,
// and logs-and-continues on any error. The event is deduped per
// (process, step_key, status, correlation) via its idempotency key.
func (s *FulfillmentService) EmitFulfillmentStep(ctx context.Context, tenantID, orderID uuid.UUID, stepKey, status, correlationID string, payload map[string]any) {
	if !s.recordingReady() {
		return
	}
	if !model.IsValidStepKey(stepKey) {
		slog.Warn("fulfillment: emit step skipped, unknown step key (best-effort)", "step_key", stepKey)
		return
	}

	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		proc, perr := s.fulfillment.GetProcessByOrder(ctx, tx, orderID)
		if errors.Is(perr, pgx.ErrNoRows) {
			return nil // no process to attach the step event to — nothing to emit
		}
		if perr != nil {
			return fmt.Errorf("lookup process for step: %w", perr)
		}

		fullPayload := map[string]any{"step_key": stepKey, "status": status}
		maps.Copy(fullPayload, payload)
		idem := EventFulfillmentStep + ":" + proc.ID.String() + ":" + stepKey + ":" + status
		if correlationID != "" {
			idem += ":" + correlationID
		}
		if _, _, eerr := s.orchestration.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
			TenantID:       tenantID,
			ProcessID:      proc.ID,
			EventType:      EventFulfillmentStep,
			IdempotencyKey: idem,
			Payload:        fullPayload,
		}); eerr != nil {
			return fmt.Errorf("enqueue fulfillment step event: %w", eerr)
		}
		return nil
	})
	if err != nil {
		slog.Warn("fulfillment: emit step failed (best-effort, ignored)",
			"tenant_id", tenantID, "order_id", orderID, "step_key", stepKey, "error", err)
	}
}

// RecordTrackingSyncAttempt records a sync_tracking provider attempt (gated,
// best-effort). The raw provider status is ALWAYS preserved via the mapping. When
// the status is unmapped (mapping.Mapped == false) this is an explicit
// unsupported-capability outcome: it is still recorded (with the raw status and an
// unmapped marker) but NO blocker is created and the await_tracking step is only
// emitted when a canonical status was resolved.
func (s *FulfillmentService) RecordTrackingSyncAttempt(ctx context.Context, tenantID, orderID uuid.UUID, provider, correlationID string, mapping model.TrackingStatusMapping, status string) {
	if !s.recordingReady() {
		return
	}
	result := map[string]any{"raw_status": mapping.Raw, "mapped": mapping.Mapped}
	if mapping.Mapped {
		result["canonical_status"] = mapping.Canonical
	} else {
		// Unsupported/unmapped provider status — recorded as an outcome, not a failure.
		result["unsupported_capability"] = true
	}
	s.RecordProviderAttempt(ctx, ProviderAttemptInput{
		TenantID:          tenantID,
		OrderID:           orderID,
		Provider:          provider,
		Operation:         model.ProviderOpSyncTracking,
		Status:            status,
		CorrelationID:     correlationID,
		RawProviderStatus: mapping.Raw,
		ResultRedacted:    result,
	})
	if mapping.Mapped {
		s.EmitFulfillmentStep(ctx, tenantID, orderID,
			model.StepAwaitTracking, model.FulfillmentStatusSucceeded, correlationID,
			map[string]any{"provider": provider, "canonical_status": mapping.Canonical, "raw_status": mapping.Raw})
	}
}

// RecordTrackingSyncFailure records a failed sync_tracking attempt + a typed
// carrier blocker (gated, best-effort).
func (s *FulfillmentService) RecordTrackingSyncFailure(ctx context.Context, tenantID, orderID uuid.UUID, provider, correlationID string, cause error) {
	if !s.recordingReady() {
		return
	}
	failClass := classifyCarrierError(cause)
	s.RecordProviderAttempt(ctx, ProviderAttemptInput{
		TenantID:      tenantID,
		OrderID:       orderID,
		Provider:      provider,
		Operation:     model.ProviderOpSyncTracking,
		Status:        model.ProviderAttemptFailed,
		CorrelationID: correlationID,
		ErrorCode:     failClass,
	})
	s.EmitFulfillmentStep(ctx, tenantID, orderID,
		model.StepAwaitTracking, model.FulfillmentStatusFailed, correlationID,
		map[string]any{"provider": provider, "failure_class": failClass})
	s.CreateCarrierBlocker(ctx, tenantID, orderID, failClass,
		fmt.Sprintf("sync_tracking via %s failed: %s", provider, failClass))
}

// CreateCarrierBlocker creates a typed fulfillment blocker for a carrier failure
// as a BEST-EFFORT side effect. The failureClass is translated to an existing
// blocker code via model.CarrierFailureBlockerCode. It is a no-op when disabled /
// not wired or when no fulfillment process exists for the order, runs its own
// WithTenant transaction, and logs-and-continues on any error.
func (s *FulfillmentService) CreateCarrierBlocker(ctx context.Context, tenantID, orderID uuid.UUID, failureClass, description string) *model.FulfillmentBlocker {
	if !s.recordingReady() {
		return nil
	}

	code := model.CarrierFailureBlockerCode(failureClass)

	var blocker *model.FulfillmentBlocker
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		proc, perr := s.fulfillment.GetProcessByOrder(ctx, tx, orderID)
		if errors.Is(perr, pgx.ErrNoRows) {
			return nil // no process — nothing to attach the blocker to
		}
		if perr != nil {
			return fmt.Errorf("lookup process for blocker: %w", perr)
		}
		out, berr := s.fulfillment.CreateBlocker(ctx, tx, model.FulfillmentBlocker{
			TenantID:    tenantID,
			ProcessID:   proc.ID,
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
		slog.Warn("fulfillment: create carrier blocker failed (best-effort, ignored)",
			"tenant_id", tenantID, "order_id", orderID, "code", code, "error", err)
		return nil
	}
	// Best-effort metric: count the blocker by its bounded category, only when a
	// blocker row was actually created (no process => blocker is nil => no count).
	// model.BlockerCategory maps the code to one of the bounded category enums.
	if blocker != nil {
		s.metrics.RecordBlocker(model.BlockerCategory(code))
	}
	return blocker
}
