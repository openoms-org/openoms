package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// AutomationOrderTransitioner is the narrow OrderService surface the automation
// status handler needs: read the order's current status (for the idempotency skip)
// and force a status transition. Implemented by *OrderService.
type AutomationOrderTransitioner interface {
	Get(ctx context.Context, tenantID, orderID uuid.UUID) (*model.Order, error)
	TransitionStatus(ctx context.Context, tenantID, orderID uuid.UUID, req model.StatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Order, error)
}

// automationSetStatusPayload is the outbox payload enqueued by the orchestration-
// routed set_status automation action (automation.EventAutomationSetStatus).
type automationSetStatusPayload struct {
	OrderID      string `json:"order_id"`
	TargetStatus string `json:"target_status"`
	RuleID       string `json:"rule_id"`
	ActionIndex  int    `json:"action_index"`
}

// AutomationStatusTransitionHandler applies a set_status automation action that was
// routed through the orchestration outbox (OPE-421). It is registered against
// automation.EventAutomationSetStatus and invoked by the orchestration worker as it
// drains the outbox.
//
// Idempotency: if the order is already in the target status the handler is a
// no-op — it does NOT re-transition, so the per-transition side effects (email,
// SMS, webhook, invoice, and the recursive order.status_changed automation event)
// fire at most once. This both deduplicates outbox re-delivery and breaks the
// self-recursion loop that a direct set_status would otherwise cause.
//
// A malformed payload or an unknown/invalid target status is a model.PermanentError:
// retrying cannot fix it, so the worker fails the event once and opens an
// automation_action_failed blocker instead of retrying forever.
type AutomationStatusTransitionHandler struct {
	orders AutomationOrderTransitioner
}

// NewAutomationStatusTransitionHandler creates the handler. orders MUST be an
// OrderService whose own pool sets the tenant context per call (Get /
// TransitionStatus wrap database.WithTenant internally), matching the privileged
// worker-pool contract used by the other orchestration handlers.
func NewAutomationStatusTransitionHandler(orders AutomationOrderTransitioner) *AutomationStatusTransitionHandler {
	return &AutomationStatusTransitionHandler{orders: orders}
}

// Handle parses the event payload, loads the order, and transitions it to the
// target status unless it is already there. See the type doc for the idempotency
// and permanent-failure contract.
func (h *AutomationStatusTransitionHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	payload, err := decodeOutboxPayload[automationSetStatusPayload](event.Payload)
	if err != nil {
		return model.Permanent(fmt.Errorf("automation.set_status: %w", err))
	}

	orderID, err := uuid.Parse(payload.OrderID)
	if err != nil {
		return model.Permanent(fmt.Errorf("automation.set_status: invalid order_id %q: %w", payload.OrderID, err))
	}
	if payload.TargetStatus == "" {
		return model.Permanent(errors.New("automation.set_status: empty target_status"))
	}

	order, err := h.orders.Get(ctx, event.TenantID, orderID)
	if err != nil {
		// A missing order can never be transitioned — permanent.
		if errors.Is(err, ErrOrderNotFound) {
			return model.Permanent(fmt.Errorf("automation.set_status: %w", err))
		}
		// Other load errors (e.g. transient DB) are retryable.
		return fmt.Errorf("automation.set_status: load order %s: %w", orderID, err)
	}

	// Idempotency: already in the target status -> no-op. Do NOT re-transition.
	if order.Status == payload.TargetStatus {
		return nil
	}

	// Force=true: automation transitions are system-driven and bypass the
	// per-tenant transition graph (matching the direct executeSetStatus path).
	//
	// IP is intentionally empty: the audit log column is inet (TransitionStatus
	// writes the actor IP there), and "automation" — the string the legacy direct
	// path passes — is not a valid inet, so it would make the audit insert (and thus
	// the whole transition) fail. The empty string is stored as NULL; the automation
	// actor is already conveyed by uuid.Nil + the audit action, not the IP slot.
	_, err = h.orders.TransitionStatus(ctx, event.TenantID, orderID, model.StatusTransitionRequest{
		Status: payload.TargetStatus,
		Force:  true,
	}, uuid.Nil, "")
	if err != nil {
		// An unknown/invalid target status (or other validation failure) is not
		// retryable — fail permanently so the worker opens a visible blocker.
		if errors.Is(err, ErrUnknownStatus) || errors.Is(err, ErrInvalidTransition) || isValidationError(err) {
			return model.Permanent(fmt.Errorf("automation.set_status: transition to %q: %w", payload.TargetStatus, err))
		}
		// Order disappeared between load and transition — also permanent.
		if errors.Is(err, ErrOrderNotFound) {
			return model.Permanent(fmt.Errorf("automation.set_status: %w", err))
		}
		// Anything else is treated as transient and retried by the worker.
		return fmt.Errorf("automation.set_status: transition to %q: %w", payload.TargetStatus, err)
	}
	return nil
}

// isValidationError reports whether err is (or wraps) a service ValidationError.
func isValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// decodeOutboxPayload normalises an orchestration outbox payload (stored as
// jsonb and scanned back as an arbitrary `any`) into the typed payload T.
func decodeOutboxPayload[T any](raw any) (T, error) {
	var p T
	if raw == nil {
		return p, errors.New("missing payload")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return p, fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("unmarshal payload: %w", err)
	}
	return p, nil
}

// Compile-time assertion that this handler satisfies OrchestrationHandler. The
// producer event-type constant automation.EventAutomationSetStatus is referenced at
// registration in cmd/server/main.go, keeping producer and consumer in sync.
var _ OrchestrationHandler = (*AutomationStatusTransitionHandler)(nil)
