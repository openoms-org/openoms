//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// TestCreateProcess_Idempotent_OneProcessPerOrder proves the OPE-423a-followup
// hardening: CreateProcess is idempotent + race-safe via the
// uq_fulfillment_processes_tenant_order unique index (migration 000040) +
// INSERT ... ON CONFLICT DO NOTHING. Two creates for the same order return the SAME
// process and leave exactly ONE row — a duplicate process is structurally impossible.
func TestCreateProcess_Idempotent_OneProcessPerOrder(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	orderID := seedOrderWithStatus(t, ctx, tenantID, model.OrderStatusProcessing)

	repo := repository.NewFulfillmentRepository()

	create := func() *model.FulfillmentProcess {
		var proc *model.FulfillmentProcess
		require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
			p, e := repo.CreateProcess(ctx, tx, model.FulfillmentProcess{TenantID: tenantID, OrderID: orderID})
			if e != nil {
				return e
			}
			proc = p
			return nil
		}))
		require.NotNil(t, proc)
		return proc
	}

	first := create()
	second := create()

	// Idempotent: the second create returns the same process, not a new one.
	assert.Equal(t, first.ID, second.ID, "second CreateProcess must return the existing process, not create a duplicate")

	// And the DB holds exactly one process row for the order.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx,
			`SELECT count(*) FROM fulfillment_processes WHERE order_id = $1`, orderID).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "exactly one fulfillment_process row must exist for the order")
		return nil
	}))
}
