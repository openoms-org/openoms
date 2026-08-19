//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// TestShipmentService_UpdateStatusByTrackingNumber_TenantIsolation proves that
// ShipmentService.UpdateStatusByTrackingNumberForTenant is tenant-isolated through
// RLS: the method looks up + updates the shipment inside database.WithTenant on the
// RLS-scoped openoms_app pool, so tenant B cannot touch tenant A's shipment.
func TestShipmentService_UpdateStatusByTrackingNumber_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)

	// Seed order + shipment for tenant A via superPool (bypasses RLS). The tenant
	// cleanup cascades to orders -> shipments via ON DELETE CASCADE.
	orderID := uuid.New()
	_, err := superPool.Exec(ctx,
		`INSERT INTO orders (id, tenant_id, customer_name) VALUES ($1, $2, $3)`,
		orderID, tenantA, "Isolation Customer A")
	require.NoError(t, err, "seed order for tenant A")

	shipmentID := uuid.New()
	tracking := "TRK-" + shipmentID.String()[:8]
	_, err = superPool.Exec(ctx,
		`INSERT INTO shipments (id, tenant_id, order_id, provider, tracking_number, status)
		 VALUES ($1, $2, $3, 'inpost', $4, 'created')`,
		shipmentID, tenantA, orderID, tracking)
	require.NoError(t, err, "seed shipment for tenant A")

	// Service runs on the RLS-scoped app pool (openoms_app, NOBYPASSRLS).
	svc := service.NewShipmentService(nil, nil, nil, nil, nil, appPool, nil, nil)

	statusOf := func() string {
		var s string
		require.NoError(t, superPool.QueryRow(ctx, "SELECT status FROM shipments WHERE id = $1", shipmentID).Scan(&s))
		return s
	}

	// Cross-tenant: tenant B must NOT be able to find/update tenant A's shipment.
	err = svc.UpdateStatusByTrackingNumberForTenant(ctx, tenantB, tracking, "inpost", "in_transit")
	require.Error(t, err, "tenant B must not update tenant A's shipment (RLS)")
	assert.Equal(t, "created", statusOf(), "shipment must be unchanged after a cross-tenant attempt")

	// Same-tenant: positive control.
	err = svc.UpdateStatusByTrackingNumberForTenant(ctx, tenantA, tracking, "inpost", "in_transit")
	require.NoError(t, err, "tenant A must update its own shipment")
	assert.Equal(t, "in_transit", statusOf(), "tenant A's update must persist")
}
