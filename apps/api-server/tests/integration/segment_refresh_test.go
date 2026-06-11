//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/handler"
	"github.com/openoms-org/openoms/apps/api-server/internal/middleware"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
	"github.com/openoms-org/openoms/apps/api-server/internal/worker"
)

// newSegmentService builds a real SegmentService bound to the supplied pool. In the
// worker/manual paths it is bound to appPool (openoms_app, NOBYPASSRLS) so the
// per-tenant database.WithTenant set_config genuinely scopes the rule query.
func newSegmentService(pool *pgxpool.Pool) *service.SegmentService {
	return service.NewSegmentService(
		repository.NewCustomerSegmentRepository(),
		repository.NewAuditRepository(),
		pool,
		slog.Default(),
	)
}

// seedSegmentCustomer inserts a customer for tenant via the superuser pool.
func seedSegmentCustomer(t *testing.T, ctx context.Context, tenant uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO customers (id, tenant_id, name) VALUES ($1, $2, $3)`,
		id, tenant, name)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customers WHERE id = $1", id) })
	return id
}

// seedCustomerOrder inserts one order for a customer (so MinOrders>=1 rules match).
func seedCustomerOrder(t *testing.T, ctx context.Context, tenant, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO orders (id, tenant_id, customer_name, customer_id, total_amount)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, tenant, "Seg Order", customerID, 100)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id) })
	return id
}

// seedSegment inserts a customer_segments row via the superuser pool. Pass nil rules
// for non-rule_based segments.
func seedSegment(t *testing.T, ctx context.Context, tenant uuid.UUID, name, segmentType string, rules json.RawMessage) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var rulesArg any
	if rules != nil {
		rulesArg = rules
	}
	_, err := superPool.Exec(ctx,
		`INSERT INTO customer_segments (id, tenant_id, name, color, segment_type, rules)
		 VALUES ($1, $2, $3, '#6366f1', $4, $5)`,
		id, tenant, name, segmentType, rulesArg)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = superPool.Exec(context.Background(), "DELETE FROM customer_segments WHERE id = $1", id) })
	return id
}

// segmentMemberIDs returns the customer ids currently in a segment (via superuser pool).
func segmentMemberIDs(t *testing.T, ctx context.Context, segmentID uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := superPool.Query(ctx,
		`SELECT customer_id FROM customer_segment_members WHERE segment_id = $1 ORDER BY customer_id`, segmentID)
	require.NoError(t, err)
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())
	return ids
}

// segmentCustomerCount returns the cached customer_count column for a segment.
func segmentCustomerCount(t *testing.T, ctx context.Context, segmentID uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT customer_count FROM customer_segments WHERE id = $1`, segmentID).Scan(&n))
	return n
}

// TestSegmentRefreshWorker_RecomputesMembershipPerTenant proves the periodic worker
// enumerates every tenant on the privileged pool and recomputes each tenant's
// rule-based segment membership under that tenant's RLS scope — with no cross-tenant
// leakage and the cached customer_count updated.
func TestSegmentRefreshWorker_RecomputesMembershipPerTenant(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	// Tenant A: one matching customer (has an order) + one non-matching (no orders).
	aMatch := seedSegmentCustomer(t, ctx, tenantA, "A Match")
	seedCustomerOrder(t, ctx, tenantA, aMatch)
	_ = seedSegmentCustomer(t, ctx, tenantA, "A NoOrders")
	segA := seedSegment(t, ctx, tenantA, "A Buyers", "rule_based", json.RawMessage(`{"min_orders":1}`))

	// Tenant B: one matching customer.
	bMatch := seedSegmentCustomer(t, ctx, tenantB, "B Match")
	seedCustomerOrder(t, ctx, tenantB, bMatch)
	segB := seedSegment(t, ctx, tenantB, "B Buyers", "rule_based", json.RawMessage(`{"min_orders":1}`))

	// Worker enumerates tenants on superPool (RLS-bypassing, like the prod workerPool);
	// the service refreshes each tenant on appPool so RLS set_config applies per tenant.
	w := worker.NewSegmentRefreshWorker(superPool, newSegmentService(appPool), slog.Default())
	require.NoError(t, w.Run(ctx))

	assert.ElementsMatch(t, []uuid.UUID{aMatch}, segmentMemberIDs(t, ctx, segA), "only the buyer is in A's segment")
	assert.Equal(t, 1, segmentCustomerCount(t, ctx, segA))
	assert.ElementsMatch(t, []uuid.UUID{bMatch}, segmentMemberIDs(t, ctx, segB), "only the buyer is in B's segment")
	assert.Equal(t, 1, segmentCustomerCount(t, ctx, segB))

	// Idempotency: a second tick yields identical membership and count.
	require.NoError(t, w.Run(ctx))
	assert.ElementsMatch(t, []uuid.UUID{aMatch}, segmentMemberIDs(t, ctx, segA))
	assert.Equal(t, 1, segmentCustomerCount(t, ctx, segA))
}

