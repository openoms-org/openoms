package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ExternalWorkflowCommandOrders is the narrow OrderService surface the external-workflow
// command handler needs to apply a whitelisted follow-on command. Implemented by *OrderService.
type ExternalWorkflowCommandOrders interface {
	Get(ctx context.Context, tenantID, orderID uuid.UUID) (*model.Order, error)
	TransitionStatus(ctx context.Context, tenantID, orderID uuid.UUID, req model.StatusTransitionRequest, actorID uuid.UUID, ip string) (*model.Order, error)
	Update(ctx context.Context, tenantID, orderID uuid.UUID, req model.UpdateOrderRequest, actorID uuid.UUID, ip string) (*model.Order, error)
}

// externalWorkflowCommandPayload is the follow-on command enqueued by ResolveCallback.
type externalWorkflowCommandPayload struct {
	OrderID string `json:"order_id"`
	Command string `json:"command"` // set_status | add_tag | add_note
	Value   string `json:"value"`
}

// ExternalWorkflowCommandHandler applies the single whitelisted follow-on command a callback
// may submit (set_status | add_tag | add_note), through the audited order service. Registered
// against EventExternalWorkflowCommand and drained by the orchestration worker. A malformed
// payload, an unknown order, or a non-whitelisted command is a model.PermanentError (retrying
// cannot fix it); transient errors are retried.
type ExternalWorkflowCommandHandler struct {
	orders ExternalWorkflowCommandOrders
}

// NewExternalWorkflowCommandHandler creates the handler. orders MUST be an OrderService whose
// own pool sets the tenant context per call (matching the privileged worker-pool contract used
// by the other orchestration handlers).
func NewExternalWorkflowCommandHandler(orders ExternalWorkflowCommandOrders) *ExternalWorkflowCommandHandler {
	return &ExternalWorkflowCommandHandler{orders: orders}
}

// Handle applies the follow-on command for one EventExternalWorkflowCommand event.
func (h *ExternalWorkflowCommandHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	p, err := decodeOutboxPayload[externalWorkflowCommandPayload](event.Payload)
	if err != nil {
		return model.Permanent(fmt.Errorf("external_workflow command: %w", err))
	}
	orderID, err := uuid.Parse(p.OrderID)
	if err != nil {
		return model.Permanent(fmt.Errorf("external_workflow command: invalid order_id %q: %w", p.OrderID, err))
	}
	if !model.IsWhitelistedCallbackCommand(p.Command) {
		return model.Permanent(fmt.Errorf("external_workflow command: %q not whitelisted", p.Command))
	}

	switch p.Command {
	case "set_status":
		return h.applySetStatus(ctx, event.TenantID, orderID, p.Value)
	case "add_tag":
		return h.applyAddTag(ctx, event.TenantID, orderID, p.Value)
	case "add_note":
		return h.applyAddNote(ctx, event.TenantID, orderID, p.Value)
	default:
		return model.Permanent(fmt.Errorf("external_workflow command: unsupported command %q", p.Command))
	}
}

func (h *ExternalWorkflowCommandHandler) applySetStatus(ctx context.Context, tenantID, orderID uuid.UUID, status string) error {
	if status == "" {
		return model.Permanent(errors.New("external_workflow command: set_status requires a value"))
	}
	order, err := h.orders.Get(ctx, tenantID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return model.Permanent(fmt.Errorf("external_workflow command: %w", err))
		}
		return fmt.Errorf("external_workflow command: load order %s: %w", orderID, err)
	}
	if order.Status == status { // idempotent: already in target
		return nil
	}
	// Not forced: the external engine is a third party and cannot opt out of the
	// tenant's transition graph. An edge the tenant does not allow fails permanently
	// below rather than silently rewriting the order lifecycle.
	_, err = h.orders.TransitionStatus(ctx, tenantID, orderID, model.StatusTransitionRequest{Status: status}, uuid.Nil, "")
	if err != nil {
		if errors.Is(err, ErrUnknownStatus) || errors.Is(err, ErrInvalidTransition) || isValidationError(err) || errors.Is(err, ErrOrderNotFound) {
			return model.Permanent(fmt.Errorf("external_workflow command: set_status %q: %w", status, err))
		}
		return fmt.Errorf("external_workflow command: set_status %q: %w", status, err)
	}
	return nil
}

func (h *ExternalWorkflowCommandHandler) applyAddTag(ctx context.Context, tenantID, orderID uuid.UUID, tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return model.Permanent(errors.New("external_workflow command: add_tag requires a value"))
	}
	order, err := h.orders.Get(ctx, tenantID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return model.Permanent(fmt.Errorf("external_workflow command: %w", err))
		}
		return fmt.Errorf("external_workflow command: load order %s: %w", orderID, err)
	}
	if slices.Contains(order.Tags, tag) { // idempotent: tag already present
		return nil
	}
	tags := append(append([]string{}, order.Tags...), tag)
	_, err = h.orders.Update(ctx, tenantID, orderID, model.UpdateOrderRequest{Tags: &tags}, uuid.Nil, "")
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) || isValidationError(err) {
			return model.Permanent(fmt.Errorf("external_workflow command: add_tag: %w", err))
		}
		return fmt.Errorf("external_workflow command: add_tag: %w", err)
	}
	return nil
}

func (h *ExternalWorkflowCommandHandler) applyAddNote(ctx context.Context, tenantID, orderID uuid.UUID, note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return model.Permanent(errors.New("external_workflow command: add_note requires a value"))
	}
	order, err := h.orders.Get(ctx, tenantID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return model.Permanent(fmt.Errorf("external_workflow command: %w", err))
		}
		return fmt.Errorf("external_workflow command: load order %s: %w", orderID, err)
	}
	combined := note
	if strings.TrimSpace(order.InternalNotes) != "" {
		combined = order.InternalNotes + "\n" + note
	}
	_, err = h.orders.Update(ctx, tenantID, orderID, model.UpdateOrderRequest{InternalNotes: &combined}, uuid.Nil, "")
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) || isValidationError(err) {
			return model.Permanent(fmt.Errorf("external_workflow command: add_note: %w", err))
		}
		return fmt.Errorf("external_workflow command: add_note: %w", err)
	}
	return nil
}

// Compile-time assertion that this handler satisfies OrchestrationHandler.
var _ OrchestrationHandler = (*ExternalWorkflowCommandHandler)(nil)
