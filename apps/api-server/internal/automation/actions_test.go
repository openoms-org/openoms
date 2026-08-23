package automation

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// --- fakes for the OPE-421 orchestration-routed set_status path ---

type fakeTransitioner struct {
	calls int
	reqs  []model.StatusTransitionRequest
}

func (f *fakeTransitioner) TransitionStatus(_ context.Context, _, _ uuid.UUID, req model.StatusTransitionRequest, _ uuid.UUID, _ string) (*model.Order, error) {
	f.calls++
	f.reqs = append(f.reqs, req)
	return &model.Order{}, nil
}

type fakeProcessEnsurer struct {
	getCalls    int
	createCalls int
	existing    *model.FulfillmentProcess // returned by GetProcessByOrder when non-nil
}

func (f *fakeProcessEnsurer) GetProcessByOrder(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.FulfillmentProcess, error) {
	f.getCalls++
	if f.existing != nil {
		return f.existing, nil
	}
	return nil, pgx.ErrNoRows
}

func (f *fakeProcessEnsurer) CreateProcess(_ context.Context, _ pgx.Tx, p model.FulfillmentProcess) (*model.FulfillmentProcess, error) {
	f.createCalls++
	p.ID = uuid.New()
	return &p, nil
}

type fakeEnqueuer struct {
	calls  int
	events []model.OrchestrationOutboxEvent
}

func (f *fakeEnqueuer) EnqueueEvent(_ context.Context, _ repository.Querier, e model.OrchestrationOutboxEvent) (bool, *model.OrchestrationOutboxEvent, error) {
	f.calls++
	f.events = append(f.events, e)
	return true, &e, nil
}

func TestDefaultActionExecutor_ActivateListing_NilDeps(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// No listing deps wired — should return nil (skip gracefully)
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "activate_listing"}, Event{
		Type:       "product.stock_restored",
		TenantID:   uuid.New(),
		EntityType: "product",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "activate_listing with nil deps should not error")
}

func TestDefaultActionExecutor_ActivateListing_WrongEntityType(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetListingActivatorDeps(&ListingActivatorDeps{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "activate_listing"}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err, "activate_listing with order entity should error")
	assert.Contains(t, err.Error(), "only supports product entities")
}

func TestDefaultActionExecutor_DeactivateListing_NilDeps(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// No listing deactivator deps wired — should return nil (skip gracefully)
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "deactivate_listing"}, Event{
		Type:       "product.out_of_stock",
		TenantID:   uuid.New(),
		EntityType: "product",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "deactivate_listing with nil deps should not error")
}

func TestDefaultActionExecutor_DeactivateListing_WrongEntityType(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetListingDeactivatorDeps(&ListingDeactivatorDeps{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "deactivate_listing"}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err, "deactivate_listing with order entity should error")
	assert.Contains(t, err.Error(), "only supports product entities")
}

func TestDefaultActionExecutor_UnknownAction(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{Type: "nonexistent"}, Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action type")
}

func TestDefaultActionExecutor_SetStatus_NilTransitioner(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "set_status",
		Params: map[string]any{"status": "confirmed"},
	}, Event{
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	// With nil transitioner, should return nil (skip gracefully)
	assert.NoError(t, err)
}

func TestDefaultActionExecutor_WebhookMissingURL(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "webhook",
		Params: map[string]any{},
	}, Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing url")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_NilDeps(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// No marketplace message deps wired — should return nil (skip gracefully)
	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": uuid.New().String()},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "send_marketplace_message with nil deps should not error")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_NilPool(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// Deps set but Pool is nil — should skip gracefully.
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": uuid.New().String()},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.NoError(t, err, "send_marketplace_message with nil Pool should skip gracefully")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_WrongEntityType(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	// Need a non-nil Pool to pass the deps check, but it won't actually be used
	// because the entity type check comes first.
	executor.pool = &pgxpool.Pool{} // Set the field directly (package-internal test)
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{
		Pool: &pgxpool.Pool{},
	})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": uuid.New().String()},
	}, Event{
		Type:       "product.stock_restored",
		TenantID:   uuid.New(),
		EntityType: "product",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err, "send_marketplace_message with product entity should error")
	assert.Contains(t, err.Error(), "only supports order entities")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_MissingTemplateID(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{
		Pool: &pgxpool.Pool{},
	})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'template_id'")
}

