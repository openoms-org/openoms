//go:build integration

package integration

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/automation"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
)

// newSetStatusOrderService builds a minimal real OrderService for the set_status
// orchestration integration tests. emailService/webhookDispatch are nil (the
// service guards them) and fulfillment is the disabled no-op service — the
// orchestration routing under test lives in the automation executor, not here.
func newSetStatusOrderService(pool *pgxpool.Pool) *service.OrderService {
	orderRepo := repository.NewOrderRepository()
	auditRepo := repository.NewAuditRepository()
	tenantRepo := repository.NewTenantRepository(pool)
	fulfillment := service.NewFulfillmentService(false,
		repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository())
	return service.NewOrderService(orderRepo, auditRepo, tenantRepo, pool, nil, nil, fulfillment)
}

// seedSetStatusOrder inserts a fresh order (status 'new') for tenant and returns its id.
func seedSetStatusOrder(t *testing.T, ctx context.Context, tenant uuid.UUID, name string) uuid.UUID {
	t.Helper()
	orderID := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1,$2,$3)`, orderID, tenant, name)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", orderID) })
	return orderID
}

// TestAutomationSetStatus_OrchestrationEnabled_EnqueueDispatchIdempotent is the
// end-to-end OPE-421 flow on the ON path: a set_status action enqueues exactly one
// automation.set_status outbox event (and ensures a fulfillment process), the
// orchestration worker drains it and the order transitions, and a re-fire of the
// same (rule, order, target) is a no-op.
func TestAutomationSetStatus_OrchestrationEnabled_EnqueueDispatchIdempotent(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderID := seedSetStatusOrder(t, ctx, tenant, "SetStatus Orch Customer")

	orderSvc := newSetStatusOrderService(appPool)
	orderRepo := repository.NewOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()

	executor := automation.NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(orderSvc, orderSvc, orderRepo, appPool)
	executor.SetOrchestration(true, orchRepo, fRepo)

	ruleID := uuid.New()
	action := automation.Action{
		Type:        "set_status",
		Params:      map[string]any{"status": "confirmed"},
		RuleID:      ruleID,
		ActionIndex: 0,
	}
	event := automation.Event{Type: "order.status_changed", TenantID: tenant, EntityType: "order", EntityID: orderID}

	// --- Enqueue: action enqueues exactly one event, ensures a process, does NOT transition ---
	require.NoError(t, executor.ExecuteAction(ctx, tenant, action, event))

	var procID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		proc, e := fRepo.GetProcessByOrder(ctx, tx, orderID)
		if e != nil {
			return e
		}
		procID = proc.ID
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1, "exactly one automation.set_status event enqueued")
		assert.Equal(t, automation.EventAutomationSetStatus, events[0].EventType)
		assert.Equal(t, model.OutboxStatusPending, events[0].Status)
		assert.Equal(t,
			automation.EventAutomationSetStatus+":"+ruleID.String()+":"+orderID.String()+":confirmed",
			events[0].IdempotencyKey)
		return nil
	}))

	// Order is still 'new' — the transition has not happened yet (it is async).
	assertOrderStatus(t, ctx, orderID, "new")

	// --- Re-fire the same action: idempotent, no second event ---
	require.NoError(t, executor.ExecuteAction(ctx, tenant, action, event))
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		assert.Len(t, events, 1, "idempotent: re-fire enqueues no duplicate event")
		return nil
	}))

	// --- Dispatch: the worker drains the outbox and the handler transitions the order ---
	disp := service.NewOrchestrationDispatcher()
	disp.Register(automation.EventAutomationSetStatus, service.NewAutomationStatusTransitionHandler(orderSvc))
	w := worker.NewOrchestrationWorker(superPool, orchRepo, disp, fRepo, time.Second, slog.Default())
	require.NoError(t, w.Run(ctx))

	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, model.OutboxStatusSucceeded, events[0].Status, "event processed successfully")
		return nil
	}))
	assertOrderStatus(t, ctx, orderID, "confirmed")

	// Handler is idempotent: re-enqueue + re-dispatch leaves the order untouched and
	// does not error (already-in-target is a no-op).
	require.NoError(t, executor.ExecuteAction(ctx, tenant, action, event))
	require.NoError(t, w.Run(ctx))
	assertOrderStatus(t, ctx, orderID, "confirmed")
}

// TestAutomationSetStatus_OrchestrationEnabled_BadStatusOpensBlocker verifies a
// permanent failure (unknown target status) marks the event failed and opens an
// automation_action_failed blocker on the process.
func TestAutomationSetStatus_OrchestrationEnabled_BadStatusOpensBlocker(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderID := seedSetStatusOrder(t, ctx, tenant, "SetStatus Bad Customer")

	orderSvc := newSetStatusOrderService(appPool)
	orderRepo := repository.NewOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()

	executor := automation.NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(orderSvc, orderSvc, orderRepo, appPool)
	executor.SetOrchestration(true, orchRepo, fRepo)

	action := automation.Action{
		Type:   "set_status",
		Params: map[string]any{"status": "this-status-does-not-exist"},
		RuleID: uuid.New(),
	}
	event := automation.Event{Type: "order.status_changed", TenantID: tenant, EntityType: "order", EntityID: orderID}
	require.NoError(t, executor.ExecuteAction(ctx, tenant, action, event))

	var procID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		proc, e := fRepo.GetProcessByOrder(ctx, tx, orderID)
		if e == nil {
			procID = proc.ID
		}
		return e
	}))

	disp := service.NewOrchestrationDispatcher()
	disp.Register(automation.EventAutomationSetStatus, service.NewAutomationStatusTransitionHandler(orderSvc))
	w := worker.NewOrchestrationWorker(superPool, orchRepo, disp, fRepo, time.Second, slog.Default())
	require.NoError(t, w.Run(ctx))

	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, model.OutboxStatusFailed, events[0].Status, "unknown status -> permanent failure")

		blockers, e := fRepo.ListBlockers(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, blockers, 1, "permanent failure opens exactly one blocker")
		assert.Equal(t, model.BlockerAutomationActionFailed, blockers[0].Code)
		assert.Equal(t, "automation", blockers[0].Category)
		return nil
	}))

	// Order never transitioned.
	assertOrderStatus(t, ctx, orderID, "new")
}

// stubTransitioner records TransitionStatus calls so the disabled-path test can
// prove the direct branch ran without depending on OrderService internals (the
// legacy direct path passes "automation" as the IP, which is not a valid inet and
// fails the audit insert — a pre-existing quirk unrelated to OPE-421 routing).
type stubTransitioner struct {
	calls    int
	lastReq  model.StatusTransitionRequest
	lastIP   string
	retOrder *model.Order
}

func (s *stubTransitioner) TransitionStatus(_ context.Context, _, orderID uuid.UUID, req model.StatusTransitionRequest, _ uuid.UUID, ip string) (*model.Order, error) {
	s.calls++
	s.lastReq = req
	s.lastIP = ip
	s.retOrder = &model.Order{ID: orderID, Status: req.Status}
	return s.retOrder, nil
}

func (s *stubTransitioner) Get(_ context.Context, _, orderID uuid.UUID) (*model.Order, error) {
	return &model.Order{ID: orderID, Status: "new"}, nil
}

// TestAutomationSetStatus_OrchestrationDisabled_DirectPath verifies the OFF path:
// set_status calls TransitionStatus directly (Force=true) and writes ZERO outbox
// rows and no fulfillment process — the orchestration machinery is untouched.
func TestAutomationSetStatus_OrchestrationDisabled_DirectPath(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	orderID := seedSetStatusOrder(t, ctx, tenant, "SetStatus Direct Customer")

	tr := &stubTransitioner{}
	orderRepo := repository.NewOrderRepository()
	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()

	executor := automation.NewDefaultActionExecutor(slog.Default())
	executor.SetOrderServices(tr, tr, orderRepo, appPool)
	// Routing wired but explicitly DISABLED — the direct path must run.
	executor.SetOrchestration(false, orchRepo, fRepo)

	action := automation.Action{Type: "set_status", Params: map[string]any{"status": "confirmed"}, RuleID: uuid.New()}
	event := automation.Event{Type: "order.status_changed", TenantID: tenant, EntityType: "order", EntityID: orderID}
	require.NoError(t, executor.ExecuteAction(ctx, tenant, action, event))

	// Direct transition path was taken (Force=true), exactly once.
	assert.Equal(t, 1, tr.calls, "disabled path calls TransitionStatus directly")
	assert.Equal(t, "confirmed", tr.lastReq.Status)
	assert.True(t, tr.lastReq.Force, "automation transitions force")

	// No fulfillment process and no outbox rows were created — orchestration untouched.
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		_, e := fRepo.GetProcessByOrder(ctx, tx, orderID)
		assert.True(t, errors.Is(e, pgx.ErrNoRows), "disabled path creates no fulfillment process")
		return nil
	}))
	var outboxCount int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orchestration_outbox WHERE tenant_id = $1`, tenant).Scan(&outboxCount))
	assert.Equal(t, 0, outboxCount, "disabled path enqueues zero outbox rows")
}

// assertOrderStatus reads the order status via the superuser pool and asserts it.
func assertOrderStatus(t *testing.T, ctx context.Context, orderID uuid.UUID, want string) {
	t.Helper()
	var status string
	require.NoError(t, superPool.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status))
	assert.Equal(t, want, status)
}
