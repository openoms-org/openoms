//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// TestFulfillmentGaugeSweep_CrossTenantGlobal proves the OPE-422 followup: the global
// stuck/blocked process gauges are computed ACROSS ALL TENANTS via the privileged pool
// (not per-tenant), and rendered as a single label-free aggregate. Seeds processes for
// two tenants, then asserts the swept gauges equal the true cross-tenant totals.
func TestFulfillmentGaugeSweep_CrossTenantGlobal(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	orderA, _ := seedFulfillmentOrder(t, ctx, tenantA, "Gauge A")
	orderB1, _ := seedFulfillmentOrder(t, ctx, tenantB, "Gauge B1")
	orderB2, _ := seedFulfillmentOrder(t, ctx, tenantB, "Gauge B2")

	// Tenant A: one blocked process. Tenant B: one stuck (in_progress + system_error)
	// and one waiting_external (→ provider_issue → counts toward the blocked gauge).
	seedProcessWithStatus(t, ctx, tenantA, orderA, model.ProcessStatusBlocked, model.ProcessHealthActionRequired)
	seedProcessWithStatus(t, ctx, tenantB, orderB1, model.ProcessStatusInProgress, model.ProcessHealthSystemError)
	seedProcessWithStatus(t, ctx, tenantB, orderB2, model.ProcessStatusWaitingExternal, model.ProcessHealthWarning)

	// Ground truth, computed directly cross-tenant on the privileged pool. blocked gauge
	// = aggregate in (blocked, waiting_external); stuck gauge = health system_error and
	// aggregate not already terminal/blocked/waiting (mirrors classifyProcessBucket).
	var wantBlocked, wantStuck int
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT count(*) FROM fulfillment_processes WHERE aggregate_status IN ('blocked','waiting_external')`).Scan(&wantBlocked))
	require.NoError(t, superPool.QueryRow(ctx,
		`SELECT count(*) FROM fulfillment_processes
		  WHERE health_status = 'system_error'
		    AND aggregate_status NOT IN ('blocked','waiting_external','completed','cancelled')`).Scan(&wantStuck))
	// Our seeds must be reflected (cross-tenant: A's blocked + B's waiting_external).
	require.GreaterOrEqual(t, wantBlocked, 2, "blocked gauge must include tenant A AND tenant B contributions")
	require.GreaterOrEqual(t, wantStuck, 1, "stuck gauge must include tenant B's system-error process")

	// Run the sweep on the privileged pool and render the metrics.
	metrics := obsmetrics.NewFulfillmentMetrics()
	svc := service.NewFulfillmentReadService(
		appPool,
		repository.NewFulfillmentRepository(),
		repository.NewFulfillmentAttemptRepository(),
		repository.NewOrchestrationRepository(),
	).WithMetrics(metrics)
	require.NoError(t, svc.SweepGlobalProcessGauges(ctx, superPool))

	var b strings.Builder
	metrics.Render(&b)
	out := b.String()

	assert.Contains(t, out, fmt.Sprintf("openoms_fulfillment_stuck_processes %d", wantStuck))
	assert.Contains(t, out, fmt.Sprintf("openoms_fulfillment_blocked_processes %d", wantBlocked))
	// Cardinality discipline: a single global aggregate, no tenant label.
	assert.NotContains(t, out, "openoms_fulfillment_stuck_processes{")
	assert.NotContains(t, out, "openoms_fulfillment_blocked_processes{")
	assert.NotContains(t, out, tenantA.String())
	assert.NotContains(t, out, tenantB.String())

	// Cross-tenant proof: a SINGLE-tenant (RLS-scoped) count for A alone is strictly less
	// than the global blocked gauge, because tenant B also contributes — so the gauge is
	// genuinely global, not the last-observed tenant.
	aOnly, err := svc.OperationsSummary(ctx, tenantA)
	require.NoError(t, err)
	assert.Less(t, aOnly.Buckets[service.BucketBlocked], wantBlocked,
		"tenant A alone has fewer blocked than the global gauge — proves the gauge is cross-tenant")
}
