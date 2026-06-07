package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// FulfillmentReader is the subset of service.FulfillmentReadService the
// operations/fulfillment HTTP handlers depend on. It is an interface so the
// handlers can be unit-tested with a stub. Every method is tenant-scoped: the
// implementation runs inside database.WithTenant so PostgreSQL RLS enforces
// isolation — a tenant can never read or mutate another tenant's rows.
type FulfillmentReader interface {
	ListProcesses(ctx context.Context, tenantID uuid.UUID, aggregateStatuses, healthStatuses []string, limit, offset int) (service.ProcessListResult, error)
	GetProcessDetail(ctx context.Context, tenantID, processID uuid.UUID) (*service.ProcessDetail, error)
	GetOrderFulfillment(ctx context.Context, tenantID, orderID uuid.UUID) (*service.ProcessDetail, error)
	OperationsSummary(ctx context.Context, tenantID uuid.UUID) (service.OperationsSummaryResult, error)
	OperationsExceptions(ctx context.Context, tenantID uuid.UUID, limit int) ([]service.ExceptionItem, error)
	IntegrationCapabilitySummary(ctx context.Context, tenantID uuid.UUID, scanLimit int) (service.IntegrationCapabilitySummaryResult, error)
	ParityReport(ctx context.Context, tenantID uuid.UUID, threshold float64) (service.ParityReport, error)
	ResolveBlocker(ctx context.Context, tenantID, blockerID, actorID uuid.UUID, ip string) (*model.FulfillmentBlocker, error)
	RetryStep(ctx context.Context, tenantID, stepID, actorID uuid.UUID, ip string) (*model.FulfillmentStep, error)
}

// FulfillmentHandler serves the tenant-scoped fulfillment read + operator-action
// endpoints under /v1/fulfillment. The reads are always available and return
// empty results when no fulfillment data has been recorded.
type FulfillmentHandler struct {
	svc FulfillmentReader
}

// NewFulfillmentHandler creates a FulfillmentHandler over the read service.
func NewFulfillmentHandler(svc FulfillmentReader) *FulfillmentHandler {
	return &FulfillmentHandler{svc: svc}
}

// ListProcesses returns a paginated, optionally status/health-filtered list of
// the tenant's fulfillment processes.
//
// Query params: aggregate_status (repeatable or comma-separated), health_status
// (repeatable or comma-separated), limit, offset.
func (h *FulfillmentHandler) ListProcesses(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())
	pagination := model.ParsePagination(r)

	aggregateStatuses := multiValueQuery(r, "aggregate_status")
	healthStatuses := multiValueQuery(r, "health_status")

	result, err := h.svc.ListProcesses(r.Context(), tenantID, aggregateStatuses, healthStatuses, pagination.Limit, pagination.Offset)
	if err != nil {
		writeServerError(w, "failed to list fulfillment processes", err)
		return
	}

	writeJSON(w, http.StatusOK, model.ListResponse[model.FulfillmentProcess]{
		Items:  result.Items,
		Total:  result.Total,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	})
}

// GetProcess returns a single fulfillment process with its units, steps,
// blockers and provider attempts.
func (h *FulfillmentHandler) GetProcess(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	processID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid process ID")
		return
	}

	detail, err := h.svc.GetProcessDetail(r.Context(), tenantID, processID)
	if err != nil {
		if errors.Is(err, service.ErrFulfillmentNotFound) {
			writeError(w, http.StatusNotFound, "fulfillment process not found")
			return
		}
		writeServerError(w, "failed to get fulfillment process", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// GetOrderFulfillment returns the assembled fulfillment detail for an order.
func (h *FulfillmentHandler) GetOrderFulfillment(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	orderID, err := uuid.Parse(chi.URLParam(r, "orderID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID")
		return
	}

	detail, err := h.svc.GetOrderFulfillment(r.Context(), tenantID, orderID)
	if err != nil {
		if errors.Is(err, service.ErrFulfillmentNotFound) {
			writeError(w, http.StatusNotFound, "fulfillment process not found for order")
			return
		}
		writeServerError(w, "failed to get order fulfillment", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// ResolveBlocker marks a fulfillment blocker resolved for the tenant.
func (h *FulfillmentHandler) ResolveBlocker(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	blockerID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid blocker ID")
		return
	}

	blocker, err := h.svc.ResolveBlocker(r.Context(), tenantID, blockerID, middleware.UserIDFromContext(r.Context()), clientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrFulfillmentNotFound) {
			writeError(w, http.StatusNotFound, "blocker not found")
			return
		}
		writeServerError(w, "failed to resolve blocker", err)
		return
	}
	writeJSON(w, http.StatusOK, blocker)
}

// RetryStep resets a failed/blocked fulfillment step toward pending and enqueues
// an idempotent retry event (when an orchestration outbox is wired).
func (h *FulfillmentHandler) RetryStep(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromContext(r.Context())

	stepID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid step ID")
		return
	}

	step, err := h.svc.RetryStep(r.Context(), tenantID, stepID, middleware.UserIDFromContext(r.Context()), clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFulfillmentNotFound):
			writeError(w, http.StatusNotFound, "step not found")
		case isValidationError(err):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeServerError(w, "failed to retry step", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, step)
}
