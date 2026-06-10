package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/obsmetrics"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// OPE-422: ADDITIVE, best-effort observability on FulfillmentReadService — the
// stuck/blocked process gauges derived from the operations summary, and the
// operator-action audit wiring. The gauge math is a PURE helper so it is tested
// without a DB here; the audit write itself runs in its own tenant transaction and
// is covered by the build + an integration-style harness, not this pure test.

func TestFulfillmentReadService_WithMetricsAndAudit_Chainable(t *testing.T) {
	m := obsmetrics.NewFulfillmentMetrics()
	audit := repository.NewAuditRepository()
	svc := NewFulfillmentReadService(nil, nil, nil, nil).WithMetrics(m).WithAudit(audit)
	require.NotNil(t, svc)
	assert.Same(t, m, svc.metrics)
	assert.NotNil(t, svc.audit)
}

func TestFulfillmentReadService_WithMetrics_NilReceiverSafe(t *testing.T) {
	var svc *FulfillmentReadService // nil
	assert.NotPanics(t, func() {
		assert.Nil(t, svc.WithMetrics(obsmetrics.NewFulfillmentMetrics()))
		assert.Nil(t, svc.WithAudit(repository.NewAuditRepository()))
	})
}

// gaugeCountsFromStatusCounts is the pure helper feeding the global stuck/blocked
// gauges (OPE-422 followup): stuck = the stuck bucket; blocked = every blocker-holding
// bucket (blocked + provider_issue + missing_data); terminal/healthy excluded.
func TestGaugeCountsFromStatusCounts(t *testing.T) {
	counts := []repository.ProcessStatusCount{
		{AggregateStatus: model.ProcessStatusInProgress, HealthStatus: model.ProcessHealthSystemError, Count: 2},  // stuck
		{AggregateStatus: model.ProcessStatusBlocked, HealthStatus: model.ProcessHealthActionRequired, Count: 4},  // blocked
		{AggregateStatus: model.ProcessStatusWaitingExternal, HealthStatus: model.ProcessHealthWarning, Count: 3}, // provider_issue -> blocked
		{AggregateStatus: model.ProcessStatusInProgress, HealthStatus: model.ProcessHealthOK, Count: 9},           // processing (excluded)
		{AggregateStatus: model.ProcessStatusReady, HealthStatus: model.ProcessHealthOK, Count: 5},                // ready (excluded)
		{AggregateStatus: model.ProcessStatusCompleted, HealthStatus: model.ProcessHealthOK, Count: 7},            // terminal (excluded)
	}
	stuck, blocked := gaugeCountsFromStatusCounts(counts)
	assert.Equal(t, 2, stuck, "only the system-error in-progress process is stuck")
	assert.Equal(t, 7, blocked, "blocked(4) + provider_issue(3); processing/ready/terminal excluded")

	stuck, blocked = gaugeCountsFromStatusCounts(nil)
	assert.Equal(t, 0, stuck)
	assert.Equal(t, 0, blocked)
}

// auditAction is a no-op when no audit writer is wired — it must NOT touch the pool
// (which is nil here), proving the audit path is skipped entirely without it.
func TestFulfillmentReadService_AuditAction_NoWriter_NoOp(t *testing.T) {
	svc := NewFulfillmentReadService(nil, nil, nil, nil) // no audit, nil pool
	assert.NotPanics(t, func() {
		svc.auditAction(context.Background(), uuid.New(), model.AuditEntry{
			Action:     "fulfillment.blocker.resolved",
			EntityType: "fulfillment_blocker",
			EntityID:   uuid.New(),
		})
	})
}
