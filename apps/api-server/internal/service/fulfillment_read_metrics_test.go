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