// TestSegmentRefreshWorker_PerTenantIsolation_OneTenantErrors proves a single
// tenant's refresh failure (here: a corrupted segment row that fails to scan) is
// logged and skipped via `continue`, so the other tenants in the same pass are still
// refreshed.
func TestSegmentRefreshWorker_PerTenantIsolation_OneTenantErrors(t *testing.T) {
	ctx := context.Background()
	good := seedTenant(t, ctx)
	bad := seedTenant(t, ctx)

	goodMatch := seedSegmentCustomer(t, ctx, good, "Good Match")
	seedCustomerOrder(t, ctx, good, goodMatch)
	segGood := seedSegment(t, ctx, good, "Good Buyers", "rule_based", json.RawMessage(`{"min_orders":1}`))

	// Corrupt the bad tenant's segment so segmentRepo.List fails to scan it (NULL into
	// the non-nullable Color string) — RefreshRuleBasedSegments returns an error.
	segBad := seedSegment(t, ctx, bad, "Bad Seg", "rule_based", json.RawMessage(`{"min_orders":1}`))
	_, err := superPool.Exec(ctx, `UPDATE customer_segments SET color = NULL WHERE id = $1`, segBad)
	require.NoError(t, err)

	w := worker.NewSegmentRefreshWorker(superPool, newSegmentService(appPool), slog.Default())
	require.NoError(t, w.Run(ctx), "one tenant's error must not abort the pass")

	assert.ElementsMatch(t, []uuid.UUID{goodMatch}, segmentMemberIDs(t, ctx, segGood),
		"the healthy tenant is still refreshed despite the other tenant erroring")
	assert.Equal(t, 1, segmentCustomerCount(t, ctx, segGood))
}

// TestSegmentService_RefreshSegment exercises the single-segment manual path:
// happy recompute, not-found, wrong segment type, idempotency and reactive removal.
func TestSegmentService_RefreshSegment(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	svc := newSegmentService(appPool)

	cust := seedSegmentCustomer(t, ctx, tenant, "Buyer")
	order := seedCustomerOrder(t, ctx, tenant, cust)
	_ = seedSegmentCustomer(t, ctx, tenant, "NonBuyer")
	seg := seedSegment(t, ctx, tenant, "Buyers", "rule_based", json.RawMessage(`{"min_orders":1}`))

	// Happy path: single segment recomputed to exactly the buyer.
	require.NoError(t, svc.RefreshSegment(ctx, tenant, seg))
	assert.ElementsMatch(t, []uuid.UUID{cust}, segmentMemberIDs(t, ctx, seg))
	assert.Equal(t, 1, segmentCustomerCount(t, ctx, seg))

	// Idempotency: re-running yields identical membership and count.
	require.NoError(t, svc.RefreshSegment(ctx, tenant, seg))
	assert.ElementsMatch(t, []uuid.UUID{cust}, segmentMemberIDs(t, ctx, seg))
	assert.Equal(t, 1, segmentCustomerCount(t, ctx, seg))

	// Reactive recompute: once the buyer no longer qualifies, refresh removes them
	// (proves clear-then-repopulate, not additive).
	_, err := superPool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, order)
	require.NoError(t, err)
	require.NoError(t, svc.RefreshSegment(ctx, tenant, seg))
	assert.Empty(t, segmentMemberIDs(t, ctx, seg))
	assert.Equal(t, 0, segmentCustomerCount(t, ctx, seg))

	// Not found.
	assert.ErrorIs(t, svc.RefreshSegment(ctx, tenant, uuid.New()), service.ErrSegmentNotFound)

	// Wrong type: a manual segment is a validation error, not a recompute.
	manual := seedSegment(t, ctx, tenant, "Manual", "manual", nil)
	err = svc.RefreshSegment(ctx, tenant, manual)
	require.Error(t, err)
	assert.NotErrorIs(t, err, service.ErrSegmentNotFound)
	var ve *service.ValidationError
	assert.ErrorAs(t, err, &ve, "non-rule_based segment yields a ValidationError")
}

// TestSegmentHandler_RefreshSegment_Integration drives the real HTTP handler against
// a DB-backed service: 204 on success, 404 for unknown id, 400 for a wrong segment
// type and 400 for a malformed id.
func TestSegmentHandler_RefreshSegment_Integration(t *testing.T) {
	ctx := context.Background()
	tenant := seedTenant(t, ctx)
	h := handler.NewSegmentHandler(newSegmentService(appPool))

	cust := seedSegmentCustomer(t, ctx, tenant, "HBuyer")
	seedCustomerOrder(t, ctx, tenant, cust)
	seg := seedSegment(t, ctx, tenant, "HBuyers", "rule_based", json.RawMessage(`{"min_orders":1}`))

	// 204 success + membership recomputed.
	rec := callRefreshSegment(h, tenant, seg.String())
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.ElementsMatch(t, []uuid.UUID{cust}, segmentMemberIDs(t, ctx, seg))

	// 404 unknown id.
	rec = callRefreshSegment(h, tenant, uuid.New().String())
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// 400 wrong type.
	manual := seedSegment(t, ctx, tenant, "HManual", "manual", nil)
	rec = callRefreshSegment(h, tenant, manual.String())
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// 400 malformed id.
	rec = callRefreshSegment(h, tenant, "not-a-uuid")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// callRefreshSegment invokes SegmentHandler.RefreshSegment with the tenant injected
// into the request context and the {id} chi URL param set.
func callRefreshSegment(h *handler.SegmentHandler, tenant uuid.UUID, idParam string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/segments/"+idParam+"/refresh", nil)
	reqCtx := context.WithValue(req.Context(), middleware.TenantIDKey, tenant)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", idParam)
	reqCtx = context.WithValue(reqCtx, chi.RouteCtxKey, rctx)
	req = req.WithContext(reqCtx)
	rec := httptest.NewRecorder()
	h.RefreshSegment(rec, req)
	return rec
}