func TestDefaultActionExecutor_SendMarketplaceMessage_InvalidTemplateID(t *testing.T) {
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetMarketplaceMessageDeps(&MarketplaceMessageDeps{
		Pool: &pgxpool.Pool{},
	})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "send_marketplace_message",
		Params: map[string]any{"template_id": "not-a-uuid"},
	}, Event{
		Type:       "order.status_changed",
		TenantID:   uuid.New(),
		EntityType: "order",
		EntityID:   uuid.New(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template_id")
}

func TestSubstituteVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		expected string
	}{
		{
			name:     "simple substitution",
			template: "Hello {{customer_name}}, your order {{order_id}} is ready.",
			vars:     map[string]string{"customer_name": "Jan", "order_id": "ORD-123"},
			expected: "Hello Jan, your order ORD-123 is ready.",
		},
		{
			name:     "no variables",
			template: "Hello, your order is ready.",
			vars:     map[string]string{"customer_name": "Jan"},
			expected: "Hello, your order is ready.",
		},
		{
			name:     "missing variable stays",
			template: "Hello {{customer_name}}, tracking: {{tracking_number}}.",
			vars:     map[string]string{"customer_name": "Jan"},
			expected: "Hello Jan, tracking: {{tracking_number}}.",
		},
		{
			name:     "empty vars map",
			template: "Hello {{customer_name}}!",
			vars:     map[string]string{},
			expected: "Hello {{customer_name}}!",
		},
		{
			name:     "multiple occurrences",
			template: "{{name}} ordered. Dear {{name}}, thank you.",
			vars:     map[string]string{"name": "Anna"},
			expected: "Anna ordered. Dear Anna, thank you.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVariables(tt.template, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSetStatusIdempotencyKey verifies the stable (rule, order, target) dedup key.
func TestSetStatusIdempotencyKey(t *testing.T) {
	ruleID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	orderID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	key := setStatusIdempotencyKey(ruleID, orderID, "confirmed")
	assert.Equal(t, "automation.set_status:"+ruleID.String()+":"+orderID.String()+":confirmed", key)
	// Same (rule, order, target) -> identical key (dedup); different target -> different key.
	assert.Equal(t, key, setStatusIdempotencyKey(ruleID, orderID, "confirmed"))
	assert.NotEqual(t, key, setStatusIdempotencyKey(ruleID, orderID, "shipped"))
}

// TestExecuteSetStatus_OrchestrationDisabled_UsesDirectPath verifies the default
// (flag off): set_status calls TransitionStatus directly and never enqueues.
func TestExecuteSetStatus_OrchestrationDisabled_UsesDirectPath(t *testing.T) {
	tr := &fakeTransitioner{}
	enq := &fakeEnqueuer{}
	ens := &fakeProcessEnsurer{}
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(tr, nil, nil, &pgxpool.Pool{})
	// Routing explicitly disabled (matching the zero value).
	executor.SetOrchestration(false, enq, ens)

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "set_status",
		Params: map[string]any{"status": "confirmed"},
	}, Event{EntityType: "order", EntityID: uuid.New()})

	require.NoError(t, err)
	assert.Equal(t, 1, tr.calls, "disabled path calls TransitionStatus directly")
	assert.Equal(t, 0, enq.calls, "disabled path never enqueues")
	assert.Equal(t, 0, ens.getCalls, "disabled path does no fulfillment-process work")
}

