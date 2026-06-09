package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/integration"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// EventSupplierOrderSubmit is the orchestration outbox event type that drives a dropship
// unit through prepare -> preflight -> submit (OPE-418/Phase-7).
const EventSupplierOrderSubmit = "supplier.order.submit"

// dropshipOrderWriter is the narrow dropship-order repo surface the handler needs.
type dropshipOrderWriter interface {
	FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.DropshipOrder, error)
	Create(ctx context.Context, tx pgx.Tx, d *model.DropshipOrder) error
}

// orderReader is the narrow order repo surface the handler needs.
type orderReader interface {
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
}

// supplierOrderItemsLoader resolves the dropship lines for an (order, supplier) into the
// pure builder's input shape (each line's product EAN + supplier SKU + quantity). It runs
// inside the handler's tenant tx.
type supplierOrderItemsLoader func(ctx context.Context, tx pgx.Tx, orderID, supplierID uuid.UUID) ([]SupplierOrderInputLine, []string, error)

// supplierProviderFactory builds a SupplierProvider for a supplier inside the handler's
// tenant tx (loads the supplier + decrypts its integration credentials). Returns a nil
// provider with a nil error when the supplier has no API provider (should not happen on the
// api-capability branch, but the handler treats it defensively as a rejected order).
type supplierProviderFactory func(ctx context.Context, tx pgx.Tx, tenantID, supplierID uuid.UUID) (integration.SupplierProvider, error)

// SupplierOrderHandler implements OrchestrationHandler for supplier.order.submit. The event
// payload carries {order_id, supplier_id, unit_id}. It loads the order + dropship lines,
// builds the request, runs prepare -> preflight -> submit recording a fulfillment step at
// each, and is idempotent: an existing dropship_orders row with a supplier_reference for the
// (order, supplier) -> skip submit. A phase that cannot proceed creates a typed blocker via
// the fulfillment service and returns nil (the unit waits); a transport error returns a
// plain error (retryable by the orchestration worker's backoff).
type SupplierOrderHandler struct {
	pool         *pgxpool.Pool
	fulfillment  *FulfillmentService
	dropshipRepo dropshipOrderWriter
	orderRepo    orderReader
	itemsFor     supplierOrderItemsLoader
	newProvider  supplierProviderFactory
}

// NewSupplierOrderHandler constructs the handler with its real collaborators (wired in main.go).
func NewSupplierOrderHandler(
	pool *pgxpool.Pool,
	fulfillment *FulfillmentService,
	dropshipRepo dropshipOrderWriter,
	orderRepo orderReader,
	itemsFor supplierOrderItemsLoader,
	newProvider supplierProviderFactory,
) *SupplierOrderHandler {
	return &SupplierOrderHandler{
		pool:         pool,
		fulfillment:  fulfillment,
		dropshipRepo: dropshipRepo,
		orderRepo:    orderRepo,
		itemsFor:     itemsFor,
		newProvider:  newProvider,
	}
}

