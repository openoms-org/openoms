package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// stubFulfillmentReader is a hand-written test double for FulfillmentReader. Each
// field overrides the corresponding method; nil fields return zero values + nil.
type stubFulfillmentReader struct {
	listProcesses     func(ctx context.Context, tenantID uuid.UUID, agg, health []string, limit, offset int) (service.ProcessListResult, error)
	getProcessDetail  func(ctx context.Context, tenantID, processID uuid.UUID) (*service.ProcessDetail, error)
	getOrderFulfill   func(ctx context.Context, tenantID, orderID uuid.UUID) (*service.ProcessDetail, error)
	opsSummary        func(ctx context.Context, tenantID uuid.UUID) (service.OperationsSummaryResult, error)
	opsExceptions     func(ctx context.Context, tenantID uuid.UUID, limit int) ([]service.ExceptionItem, error)
	capabilitySummary func(ctx context.Context, tenantID uuid.UUID, scanLimit int) (service.IntegrationCapabilitySummaryResult, error)
	resolveBlocker    func(ctx context.Context, tenantID, blockerID, actorID uuid.UUID, ip string) (*model.FulfillmentBlocker, error)
	retryStep         func(ctx context.Context, tenantID, stepID, actorID uuid.UUID, ip string) (*model.FulfillmentStep, error)
}

func (s *stubFulfillmentReader) ListProcesses(ctx context.Context, tenantID uuid.UUID, agg, health []string, limit, offset int) (service.ProcessListResult, error) {
	if s.listProcesses != nil {
		return s.listProcesses(ctx, tenantID, agg, health, limit, offset)
	}
	return service.ProcessListResult{Items: []model.FulfillmentProcess{}}, nil
}

func (s *stubFulfillmentReader) GetProcessDetail(ctx context.Context, tenantID, processID uuid.UUID) (*service.ProcessDetail, error) {
	if s.getProcessDetail != nil {
		return s.getProcessDetail(ctx, tenantID, processID)
	}
	return nil, service.ErrFulfillmentNotFound
}

func (s *stubFulfillmentReader) GetOrderFulfillment(ctx context.Context, tenantID, orderID uuid.UUID) (*service.ProcessDetail, error) {
	if s.getOrderFulfill != nil {
		return s.getOrderFulfill(ctx, tenantID, orderID)
	}
	return nil, service.ErrFulfillmentNotFound
}

func (s *stubFulfillmentReader) OperationsSummary(ctx context.Context, tenantID uuid.UUID) (service.OperationsSummaryResult, error) {
	if s.opsSummary != nil {
		return s.opsSummary(ctx, tenantID)
	}
	return service.OperationsSummaryResult{Buckets: map[service.OperatorBucket]int{}}, nil
}

func (s *stubFulfillmentReader) OperationsExceptions(ctx context.Context, tenantID uuid.UUID, limit int) ([]service.ExceptionItem, error) {
	if s.opsExceptions != nil {
		return s.opsExceptions(ctx, tenantID, limit)
	}
	return []service.ExceptionItem{}, nil
}

func (s *stubFulfillmentReader) IntegrationCapabilitySummary(ctx context.Context, tenantID uuid.UUID, scanLimit int) (service.IntegrationCapabilitySummaryResult, error) {
	if s.capabilitySummary != nil {
		return s.capabilitySummary(ctx, tenantID, scanLimit)
	}
	return service.IntegrationCapabilitySummaryResult{}, nil
}

func (s *stubFulfillmentReader) ResolveBlocker(ctx context.Context, tenantID, blockerID, actorID uuid.UUID, ip string) (*model.FulfillmentBlocker, error) {
	if s.resolveBlocker != nil {
		return s.resolveBlocker(ctx, tenantID, blockerID, actorID, ip)
	}
	return nil, service.ErrFulfillmentNotFound
}

func (s *stubFulfillmentReader) RetryStep(ctx context.Context, tenantID, stepID, actorID uuid.UUID, ip string) (*model.FulfillmentStep, error) {
	if s.retryStep != nil {
		return s.retryStep(ctx, tenantID, stepID, actorID, ip)
	}
	return nil, service.ErrFulfillmentNotFound
}

// reqWithTenant builds a request whose context carries an authenticated tenant id
// and the given chi URL params, mirroring what the middleware stack provides.
func reqWithTenant(method, target string, tenantID uuid.UUID, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestFulfillmentHandler_ListProcesses_HappyPath(t *testing.T) {
	tenantID := uuid.New()
	proc := model.FulfillmentProcess{ID: uuid.New(), TenantID: tenantID, OrderID: uuid.New(), AggregateStatus: model.ProcessStatusBlocked, HealthStatus: model.ProcessHealthActionRequired}

	var gotAgg, gotHealth []string
	stub := &stubFulfillmentReader{
		listProcesses: func(_ context.Context, tid uuid.UUID, agg, health []string, _, _ int) (service.ProcessListResult, error) {
			assert.Equal(t, tenantID, tid)
			gotAgg, gotHealth = agg, health
			return service.ProcessListResult{Items: []model.FulfillmentProcess{proc}, Total: 1}, nil
		},
	}
	h := NewFulfillmentHandler(stub)

	req := reqWithTenant(http.MethodGet, "/v1/fulfillment/processes?aggregate_status=blocked,waiting_external&health_status=action_required", tenantID, nil)
	rr := httptest.NewRecorder()
	h.ListProcesses(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, []string{"blocked", "waiting_external"}, gotAgg)
	assert.Equal(t, []string{"action_required"}, gotHealth)

	var resp model.ListResponse[model.FulfillmentProcess]
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, proc.ID, resp.Items[0].ID)
}