// TestExecuteSetStatus_OrchestrationEnabled_Enqueues exercises the tx-bound
// orchestration path directly (with fakes, no DB): it ensures a process and
// enqueues exactly one automation.set_status event with the right type/key/payload,
// and the transitioner is NOT invoked.
func TestExecuteSetStatus_OrchestrationEnabled_Enqueues(t *testing.T) {
	tr := &fakeTransitioner{}
	enq := &fakeEnqueuer{}
	ens := &fakeProcessEnsurer{} // empty -> GetProcessByOrder returns ErrNoRows -> CreateProcess
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(tr, nil, nil, &pgxpool.Pool{})
	executor.SetOrchestration(true, enq, ens)

	ruleID := uuid.New()
	orderID := uuid.New()
	tenantID := uuid.New()
	payload := map[string]any{
		"order_id":      orderID.String(),
		"target_status": "confirmed",
		"rule_id":       ruleID.String(),
		"action_index":  0,
	}
	idemKey := setStatusIdempotencyKey(ruleID, orderID, "confirmed")

	// Call the tx-bound helper directly (the WithTenant wrapper needs a real pool).
	err := executor.ensureProcessAndEnqueueSetStatus(context.Background(), nil, tenantID, orderID, idemKey, payload)
	require.NoError(t, err)

	assert.Equal(t, 0, tr.calls, "orchestration path must NOT call TransitionStatus")
	assert.Equal(t, 1, ens.getCalls, "looks up the process once")
	assert.Equal(t, 1, ens.createCalls, "creates the process when none exists")
	require.Len(t, enq.events, 1, "enqueues exactly one event")
	ev := enq.events[0]
	assert.Equal(t, EventAutomationSetStatus, ev.EventType)
	assert.Equal(t, idemKey, ev.IdempotencyKey)
	assert.Equal(t, tenantID, ev.TenantID)
	assert.NotEqual(t, uuid.Nil, ev.ProcessID, "event is anchored to a process")
}

// TestExecuteSetStatus_OrchestrationEnabled_ReusesExistingProcess verifies the
// ensure-process step is idempotent: an existing process is reused, not recreated.
func TestExecuteSetStatus_OrchestrationEnabled_ReusesExistingProcess(t *testing.T) {
	enq := &fakeEnqueuer{}
	existing := &model.FulfillmentProcess{ID: uuid.New()}
	ens := &fakeProcessEnsurer{existing: existing}
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(&fakeTransitioner{}, nil, nil, &pgxpool.Pool{})
	executor.SetOrchestration(true, enq, ens)

	err := executor.ensureProcessAndEnqueueSetStatus(context.Background(), nil, uuid.New(), uuid.New(), "k", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, 1, ens.getCalls)
	assert.Equal(t, 0, ens.createCalls, "existing process is reused, not recreated")
	require.Len(t, enq.events, 1)
	assert.Equal(t, existing.ID, enq.events[0].ProcessID)
}

// TestExecuteSetStatus_OrchestrationEnabled_LookupErrorPropagates verifies a
// non-ErrNoRows lookup failure surfaces to the caller (engine logs it) rather than
// being swallowed.
func TestExecuteSetStatus_OrchestrationEnabled_LookupErrorPropagates(t *testing.T) {
	enq := &fakeEnqueuer{}
	ens := &errLookupEnsurer{err: errors.New("boom")}
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(&fakeTransitioner{}, nil, nil, &pgxpool.Pool{})
	executor.SetOrchestration(true, enq, ens)

	err := executor.ensureProcessAndEnqueueSetStatus(context.Background(), nil, uuid.New(), uuid.New(), "k", map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup fulfillment process")
	assert.Equal(t, 0, enq.calls, "no event enqueued when the process lookup fails")
}

// TestExecuteSetStatus_DefaultRespectsOrderGraph pins the CORR-07 contract: a
// set_status action without an explicit opt-in must NOT force the transition, so
// OrderService validates it against the tenant's status graph and an illegal jump
// (e.g. new -> completed) is rejected instead of silently applied.
func TestExecuteSetStatus_DefaultRespectsOrderGraph(t *testing.T) {
	tr := &fakeTransitioner{}
	executor := NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(tr, nil, nil, &pgxpool.Pool{})

	err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
		Type:   "set_status",
		Params: map[string]any{"status": "completed"},
	}, Event{EntityType: "order", EntityID: uuid.New()})

	require.NoError(t, err)
	require.Len(t, tr.reqs, 1)
	assert.False(t, tr.reqs[0].Force, "set_status must not force by default")
	assert.Equal(t, "completed", tr.reqs[0].Status)
}