// Handle runs the three synchronous phases. The payload is {order_id, supplier_id, unit_id}.
func (h *SupplierOrderHandler) Handle(ctx context.Context, event model.OrchestrationOutboxEvent) error {
	p, ok := event.Payload.(map[string]any)
	if !ok {
		return model.Permanent(fmt.Errorf("supplier order submit: malformed payload %T", event.Payload))
	}
	orderID, err := uuid.Parse(fmt.Sprint(p["order_id"]))
	if err != nil {
		return model.Permanent(fmt.Errorf("supplier order submit: invalid order_id: %w", err))
	}
	supplierID, err := uuid.Parse(fmt.Sprint(p["supplier_id"]))
	if err != nil {
		return model.Permanent(fmt.Errorf("supplier order submit: invalid supplier_id: %w", err))
	}
	unitID, err := uuid.Parse(fmt.Sprint(p["unit_id"]))
	if err != nil {
		return model.Permanent(fmt.Errorf("supplier order submit: invalid unit_id: %w", err))
	}
	tenantID := event.TenantID

	return database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		// Idempotency: already submitted? (a dropship_orders row with a supplier_reference)
		existing, ferr := h.dropshipRepo.FindByOrderID(ctx, tx, orderID)
		if ferr != nil {
			return fmt.Errorf("lookup existing dropship orders: %w", ferr) // retryable
		}
		for i := range existing {
			if existing[i].SupplierID == supplierID &&
				existing[i].SupplierReference != nil && strings.TrimSpace(*existing[i].SupplierReference) != "" {
				return nil // already placed — idempotent no-op
			}
		}

		// PREPARE
		lines, missing, lerr := h.itemsFor(ctx, tx, orderID, supplierID)
		if lerr != nil {
			return fmt.Errorf("load dropship lines: %w", lerr) // retryable
		}
		ord, oerr := h.orderRepo.FindByID(ctx, tx, orderID)
		if oerr != nil {
			return fmt.Errorf("load order: %w", oerr) // retryable
		}
		if ord == nil {
			return model.Permanent(fmt.Errorf("supplier order submit: order %s not found", orderID))
		}
		req, buildMissing, ambiguous := BuildSupplierOrderRequest(ord.ID.String(), lines)
		missing = append(missing, buildMissing...)
		if ambiguous {
			h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierOrderAmbiguousSKU, "ambiguous SKU/EAN identity")
			return nil
		}
		if len(missing) > 0 {
			h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierOrderMissingData,
				fmt.Sprintf("%d line(s) missing EAN/SKU", len(missing)))
			return nil
		}
		if len(req.Lines) == 0 {
			h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierOrderMissingData, "no order lines to submit")
			return nil
		}

		provider, perr := h.newProvider(ctx, tx, tenantID, supplierID)
		if perr != nil {
			return fmt.Errorf("build supplier provider: %w", perr) // retryable (transient)
		}
		if provider == nil {
			// api capability classified but no provider resolvable now — treat as rejected.
			h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierOrderRejected, "no supplier order provider available")
			return nil
		}

		// PREFLIGHT (optional capability)
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepPreflightSupplierOrder, model.FulfillmentStatusRunning, nil)
		if pf, isPreflighter := provider.(integration.SupplierPreflighter); isPreflighter {
			res, pferr := pf.Preflight(ctx, req)
			if pferr != nil {
				return fmt.Errorf("supplier preflight: %w", pferr) // retryable transport
			}
			if !res.Accepted {
				code := model.BlockerSupplierOrderRejected
				desc := "supplier preflight rejected"
				if len(res.MissingFields) > 0 {
					code = model.BlockerSupplierOrderMissingData
					desc = fmt.Sprintf("supplier preflight: missing fields %v", res.MissingFields)
				}
				h.blocker(ctx, tenantID, orderID, unitID, code, desc)
				return nil
			}
			if res.PaymentDue {
				h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierPaymentAwaiting, "supplier order awaiting payment")
				return nil
			}
			if len(res.SplitLines) > 0 {
				h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierPartialFulfillment, "supplier split requires operator review")
				return nil
			}
		}
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepPreflightSupplierOrder, model.FulfillmentStatusSucceeded, nil)

		// SUBMIT
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusRunning, nil)
		result, serr := provider.CreateOrder(ctx, req)
		if serr != nil {
			// A business rejection is permanent; transport is retryable. The adapter should
			// wrap business errors as model.Permanent; otherwise treat as retryable.
			if model.IsPermanent(serr) {
				h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierOrderRejected, serr.Error())
				return nil
			}
			return fmt.Errorf("supplier create order: %w", serr) // retryable
		}
		// Persist the supplier order id on a dropship_orders row.
		if e := h.recordSubmitted(ctx, tx, tenantID, orderID, supplierID, ord.Currency, result); e != nil {
			return e
		}
		ref := result.ExternalOrderID
		if result.OrderNumber != "" {
			ref = result.OrderNumber
		}
		h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusSucceeded,
			map[string]any{"supplier_reference": ref})
		h.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusWaitingExternal)
		h.fulfillment.AggregateMixedOrder(ctx, tenantID, orderID)
		return nil
	})
}

// blocker records the blocked step + unit transition + a typed supplier blocker (best-effort).
// The fulfillment recording methods open their own tenant transactions, so this is safe to
// call from inside the handler's tx.
func (h *SupplierOrderHandler) blocker(ctx context.Context, tenantID, orderID, unitID uuid.UUID, code, desc string) {
	h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusBlocked, map[string]any{"reason": code})
	h.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusBlocked)
	h.fulfillment.CreateSupplierBlocker(ctx, tenantID, orderID, &unitID, code, desc)
	h.fulfillment.AggregateMixedOrder(ctx, tenantID, orderID)
}

// recordSubmitted creates the dropship_orders row carrying the supplier order id (external
// order id, or the supplier's order number when present). Runs inside the handler's tx so the
// supplier_reference write commits with the idempotency guard's read.
func (h *SupplierOrderHandler) recordSubmitted(ctx context.Context, tx pgx.Tx, tenantID, orderID, supplierID uuid.UUID, currency string, result *integration.SupplierOrderResult) error {
	ref := result.ExternalOrderID
	if result.OrderNumber != "" {
		ref = result.OrderNumber
	}
	if currency == "" {
		currency = "PLN"
	}
	d := &model.DropshipOrder{
		ID:                uuid.New(),
		TenantID:          tenantID,
		OrderID:           orderID,
		SupplierID:        supplierID,
		Status:            "sent",
		SupplierReference: &ref,
		Currency:          currency,
	}
	if err := h.dropshipRepo.Create(ctx, tx, d); err != nil {
		return fmt.Errorf("record dropship order: %w", err)
	}
	return nil
}

// Compile-time assertion that the handler satisfies OrchestrationHandler.
var _ OrchestrationHandler = (*SupplierOrderHandler)(nil)
