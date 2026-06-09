//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/netutil"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
)

const ewfSigningSecret = "ewf-test-signing-secret"

// seedEWFIntegration inserts an external_workflow integration for tenant and returns its id.
func seedEWFIntegration(t *testing.T, ctx context.Context, tenant uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO integrations (id, tenant_id, provider, status) VALUES ($1,$2,'external_workflow','active')`,
		id, tenant)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM integrations WHERE id = $1", id) })
	return id
}

// seedEWFOrder inserts an order for tenant and returns its id.
func seedEWFOrder(t *testing.T, ctx context.Context, tenant uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx, `INSERT INTO orders (id, tenant_id, customer_name, status) VALUES ($1,$2,$3,'new')`,
		id, tenant, "EWF Customer")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id) })
	return id
}

// newEWFService builds an ExternalWorkflowService for the integration tests with a fixed
// config loader (the signing secret) and order loader.
func newEWFService(enabled bool) *service.ExternalWorkflowService {
	fulfillment := service.NewFulfillmentService(false,
		repository.NewFulfillmentRepository(), repository.NewOrchestrationRepository()).
		WithRecording(appPool, repository.NewFulfillmentAttemptRepository())
	svc := service.NewExternalWorkflowService(enabled, appPool, superPool, fulfillment,
		repository.NewOrchestrationRepository(), repository.NewExternalWorkflowTokenRepository(),
		repository.NewAuditRepository())
	svc.SetConfigLoader(func(_ context.Context, _, _ uuid.UUID) (service.ExternalWorkflowConfig, error) {
		return service.ExternalWorkflowConfig{SigningSecret: ewfSigningSecret}, nil
	})
	orderRepo := repository.NewOrderRepository()
	svc.SetOrderLoader(func(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.Order, error) {
		return orderRepo.FindByID(ctx, tx, orderID)
	})
	return svc
}

// issueEWFToken issues a token for the integration with the given scopes and returns the raw
// token plus its stored model (with the hash already set).
func issueEWFToken(t *testing.T, ctx context.Context, tenant, integrationID uuid.UUID, scopes []string) (string, *model.ExternalWorkflowToken) {
	t.Helper()
	raw := uuid.NewString() + uuid.NewString()
	repo := repository.NewExternalWorkflowTokenRepository()
	var stored *model.ExternalWorkflowToken
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		var e error
		stored, e = repo.Issue(ctx, tx, model.ExternalWorkflowToken{
			TenantID:      tenant,
			IntegrationID: integrationID,
			TokenHash:     model.HashExternalWorkflowToken(raw),
			Scopes:        scopes,
		})
		return e
	}))
	return raw, stored
}

// enqueueEWFEvent ensures a process for the order and enqueues an EventExternalWorkflow event
// with the given nonce + integration. Returns the process id.
func enqueueEWFEvent(t *testing.T, ctx context.Context, tenant, orderID, integrationID uuid.UUID, nonce string) uuid.UUID {
	t.Helper()
	fRepo := repository.NewFulfillmentRepository()
	orchRepo := repository.NewOrchestrationRepository()
	var procID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		proc, e := fRepo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenant, OrderID: orderID})
		if e != nil {
			return e
		}
		procID = proc.ID
		payload := map[string]any{
			"integration_id":    integrationID.String(),
			"correlation_nonce": nonce,
			"order_id":          orderID.String(),
			"redacted_payload":  map[string]any{"id": orderID.String(), "status": "new"},
		}
		_, _, e = orchRepo.EnqueueEvent(ctx, tx, model.OrchestrationOutboxEvent{
			TenantID:       tenant,
			ProcessID:      procID,
			EventType:      service.EventExternalWorkflow,
			IdempotencyKey: service.EventExternalWorkflow + ":" + orderID.String() + ":" + nonce,
			Payload:        payload,
		})
		return e
	}))
	return procID
}

// signedCallback builds a properly-signed callback body + signature header.
func signedCallback(t *testing.T, req model.ExternalWorkflowCallbackRequest) ([]byte, string) {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	return body, model.SignExternalWorkflowBody(body, ewfSigningSecret)
}

// TestExternalWorkflow_TokenHashedLookup verifies token issue + cross-tenant hash lookup, and
// that an unknown hash returns ErrExternalWorkflowTokenNotFound.
func TestExternalWorkflow_TokenHashedLookup(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	integrationID := seedEWFIntegration(t, ctx, tenant)
	raw, stored := issueEWFToken(t, ctx, tenant, integrationID, []string{"add_tag"})

	repo := repository.NewExternalWorkflowTokenRepository()
	got, err := repo.FindByHashCrossTenant(ctx, superPool, model.HashExternalWorkflowToken(raw))
	require.NoError(t, err)
	assert.Equal(t, stored.ID, got.ID)
	assert.Equal(t, tenant, got.TenantID)
	assert.Equal(t, integrationID, got.IntegrationID)
	assert.Equal(t, []string{"add_tag"}, got.Scopes)

	_, err = repo.FindByHashCrossTenant(ctx, superPool, model.HashExternalWorkflowToken("does-not-exist"))
	assert.ErrorIs(t, err, repository.ErrExternalWorkflowTokenNotFound)
}

// TestExternalWorkflow_CallbackResolvesAndFollowOn seeds an order + event and resolves it via a
// signed callback carrying an in-scope add_tag command: the event becomes succeeded, exactly one
// command event is enqueued, and an audit row exists. A replay of the same nonce is rejected.
func TestExternalWorkflow_CallbackResolvesAndFollowOn(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	integrationID := seedEWFIntegration(t, ctx, tenant)
	orderID := seedEWFOrder(t, ctx, tenant)
	raw, _ := issueEWFToken(t, ctx, tenant, integrationID, []string{"add_tag"})
	nonce := uuid.NewString()
	procID := enqueueEWFEvent(t, ctx, tenant, orderID, integrationID, nonce)

	svc := newEWFService(true)
	req := model.ExternalWorkflowCallbackRequest{
		CorrelationNonce: nonce, Status: "succeeded", ExternalExecID: "exec-123",
		Command: "add_tag", CommandValue: "from-engine",
	}
	body, sig := signedCallback(t, req)

	gotTenant, err := svc.ResolveCallbackHTTP(ctx, model.HashExternalWorkflowToken(raw), sig, body, req, "1.2.3.4", time.Now())
	require.NoError(t, err)
	assert.Equal(t, tenant, gotTenant)

	orchRepo := repository.NewOrchestrationRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		var dispatch, command int
		for _, ev := range events {
			switch ev.EventType {
			case service.EventExternalWorkflow:
				dispatch++
				assert.Equal(t, model.OutboxStatusSucceeded, ev.Status, "correlated event resolved succeeded")
			case service.EventExternalWorkflowCommand:
				command++
				assert.Equal(t, model.OutboxStatusPending, ev.Status)
			}
		}
		assert.Equal(t, 1, dispatch, "exactly one dispatch event")
		assert.Equal(t, 1, command, "exactly one in-scope follow-on command enqueued")
		return nil
	}))

	// Audit row exists.
	var auditCount int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_log WHERE tenant_id = $1 AND action = 'external_workflow.callback'`, tenant).
		Scan(&auditCount))
	assert.Equal(t, 1, auditCount, "callback writes one audit row")

	// External execution id recorded on the latest attempt is not required here (no attempt row
	// was created by a real worker run); the SetLatestAttemptExternalExecID is best-effort.

	// --- Replay: same nonce -> not resolvable (single-use) ---
	_, err = svc.ResolveCallbackHTTP(ctx, model.HashExternalWorkflowToken(raw), sig, body, req, "1.2.3.4", time.Now())
	assert.ErrorIs(t, err, service.ErrCorrelationNotResolvable, "consumed nonce rejected on replay")
}

