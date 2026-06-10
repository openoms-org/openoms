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
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

// seedSupplierAndProduct inserts a supplier + supplier_product for the tenant (RLS-scoped
// via WithTenant) and returns their ids.
func seedSupplierAndProduct(t *testing.T, ctx context.Context, tenantID uuid.UUID) (supplierID, supplierProductID uuid.UUID) {
	t.Helper()
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO suppliers (tenant_id, name, feed_format) VALUES ($1,'Acme','iof') RETURNING id`,
			tenantID).Scan(&supplierID); err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`INSERT INTO supplier_products (tenant_id, supplier_id, external_id, name, stock_quantity)
			 VALUES ($1,$2,'EXT-1','Widget',100) RETURNING id`,
			tenantID, supplierID).Scan(&supplierProductID)
	}))
	return supplierID, supplierProductID
}

// TestSupplierAvailability_UpsertIdempotent proves the snapshot upsert is idempotent on
// (tenant, supplier_product, warehouse) — two upserts leave exactly one row, updated.
func TestSupplierAvailability_UpsertIdempotent(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, spID := seedSupplierAndProduct(t, ctx, tenantID)
	repo := repository.NewSupplierAvailabilityRepository()

	upsert := func(qty int) {
		require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
			_, e := repo.UpsertSnapshot(ctx, tx, model.SupplierAvailability{
				TenantID: tenantID, SupplierID: supplierID, SupplierProductID: spID,
				WarehouseExternalID: "", SourceQuantity: qty,
				AvailabilityType: model.AvailabilityExactQuantity, FreshnessObservedAt: time.Now(),
			})
			return e
		}))
	}
	upsert(100)
	upsert(80)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var n, qty int
		if e := tx.QueryRow(ctx, `SELECT count(*), max(source_quantity) FROM supplier_availability WHERE supplier_product_id=$1`, spID).Scan(&n, &qty); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "exactly one snapshot row")
		assert.Equal(t, 80, qty, "second upsert updated the quantity")
		return nil
	}))
}

// TestSupplierAvailability_RLSIsolation proves tenant A never sees tenant B's snapshots.
func TestSupplierAvailability_RLSIsolation(t *testing.T) {
	ctx := context.Background()
	tenantA := seedTenant(t, ctx)
	tenantB := seedTenant(t, ctx)
	sa, spa := seedSupplierAndProduct(t, ctx, tenantA)
	sb, spb := seedSupplierAndProduct(t, ctx, tenantB)
	repo := repository.NewSupplierAvailabilityRepository()

	seed := func(tenant, supplier, sp uuid.UUID) {
		require.NoError(t, database.WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
			_, e := repo.UpsertSnapshot(ctx, tx, model.SupplierAvailability{
				TenantID: tenant, SupplierID: supplier, SupplierProductID: sp,
				SourceQuantity: 50, AvailabilityType: model.AvailabilityExactQuantity, FreshnessObservedAt: time.Now(),
			})
			return e
		}))
	}
	seed(tenantA, sa, spa)
	seed(tenantB, sb, spb)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM supplier_availability`).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "tenant A sees only its own snapshot")
		return nil
	}))
}

// TestSupplierAvailability_PolicyScopeDiscriminatorCheck proves chk_sap_scope_discriminator
// (migration 000044) rejects rows whose discriminator columns do not match their scope.
// Without it such rows escape the partial unique indexes from 000041 entirely (NULL index
// keys never conflict), so duplicates and unreachable policies could accumulate.
func TestSupplierAvailability_PolicyScopeDiscriminatorCheck(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, _ := seedSupplierAndProduct(t, ctx, tenantID)

	var productID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `INSERT INTO products (tenant_id, name, price) VALUES ($1,'P',10) RETURNING id`, tenantID).Scan(&productID)
	}))

	cases := []struct {
		name string
		sql  string
		args []any
	}{
		{"supplier scope without supplier_id",
			`INSERT INTO supplier_availability_policy (tenant_id, scope) VALUES ($1,'supplier')`,
			[]any{tenantID}},
		{"product scope without supplier_id",
			`INSERT INTO supplier_availability_policy (tenant_id, scope, product_id) VALUES ($1,'product',$2)`,
			[]any{tenantID, productID}},
		{"channel scope without channel",
			`INSERT INTO supplier_availability_policy (tenant_id, scope) VALUES ($1,'channel')`,
			[]any{tenantID}},
		{"listing scope with extra supplier_id",
			`INSERT INTO supplier_availability_policy (tenant_id, scope, listing_id, supplier_id) VALUES ($1,'listing',$2,$3)`,
			[]any{tenantID, uuid.New(), supplierID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
				_, e := tx.Exec(ctx, tc.sql, tc.args...)
				return e
			})
			require.Error(t, err, "scope/discriminator mismatch must be rejected")
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			assert.Equal(t, "23514", pgErr.Code, "check_violation")
			assert.Equal(t, "chk_sap_scope_discriminator", pgErr.ConstraintName)
		})
	}
}

