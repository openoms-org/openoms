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
	FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.DropshipOrder, error)
	Create(ctx context.Context, tx pgx.Tx, d *model.DropshipOrder) error
	UpdateFields(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateDropshipStatusRequest) error
	MarkSubmitAttempted(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// orderReader is the narrow order repo surface the handler needs.
type orderReader interface {
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
}

// supplierOrderItemsLoader resolves the dropship lines for an (order, supplier) into the
// pure builder's input shape (each line's product EAN + supplier catalogue id + quantity).
// It runs inside the handler's tenant tx.
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
//
// OPE-517/OPE-518 two-phase submit: all supplier I/O (preflight + CreateOrder) happens
// OUTSIDE any open database transaction. The dropship row is locked FOR UPDATE before the
// idempotency check, and a submit-intent marker (submit_attempted_at) is committed before
// CreateOrder. If recording the supplier reference fails after a successful CreateOrder,
// the retry observes the marker and opens an operator blocker instead of placing a
// duplicate purchase order at the wholesaler.
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

// supplierSubmitPrep carries the result of the prepare phase (TX1) into the supplier I/O
// phases that run outside any open transaction.
type supplierSubmitPrep struct {
	existingID uuid.UUID // pending dropship_orders row to update; uuid.Nil -> defensive create
	req        integration.SupplierOrderRequest
	provider   integration.SupplierProvider
	currency   string
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

	// PREPARE (TX1): lock + idempotency/marker gate + request assembly. No supplier I/O.
	prep, err := h.prepare(ctx, tenantID, orderID, supplierID, unitID)
	if err != nil {
		return err // retryable
	}
	if prep == nil {
		return nil // handled in TX1: idempotent no-op, or a typed blocker was opened
	}

	// PREFLIGHT (optional capability) — supplier I/O, outside any open transaction.
	h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepPreflightSupplierOrder, model.FulfillmentStatusRunning, nil)
	if pf, isPreflighter := prep.provider.(integration.SupplierPreflighter); isPreflighter {
		res, pferr := pf.Preflight(ctx, prep.req)
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

	// SUBMIT — two-phase. First commit the submit-intent marker, then call the supplier.
	h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusRunning, nil)
	dsID, proceed, err := h.markSubmitIntent(ctx, tenantID, orderID, supplierID, unitID, prep)
	if err != nil {
		return err // retryable; no marker was committed
	}
	if !proceed {
		return nil // concurrent outcome observed: no-op, or a blocker was opened
	}

	// The supplier call runs with NO open transaction. ClientOrderNumber (the customer
	// order id) keeps flowing to the adapter — it is the supplier-side idempotency token.
	result, serr := prep.provider.CreateOrder(ctx, prep.req)
	if serr != nil {
		// A business rejection is permanent; transport is retryable. The adapter should
		// wrap business errors as model.Permanent; otherwise treat as retryable. Note the
		// marker stays set: a retry after an AMBIGUOUS transport failure (the order may
		// have been placed) opens an operator blocker instead of resubmitting blindly.
		if model.IsPermanent(serr) {
			h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierOrderRejected, serr.Error())
			return nil
		}
		return fmt.Errorf("supplier create order: %w", serr) // retryable -> marker branch next time
	}

	// RECORD (TX2): persist the supplier order id onto the marked row. The UPDATE takes the
	// row lock; the marker is KEPT as historical evidence. If this fails after a successful
	// CreateOrder, the retry hits the marker branch -> operator blocker, never a duplicate.
	ref := result.ExternalOrderID
	if result.OrderNumber != "" {
		ref = result.OrderNumber
	}
	if rerr := database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		if uerr := h.dropshipRepo.UpdateFields(ctx, tx, dsID, model.UpdateDropshipStatusRequest{
			Status:            "sent",
			SupplierReference: &ref,
		}); uerr != nil {
			return fmt.Errorf("update dropship order with supplier reference: %w", uerr)
		}
		return nil
	}); rerr != nil {
		return fmt.Errorf("record supplier order submission: %w", rerr) // retryable -> marker branch
	}
	h.fulfillment.RecordStep(ctx, tenantID, unitID, model.StepCreateDropshipOrder, model.FulfillmentStatusSucceeded,
		map[string]any{"supplier_reference": ref})
	h.fulfillment.RecordUnitTransition(ctx, tenantID, unitID, model.FulfillmentStatusWaitingExternal)
	h.fulfillment.AggregateMixedOrder(ctx, tenantID, orderID)
	return nil
}