// TestExternalWorkflow_OutOfScopeCommandRejected verifies a command outside the token scope is
// rejected with "not permitted" while the correlated event is still resolved.
func TestExternalWorkflow_OutOfScopeCommandRejected(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	integrationID := seedEWFIntegration(t, ctx, tenant)
	orderID := seedEWFOrder(t, ctx, tenant)
	raw, _ := issueEWFToken(t, ctx, tenant, integrationID, []string{"add_tag"}) // NOT set_status
	nonce := uuid.NewString()
	procID := enqueueEWFEvent(t, ctx, tenant, orderID, integrationID, nonce)

	svc := newEWFService(true)
	req := model.ExternalWorkflowCallbackRequest{
		CorrelationNonce: nonce, Status: "succeeded", Command: "set_status", CommandValue: "confirmed",
	}
	body, sig := signedCallback(t, req)

	_, err := svc.ResolveCallbackHTTP(ctx, model.HashExternalWorkflowToken(raw), sig, body, req, "1.2.3.4", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not permitted")

	// The event WAS marked succeeded before the command was evaluated and rejected — but because
	// the resolution runs in one transaction and the command rejection returns an error, the
	// whole tx rolls back: the event must remain pending so a corrected callback can retry.
	orchRepo := repository.NewOrchestrationRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, model.OutboxStatusPending, events[0].Status,
			"rejected-command callback rolls back: event stays pending")
		return nil
	}))
}