func TestFulfillmentHandler_ListProcesses_EmptyState(t *testing.T) {
	tenantID := uuid.New()
	h := NewFulfillmentHandler(&stubFulfillmentReader{}) // default returns empty result

	req := reqWithTenant(http.MethodGet, "/v1/fulfillment/processes", tenantID, nil)
	rr := httptest.NewRecorder()
	h.ListProcesses(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.ListResponse[model.FulfillmentProcess]
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Items)
}

func TestFulfillmentHandler_GetProcess_NotFound(t *testing.T) {
	tenantID := uuid.New()
	stub := &stubFulfillmentReader{
		getProcessDetail: func(_ context.Context, _, _ uuid.UUID) (*service.ProcessDetail, error) {
			return nil, service.ErrFulfillmentNotFound
		},
	}
	h := NewFulfillmentHandler(stub)

	req := reqWithTenant(http.MethodGet, "/v1/fulfillment/processes/"+uuid.NewString(), tenantID, map[string]string{"id": uuid.NewString()})
	rr := httptest.NewRecorder()
	h.GetProcess(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "fulfillment process not found", resp["error"])
}

func TestFulfillmentHandler_GetProcess_InvalidID(t *testing.T) {
	tenantID := uuid.New()
	h := NewFulfillmentHandler(&stubFulfillmentReader{})

	req := reqWithTenant(http.MethodGet, "/v1/fulfillment/processes/not-a-uuid", tenantID, map[string]string{"id": "not-a-uuid"})
	rr := httptest.NewRecorder()
	h.GetProcess(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFulfillmentHandler_GetProcess_HappyPath(t *testing.T) {
	tenantID := uuid.New()
	processID := uuid.New()
	detail := &service.ProcessDetail{
		Process:  model.FulfillmentProcess{ID: processID, TenantID: tenantID},
		Units:    []service.UnitDetail{},
		Blockers: []model.FulfillmentBlocker{},
		Attempts: []model.ProviderAttempt{},
	}
	stub := &stubFulfillmentReader{
		getProcessDetail: func(_ context.Context, tid, pid uuid.UUID) (*service.ProcessDetail, error) {
			assert.Equal(t, tenantID, tid)
			assert.Equal(t, processID, pid)
			return detail, nil
		},
	}
	h := NewFulfillmentHandler(stub)

	req := reqWithTenant(http.MethodGet, "/v1/fulfillment/processes/"+processID.String(), tenantID, map[string]string{"id": processID.String()})
	rr := httptest.NewRecorder()
	h.GetProcess(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp service.ProcessDetail
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, processID, resp.Process.ID)
}

func TestFulfillmentHandler_GetOrderFulfillment_NotFound(t *testing.T) {
	tenantID := uuid.New()
	orderID := uuid.New()
	h := NewFulfillmentHandler(&stubFulfillmentReader{}) // default returns ErrFulfillmentNotFound

	req := reqWithTenant(http.MethodGet, "/v1/fulfillment/orders/"+orderID.String(), tenantID, map[string]string{"orderID": orderID.String()})
	rr := httptest.NewRecorder()
	h.GetOrderFulfillment(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestFulfillmentHandler_ResolveBlocker_HappyPath(t *testing.T) {
	tenantID := uuid.New()
	blockerID := uuid.New()
	resolved := &model.FulfillmentBlocker{ID: blockerID, TenantID: tenantID, Status: model.BlockerStatusResolved}
	stub := &stubFulfillmentReader{
		resolveBlocker: func(_ context.Context, tid, bid, _ uuid.UUID, _ string) (*model.FulfillmentBlocker, error) {
			assert.Equal(t, tenantID, tid)
			assert.Equal(t, blockerID, bid)
			return resolved, nil
		},
	}
	h := NewFulfillmentHandler(stub)

	req := reqWithTenant(http.MethodPost, "/v1/fulfillment/blockers/"+blockerID.String()+"/resolve", tenantID, map[string]string{"id": blockerID.String()})
	rr := httptest.NewRecorder()
	h.ResolveBlocker(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.FulfillmentBlocker
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, model.BlockerStatusResolved, resp.Status)
}

func TestFulfillmentHandler_ResolveBlocker_NotFound(t *testing.T) {
	tenantID := uuid.New()
	blockerID := uuid.New()
	h := NewFulfillmentHandler(&stubFulfillmentReader{}) // default returns ErrFulfillmentNotFound

	req := reqWithTenant(http.MethodPost, "/v1/fulfillment/blockers/"+blockerID.String()+"/resolve", tenantID, map[string]string{"id": blockerID.String()})
	rr := httptest.NewRecorder()
	h.ResolveBlocker(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestFulfillmentHandler_RetryStep_ValidationError(t *testing.T) {
	tenantID := uuid.New()
	stepID := uuid.New()
	stub := &stubFulfillmentReader{
		retryStep: func(_ context.Context, _, _, _ uuid.UUID, _ string) (*model.FulfillmentStep, error) {
			return nil, service.NewValidationError(errors.New("step is not retryable"))
		},
	}
	h := NewFulfillmentHandler(stub)

	req := reqWithTenant(http.MethodPost, "/v1/fulfillment/steps/"+stepID.String()+"/retry", tenantID, map[string]string{"id": stepID.String()})
	rr := httptest.NewRecorder()
	h.RetryStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestFulfillmentHandler_RetryStep_HappyPath(t *testing.T) {
	tenantID := uuid.New()
	stepID := uuid.New()
	step := &model.FulfillmentStep{ID: stepID, TenantID: tenantID, Status: model.FulfillmentStatusPending}
	stub := &stubFulfillmentReader{
		retryStep: func(_ context.Context, tid, sid, _ uuid.UUID, _ string) (*model.FulfillmentStep, error) {
			assert.Equal(t, tenantID, tid)
			assert.Equal(t, stepID, sid)
			return step, nil
		},
	}
	h := NewFulfillmentHandler(stub)

	req := reqWithTenant(http.MethodPost, "/v1/fulfillment/steps/"+stepID.String()+"/retry", tenantID, map[string]string{"id": stepID.String()})
	rr := httptest.NewRecorder()
	h.RetryStep(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.FulfillmentStep
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, model.FulfillmentStatusPending, resp.Status)
}

func TestOperationsHandler_Summary_HappyPath(t *testing.T) {
	tenantID := uuid.New()
	stub := &stubFulfillmentReader{
		opsSummary: func(_ context.Context, tid uuid.UUID) (service.OperationsSummaryResult, error) {
			assert.Equal(t, tenantID, tid)
			return service.OperationsSummaryResult{
				Buckets: map[service.OperatorBucket]int{service.BucketBlocked: 2, service.BucketReady: 1},
				Total:   3,
			}, nil
		},
	}
	h := NewOperationsHandler(stub)

	req := reqWithTenant(http.MethodGet, "/v1/operations/summary", tenantID, nil)
	rr := httptest.NewRecorder()
	h.Summary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp service.OperationsSummaryResult
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.Buckets[service.BucketBlocked])
}

func TestOperationsHandler_Exceptions_EmptyState(t *testing.T) {
	tenantID := uuid.New()
	h := NewOperationsHandler(&stubFulfillmentReader{}) // default returns empty slice

	req := reqWithTenant(http.MethodGet, "/v1/operations/exceptions", tenantID, nil)
	rr := httptest.NewRecorder()
	h.Exceptions(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Items []service.ExceptionItem `json:"items"`
		Total int                     `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Items)
}

func TestOperationsHandler_Exceptions_PassesLimit(t *testing.T) {
	tenantID := uuid.New()
	var gotLimit int
	stub := &stubFulfillmentReader{
		opsExceptions: func(_ context.Context, _ uuid.UUID, limit int) ([]service.ExceptionItem, error) {
			gotLimit = limit
			return []service.ExceptionItem{{Bucket: service.BucketBlocked}}, nil
		},
	}
	h := NewOperationsHandler(stub)

	req := reqWithTenant(http.MethodGet, "/v1/operations/exceptions?limit=7", tenantID, nil)
	rr := httptest.NewRecorder()
	h.Exceptions(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 7, gotLimit)
}

func TestOperationsHandler_IntegrationCapabilitySummary_HappyPath(t *testing.T) {
	tenantID := uuid.New()
	stub := &stubFulfillmentReader{
		capabilitySummary: func(_ context.Context, tid uuid.UUID, _ int) (service.IntegrationCapabilitySummaryResult, error) {
			assert.Equal(t, tenantID, tid)
			return service.IntegrationCapabilitySummaryResult{AutomationCoverage: 0.75, TotalProcesses: 4}, nil
		},
	}
	h := NewOperationsHandler(stub)

	req := reqWithTenant(http.MethodGet, "/v1/operations/integration-capability-summary", tenantID, nil)
	rr := httptest.NewRecorder()
	h.IntegrationCapabilitySummary(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp service.IntegrationCapabilitySummaryResult
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.InDelta(t, 0.75, resp.AutomationCoverage, 0.0001)
	assert.Equal(t, 4, resp.TotalProcesses)
}
