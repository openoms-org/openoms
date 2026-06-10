//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// OPE-523 (CQ-FULF-03): EnsureUnit/RecordStep dedupe application-side in separate
// best-effort transactions, so concurrent callers (poller + operator action) could
// both miss the existing row and insert duplicates. Migration 000045 adds unique
// indexes mirroring the app-level dedupe keys:
//
//   - fulfillment_units:  (tenant_id, process_id, unit_type, (metadata->>'key'))
//     NULLS NOT DISTINCT (EnsureUnit stores the dedupe key in metadata under
//     'key' only when non-empty; keyless units collapse too)
//   - fulfillment_steps:  (tenant_id, unit_id, step_key)
//
// These tests prove the DB backstop holds even when the service-level dedupe is
// bypassed entirely.

// requireUniqueViolation asserts err is a PostgreSQL unique_violation (23505).
func requireUniqueViolation(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr, "expected a *pgconn.PgError, got: %v", err)
	assert.Equal(t, "23505", pgErr.Code, "expected unique_violation, got SQLSTATE %s: %s", pgErr.Code, pgErr.Message)
}

// TestFulfillmentDedup_UnitUniqueBackstop proves uq_fulfillment_units_dedupe: a raw
// duplicate INSERT (bypassing the service's read-then-create dedupe) is rejected
// with a unique violation, for both a keyed (dropship) and a keyless (warehouse)
// unit — the NULLS NOT DISTINCT index collapses missing keys too.
func TestFulfillmentDedup_UnitUniqueBackstop(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	orderID, processID := seedFulfillmentOrder(t, ctx, tenantID, "Dedup Unit Customer")

	svc := newUnitService()

	// Keyed unit (dropship deduped per supplier, key stored in metadata->>'key').
	supplierKey := uuid.New().String()
	keyed := svc.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeDropship, supplierKey, nil)
	require.NotNil(t, keyed)

	keyedDupErr := database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO fulfillment_units (tenant_id, process_id, unit_type, status, metadata)
			 VALUES ($1,$2,$3,'pending', jsonb_build_object('key', $4::text))`,
			tenantID, processID, model.UnitTypeDropship, supplierKey)
		return e
	})
	requireUniqueViolation(t, keyedDupErr)

	// Keyless unit (warehouse path passes an empty dedupe key -> no 'key' in metadata).
	keyless := svc.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeWarehouse, "", nil)
	require.NotNil(t, keyless)

	keylessDupErr := database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO fulfillment_units (tenant_id, process_id, unit_type, status, metadata)
			 VALUES ($1,$2,$3,'pending','{}'::jsonb)`,
			tenantID, processID, model.UnitTypeWarehouse)
		return e
	})
	requireUniqueViolation(t, keylessDupErr)

	// A DIFFERENT key on the same (process, unit_type) is still a legitimate row.
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO fulfillment_units (tenant_id, process_id, unit_type, status, metadata)
			 VALUES ($1,$2,$3,'pending', jsonb_build_object('key', $4::text))`,
			tenantID, processID, model.UnitTypeDropship, uuid.New().String())
		return e
	}))
}

// TestFulfillmentDedup_StepUniqueBackstop proves uq_fulfillment_steps_unit_step: a
// raw duplicate INSERT of the same (unit, step_key) is rejected, while the
// service-level upsert (RecordStep on an existing step) keeps working — it UPDATEs
// the existing row (attempts increment) instead of inserting a second one.
func TestFulfillmentDedup_StepUniqueBackstop(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	orderID, _ := seedFulfillmentOrder(t, ctx, tenantID, "Dedup Step Customer")

	svc := newUnitService()
	unit := svc.EnsureUnit(ctx, tenantID, orderID, model.UnitTypeWarehouse, "", nil)
	require.NotNil(t, unit)

	first := svc.RecordStep(ctx, tenantID, unit.ID, model.StepPickItems, model.FulfillmentStatusRunning, nil)
	require.NotNil(t, first)
	assert.Equal(t, 1, first.Attempts)

	// Raw duplicate insert bypassing the service -> unique violation.
	dupErr := database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx,
			`INSERT INTO fulfillment_steps (tenant_id, unit_id, step_key, status, attempts, metadata)
			 VALUES ($1,$2,$3,'running',1,'{}'::jsonb)`,
			tenantID, unit.ID, model.StepPickItems)
		return e
	})
	requireUniqueViolation(t, dupErr)

	// Upsert semantics intact: re-recording updates the row, no duplicate insert.
	second := svc.RecordStep(ctx, tenantID, unit.ID, model.StepPickItems, model.FulfillmentStatusSucceeded, nil)
	require.NotNil(t, second)
	assert.Equal(t, first.ID, second.ID, "re-recording must update the existing step, not insert a new one")
	assert.Equal(t, 2, second.Attempts)
	assert.Equal(t, model.FulfillmentStatusSucceeded, second.Status)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx,
			`SELECT count(*) FROM fulfillment_steps WHERE unit_id = $1 AND step_key = $2`,
			unit.ID, model.StepPickItems).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "exactly one step row must exist for the (unit, step_key)")
		return nil
	}))
}

// raceTimeout bounds the channel-orchestrated race tests below.
const raceTimeout = 10 * time.Second