// TestExternalWorkflow_BadSignatureRejected verifies a wrong HMAC is rejected, and an unknown
// token is rejected, without resolving the event.
func TestExternalWorkflow_BadSignatureRejected(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	integrationID := seedEWFIntegration(t, ctx, tenant)
	orderID := seedEWFOrder(t, ctx, tenant)
	raw, _ := issueEWFToken(t, ctx, tenant, integrationID, nil)
	nonce := uuid.NewString()
	enqueueEWFEvent(t, ctx, tenant, orderID, integrationID, nonce)

	svc := newEWFService(true)
	req := model.ExternalWorkflowCallbackRequest{CorrelationNonce: nonce, Status: "succeeded"}
	body, _ := signedCallback(t, req)

	// Wrong signature.
	_, err := svc.ResolveCallbackHTTP(ctx, model.HashExternalWorkflowToken(raw), "sha256=deadbeef", body, req, "1.2.3.4", time.Now())
	assert.ErrorIs(t, err, service.ErrExternalWorkflowBadSignature)

	// Unknown token.
	_, sig := signedCallback(t, req)
	_, err = svc.ResolveCallbackHTTP(ctx, model.HashExternalWorkflowToken("unknown-token"), sig, body, req, "1.2.3.4", time.Now())
	assert.ErrorIs(t, err, repository.ErrExternalWorkflowTokenNotFound)
}

// TestExternalWorkflow_CrossTenantRLS verifies tenant B cannot see tenant A's tokens through a
// tenant-scoped (WithTenant) query.
func TestExternalWorkflow_CrossTenantRLS(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)
	integrationA := seedEWFIntegration(t, ctx, tenantA)
	issueEWFToken(t, ctx, tenantA, integrationA, []string{"add_tag"})

	// Tenant B's scoped count excludes A's token.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT COUNT(*) FROM external_workflow_tokens`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 0, n, "tenant B sees zero of tenant A's tokens under RLS")
		return nil
	}))
	// Tenant A's scoped count sees its own.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT COUNT(*) FROM external_workflow_tokens`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "tenant A sees its own token under RLS")
		return nil
	}))
}