// prepare runs TX1: it locks the (order, supplier) pending dropship row FOR UPDATE, applies
// the idempotency guard (supplier_reference set -> no-op) and the submit-intent marker gate
// (marker set + reference NULL -> operator blocker, never an automatic resubmit), then
// assembles the supplier order request and provider. It returns nil prep with nil error when
// the event was fully handled in TX1 (no-op or blocker).
func (h *SupplierOrderHandler) prepare(ctx context.Context, tenantID, orderID, supplierID, unitID uuid.UUID) (*supplierSubmitPrep, error) {
	var prep *supplierSubmitPrep
	err := database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		// Idempotency: already submitted? (a dropship_orders row with a supplier_reference).
		// Also capture the existing pending row for this (order, supplier) — the dropship gate
		// creates it before enqueuing the submit, so success UPDATES it rather than inserting a
		// duplicate. existingID == uuid.Nil means no row yet (defensive create).
		existing, ferr := h.dropshipRepo.FindByOrderID(ctx, tx, orderID)
		if ferr != nil {
			return fmt.Errorf("lookup existing dropship orders: %w", ferr) // retryable
		}
		var existingID uuid.UUID
		for i := range existing {
			if existing[i].SupplierID != supplierID {
				continue
			}
			if existing[i].SupplierReference != nil && strings.TrimSpace(*existing[i].SupplierReference) != "" {
				return nil // already placed — idempotent no-op
			}
			existingID = existing[i].ID // pending row to update on success
		}
		// OPE-518: lock the pending row FOR UPDATE before trusting supplier_reference, so a
		// concurrent submitter (manual pending→sent transition) serializes against this tx
		// and the re-read below observes its committed outcome.
		if existingID != uuid.Nil {
			locked, lkerr := h.dropshipRepo.FindByIDForUpdate(ctx, tx, existingID)
			if lkerr != nil {
				return fmt.Errorf("lock dropship order: %w", lkerr) // retryable
			}
			switch {
			case locked == nil:
				existingID = uuid.Nil // row vanished concurrently — fall back to defensive create
			case locked.SupplierReference != nil && strings.TrimSpace(*locked.SupplierReference) != "":
				return nil // placed by a concurrent submitter — idempotent no-op
			case locked.SubmitAttemptedAt != nil:
				// OPE-517: a previous attempt committed the submit intent but never recorded a
				// reference — the order MAY exist at the supplier. Never resubmit blindly.
				h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierManualSubmissionRequired,
					fmt.Sprintf("possible duplicate submission — verify order %s at the supplier before retrying", orderID))
				return nil
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
		prep = &supplierSubmitPrep{existingID: existingID, req: req, provider: provider, currency: ord.Currency}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return prep, nil
}

// markSubmitIntent commits the submit-intent marker in its own short transaction, re-checking
// the locked row first: a reference set concurrently -> no-op (proceed=false), a marker set
// concurrently -> operator blocker (proceed=false). When no pending row exists it creates one
// (defensive) carrying the marker, so a retry can never resubmit blindly. It returns the
// dropship row id to record the supplier reference onto.
func (h *SupplierOrderHandler) markSubmitIntent(ctx context.Context, tenantID, orderID, supplierID, unitID uuid.UUID, prep *supplierSubmitPrep) (uuid.UUID, bool, error) {
	dsID := prep.existingID
	proceed := true
	err := database.WithTenant(ctx, h.pool, tenantID, func(tx pgx.Tx) error {
		if dsID != uuid.Nil {
			locked, lerr := h.dropshipRepo.FindByIDForUpdate(ctx, tx, dsID)
			if lerr != nil {
				return fmt.Errorf("lock dropship order for submit intent: %w", lerr) // retryable
			}
			if locked != nil {
				if locked.SupplierReference != nil && strings.TrimSpace(*locked.SupplierReference) != "" {
					proceed = false // placed by a concurrent submitter — idempotent no-op
					return nil
				}
				if locked.SubmitAttemptedAt != nil {
					h.blocker(ctx, tenantID, orderID, unitID, model.BlockerSupplierManualSubmissionRequired,
						fmt.Sprintf("possible duplicate submission — verify order %s at the supplier before retrying", orderID))
					proceed = false
					return nil
				}
				return h.dropshipRepo.MarkSubmitAttempted(ctx, tx, dsID)
			}
			dsID = uuid.Nil // row vanished — defensive create below
		}
		currency := prep.currency
		if currency == "" {
			currency = "PLN"
		}
		d := &model.DropshipOrder{
			ID:         uuid.New(),
			TenantID:   tenantID,
			OrderID:    orderID,
			SupplierID: supplierID,
			Status:     "pending",
			Currency:   currency,
		}
		if cerr := h.dropshipRepo.Create(ctx, tx, d); cerr != nil {
			return fmt.Errorf("create dropship order for submit intent: %w", cerr)
		}
		if merr := h.dropshipRepo.MarkSubmitAttempted(ctx, tx, d.ID); merr != nil {
			return merr
		}
		dsID = d.ID
		return nil
	})
	if err != nil {
		return uuid.Nil, false, err
	}
	return dsID, proceed, nil
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

// Compile-time assertion that the handler satisfies OrchestrationHandler.
var _ OrchestrationHandler = (*SupplierOrderHandler)(nil)