// TestExecuteSetStatus_ForceParamOptsIn verifies a rule can still bypass the graph,
// but only by explicitly asking for it. The dashboard stores action params as JSON,
// so a boolean and its string form must both opt in.
func TestExecuteSetStatus_ForceParamOptsIn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		force any
	}{
		{"json bool", true},
		{"string form", "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &fakeTransitioner{}
			executor := NewDefaultActionExecutor(slog.Default())
			executor.SetOrderServices(tr, nil, nil, &pgxpool.Pool{})

			err := executor.ExecuteAction(context.Background(), uuid.New(), Action{
				Type:   "set_status",
				Params: map[string]any{"status": "completed", "force": tc.force},
			}, Event{EntityType: "order", EntityID: uuid.New()})

			require.NoError(t, err)
			require.Len(t, tr.reqs, 1)
			assert.True(t, tr.reqs[0].Force, "explicit force param opts into bypassing the graph")
		})
	}
}

// TestExecuteSetStatus_ForceParamNotOptIn covers the values that must leave the
// graph check in place — absent, false, and anything non-boolean.
func TestExecuteSetStatus_ForceParamNotOptIn(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params map[string]any
	}{
		{"absent", map[string]any{"status": "completed"}},
		{"json false", map[string]any{"status": "completed", "force": false}},
		{"string false", map[string]any{"status": "completed", "force": "false"}},
		{"garbage", map[string]any{"status": "completed", "force": "yes please"}},
		{"number", map[string]any{"status": "completed", "force": 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &fakeTransitioner{}
			executor := NewDefaultActionExecutor(slog.Default())
			executor.SetOrderServices(tr, nil, nil, &pgxpool.Pool{})

			require.NoError(t, executor.ExecuteAction(context.Background(), uuid.New(), Action{
				Type:   "set_status",
				Params: tc.params,
			}, Event{EntityType: "order", EntityID: uuid.New()}))
			require.Len(t, tr.reqs, 1)
			assert.False(t, tr.reqs[0].Force)
		})
	}
}

// TestExecuteSetStatus_OrchestrationEnabled_PayloadCarriesForce verifies the
// orchestration-routed path does not lose the opt-in: the flag has to survive the
// outbox round-trip, otherwise the async handler would silently re-force.
func TestExecuteSetStatus_OrchestrationEnabled_PayloadCarriesForce(t *testing.T) {
	for _, tc := range []struct {
		name   string
		params map[string]any
		want   bool
	}{
		{"default", map[string]any{"status": "completed"}, false},
		{"opt-in", map[string]any{"status": "completed", "force": true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enq := &fakeEnqueuer{}
			ens := &fakeProcessEnsurer{}
			executor := NewDefaultActionExecutor(slog.Default())
			executor.SetOrderServices(&fakeTransitioner{}, nil, nil, &pgxpool.Pool{})
			executor.SetOrchestration(true, enq, ens)

			orderID := uuid.New()
			payload := setStatusPayload(orderID, "completed", uuid.New(), 0, paramBool(tc.params["force"]))
			require.NoError(t, executor.ensureProcessAndEnqueueSetStatus(
				context.Background(), nil, uuid.New(), orderID, "k", payload))

			require.Len(t, enq.events, 1)
			got, ok := enq.events[0].Payload.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tc.want, got["force"], "force flag is carried in the outbox payload")
		})
	}
}

type errLookupEnsurer struct{ err error }

func (e *errLookupEnsurer) GetProcessByOrder(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.FulfillmentProcess, error) {
	return nil, e.err
}

func (e *errLookupEnsurer) CreateProcess(_ context.Context, _ pgx.Tx, p model.FulfillmentProcess) (*model.FulfillmentProcess, error) {
	return &p, nil
}