// TestExternalWorkflow_TimeoutOpensBlocker drives the dispatch handler + worker: the first
// dispatch POSTs (200) and parks the event pending-callback; after the deadline passes with no
// callback, the worker re-dispatches and the handler (Attempts>0) fails it permanently, opening
// an external_workflow_timeout blocker.
func TestExternalWorkflow_TimeoutOpensBlocker(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	integrationID := seedEWFIntegration(t, ctx, tenant)
	orderID := seedEWFOrder(t, ctx, tenant)
	nonce := uuid.NewString()
	procID := enqueueEWFEvent(t, ctx, tenant, orderID, integrationID, nonce)

	// A stub external engine returning 200 OK so the first dispatch succeeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Config loader returns the stub URL + a 1s timeout (so the deadline passes quickly).
	loadCfg := func(_ context.Context, _, _ uuid.UUID) (service.ExternalWorkflowConfig, error) {
		return service.ExternalWorkflowConfig{
			OutboundURL: srv.URL, SigningSecret: ewfSigningSecret, TimeoutSecs: 1, Criticality: "blocker",
		}, nil
	}
	handler := service.NewExternalWorkflowHandler(netutil.SafeHTTPClient(5*time.Second), loadCfg)
	disp := service.NewOrchestrationDispatcher()
	disp.Register(service.EventExternalWorkflow, handler)

	orchRepo := repository.NewOrchestrationRepository()
	fRepo := repository.NewFulfillmentRepository()
	w := worker.NewOrchestrationWorker(superPool, orchRepo, disp, fRepo, time.Second, slog.Default())

	// First run: dispatch succeeds -> event parked pending-callback with next_attempt_at in the
	// future, attempts incremented.
	require.NoError(t, w.Run(ctx))
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, model.OutboxStatusPending, events[0].Status, "parked pending-callback")
		assert.GreaterOrEqual(t, events[0].Attempts, 1, "attempts incremented on park")
		return nil
	}))

	// Force the deadline into the past (simulate timeout with no callback).
	dispatchID := firstOutboxIDForType(t, ctx, procID, service.EventExternalWorkflow)
	_, err := superPool.Exec(ctx,
		`UPDATE orchestration_outbox SET next_attempt_at = now() - interval '1 minute' WHERE id = $1`, dispatchID)
	require.NoError(t, err)

	// Second run: re-dispatch past the deadline -> Attempts>0 -> permanent -> timeout blocker.
	require.NoError(t, w.Run(ctx))
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, events, 1)
		assert.Equal(t, model.OutboxStatusFailed, events[0].Status, "timed-out event failed permanently")

		blockers, e := fRepo.ListBlockers(ctx, tx, procID)
		if e != nil {
			return e
		}
		require.Len(t, blockers, 1, "exactly one timeout blocker opened")
		assert.Equal(t, model.BlockerExternalWorkflowTimeout, blockers[0].Code)
		assert.Equal(t, "integration", blockers[0].Category)
		return nil
	}))
}

// TestExternalWorkflow_DisabledServiceNoOp verifies the gated-off parity: a disabled service's
// DispatchForOrder is a no-op (no process, no outbox rows).
func TestExternalWorkflow_DisabledServiceNoOp(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	integrationID := seedEWFIntegration(t, ctx, tenant)
	orderID := seedEWFOrder(t, ctx, tenant)

	svc := newEWFService(false) // disabled
	require.NoError(t, svc.DispatchForOrder(ctx, tenant, orderID, integrationID))

	fRepo := repository.NewFulfillmentRepository()
	require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		_, e := fRepo.GetProcessByOrder(ctx, tx, orderID)
		assert.True(t, errors.Is(e, pgx.ErrNoRows), "disabled connector creates no fulfillment process")
		return nil
	}))
	var outboxCount int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM orchestration_outbox WHERE tenant_id = $1`, tenant).Scan(&outboxCount))
	assert.Equal(t, 0, outboxCount, "disabled connector enqueues zero outbox rows")
}

// firstOutboxIDForType returns the id of the first outbox event of the given type for a process.
func firstOutboxIDForType(t *testing.T, ctx context.Context, procID uuid.UUID, eventType string) uuid.UUID {
	t.Helper()
	orchRepo := repository.NewOrchestrationRepository()
	var id uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, procTenant(t, ctx, procID), func(tx pgx.Tx) error {
		events, e := orchRepo.ListByProcess(ctx, tx, procID)
		if e != nil {
			return e
		}
		for _, ev := range events {
			if ev.EventType == eventType {
				id = ev.ID
				return nil
			}
		}
		return errors.New("no event of type found")
	}))
	return id
}

// procTenant resolves a process's tenant id via the superuser pool.
func procTenant(t *testing.T, ctx context.Context, procID uuid.UUID) uuid.UUID {
	t.Helper()
	var tid uuid.UUID
	require.NoError(t, superPool.QueryRow(ctx, `SELECT tenant_id FROM fulfillment_processes WHERE id = $1`, procID).Scan(&tid))
	return tid
}