// TestSupplierAvailability_PolicyChainResolves proves the 4-scope policy load returns the
// applicable rows for a (supplier, product) context (supplier + product scopes).
func TestSupplierAvailability_PolicyChainResolves(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, _ := seedSupplierAndProduct(t, ctx, tenantID)
	repo := repository.NewSupplierAvailabilityRepository()

	var productID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, `INSERT INTO products (tenant_id, name, price) VALUES ($1,'P',10) RETURNING id`, tenantID).Scan(&productID); e != nil {
			return e
		}
		fw := 7200
		if _, e := repo.UpsertPolicy(ctx, tx, model.SupplierAvailabilityPolicy{
			TenantID: tenantID, Scope: model.PolicyScopeSupplier, SupplierID: &supplierID,
			Mode: model.PolicyModeAuto, SafetyBuffer: 5, FreshnessWindowSecs: &fw,
		}); e != nil {
			return e
		}
		_, e := repo.UpsertPolicy(ctx, tx, model.SupplierAvailabilityPolicy{
			TenantID: tenantID, Scope: model.PolicyScopeProduct, SupplierID: &supplierID, ProductID: &productID,
			Mode: model.PolicyModeAuto, SafetyBuffer: 9,
		})
		return e
	}))

	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		policies, e := repo.ListPoliciesForContext(ctx, tx, supplierID, productID, nil, nil)
		if e != nil {
			return e
		}
		assert.Len(t, policies, 2, "supplier + product scope rows both apply")
		return nil
	}))
}

// TestSupplierAvailability_SetPolicyTwiceUpdates proves the audited SetPolicy path is a
// real upsert (OPE-528): a second SetPolicy for the same scope target UPDATES the existing
// row instead of raising unique_violation, and exactly one row remains.
func TestSupplierAvailability_SetPolicyTwiceUpdates(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, _ := seedSupplierAndProduct(t, ctx, tenantID)
	svc := service.NewSupplierAvailabilityService(true, appPool, repository.NewSupplierAvailabilityRepository(), nil)

	first, err := svc.SetPolicy(ctx, tenantID, uuid.New(), "127.0.0.1", model.SupplierAvailabilityPolicy{
		Scope: model.PolicyScopeSupplier, SupplierID: &supplierID,
		Mode: model.PolicyModeAuto, SafetyBuffer: 5,
	})
	require.NoError(t, err)

	oq := 42
	second, err := svc.SetPolicy(ctx, tenantID, uuid.New(), "127.0.0.1", model.SupplierAvailabilityPolicy{
		Scope: model.PolicyScopeSupplier, SupplierID: &supplierID,
		Mode: model.PolicyModeManual, SafetyBuffer: 9, OverrideQuantity: &oq,
	})
	require.NoError(t, err, "second SetPolicy for the same target must update, not conflict")
	assert.Equal(t, first.ID, second.ID, "same row updated in place")
	assert.Equal(t, model.PolicyModeManual, second.Mode)
	assert.Equal(t, 9, second.SafetyBuffer)
	require.NotNil(t, second.OverrideQuantity)
	assert.Equal(t, 42, *second.OverrideQuantity)

	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var n int
		if e := tx.QueryRow(ctx, `SELECT count(*) FROM supplier_availability_policy WHERE supplier_id=$1`, supplierID).Scan(&n); e != nil {
			return e
		}
		assert.Equal(t, 1, n, "exactly one policy row for the target")
		return nil
	}))
}

// TestSupplierAvailability_PolicyUpsertAllScopes exercises every scope-specific conflict
// target of UpsertPolicy (each scope has its own partial unique index, so each branch
// carries its own ON CONFLICT clause): insert then update per scope, one row each.
func TestSupplierAvailability_PolicyUpsertAllScopes(t *testing.T) {
	ctx := context.Background()
	tenantID := seedTenant(t, ctx)
	supplierID, _ := seedSupplierAndProduct(t, ctx, tenantID)
	repo := repository.NewSupplierAvailabilityRepository()

	var productID, listingID uuid.UUID
	require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, `INSERT INTO products (tenant_id, name, price) VALUES ($1,'P',10) RETURNING id`, tenantID).Scan(&productID); e != nil {
			return e
		}
		var integrationID uuid.UUID
		if e := tx.QueryRow(ctx, `INSERT INTO integrations (tenant_id, provider) VALUES ($1,'allegro') RETURNING id`, tenantID).Scan(&integrationID); e != nil {
			return e
		}
		return tx.QueryRow(ctx, `INSERT INTO product_listings (tenant_id, product_id, integration_id) VALUES ($1,$2,$3) RETURNING id`,
			tenantID, productID, integrationID).Scan(&listingID)
	}))

	channel := "allegro"
	policies := []model.SupplierAvailabilityPolicy{
		{Scope: model.PolicyScopeSupplier, SupplierID: &supplierID},
		{Scope: model.PolicyScopeProduct, SupplierID: &supplierID, ProductID: &productID},
		{Scope: model.PolicyScopeListing, ListingID: &listingID},
		{Scope: model.PolicyScopeChannel, Channel: &channel},
	}
	for _, p := range policies {
		t.Run(p.Scope, func(t *testing.T) {
			p.TenantID, p.Mode, p.SafetyBuffer = tenantID, model.PolicyModeAuto, 1
			var firstID uuid.UUID
			require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
				saved, e := repo.UpsertPolicy(ctx, tx, p)
				if e != nil {
					return e
				}
				firstID = saved.ID
				return nil
			}))
			p.SafetyBuffer = 7
			require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
				saved, e := repo.UpsertPolicy(ctx, tx, p)
				if e != nil {
					return e
				}
				assert.Equal(t, firstID, saved.ID, "same row updated in place")
				assert.Equal(t, 7, saved.SafetyBuffer, "rule columns updated")
				return nil
			}))
			require.NoError(t, database.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
				var n int
				if e := tx.QueryRow(ctx, `SELECT count(*) FROM supplier_availability_policy WHERE scope=$1`, p.Scope).Scan(&n); e != nil {
					return e
				}
				assert.Equal(t, 1, n, "exactly one row per scope target")
				return nil
			}))
		})
	}
}
