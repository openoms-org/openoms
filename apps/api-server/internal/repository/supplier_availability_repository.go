package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// SupplierAvailabilityRepository is tenant-scoped: every method takes a pgx.Tx obtained
// from database.WithTenant, so PostgreSQL RLS scopes all rows to the current tenant.
type SupplierAvailabilityRepository struct{}

// NewSupplierAvailabilityRepository constructs the repository.
func NewSupplierAvailabilityRepository() *SupplierAvailabilityRepository {
	return &SupplierAvailabilityRepository{}
}

const supplierAvailabilityColumns = `id, tenant_id, supplier_id, supplier_product_id, product_id,
	warehouse_external_id, source_quantity, availability_type, min_handling_days, max_handling_days,
	next_delivery_date, reservation_supported, freshness_observed_at, source_max_stale_seconds,
	last_successful_sync_id, raw, created_at, updated_at`

// UpsertSnapshot inserts or updates the snapshot for (tenant, supplier_product, warehouse),
// keyed by the uq_supplier_availability_product_wh unique index. Idempotent.
func (r *SupplierAvailabilityRepository) UpsertSnapshot(ctx context.Context, tx pgx.Tx, a model.SupplierAvailability) (*model.SupplierAvailability, error) {
	raw := a.Raw
	if raw == nil {
		raw = []byte("{}")
	}
	out, err := scanSupplierAvailability(tx.QueryRow(ctx,
		`INSERT INTO supplier_availability
		   (tenant_id, supplier_id, supplier_product_id, product_id, warehouse_external_id,
		    source_quantity, availability_type, min_handling_days, max_handling_days, next_delivery_date,
		    reservation_supported, freshness_observed_at, source_max_stale_seconds, last_successful_sync_id, raw)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb)
		 ON CONFLICT (tenant_id, supplier_product_id, warehouse_external_id) DO UPDATE SET
		    product_id = EXCLUDED.product_id,
		    source_quantity = EXCLUDED.source_quantity,
		    availability_type = EXCLUDED.availability_type,
		    min_handling_days = EXCLUDED.min_handling_days,
		    max_handling_days = EXCLUDED.max_handling_days,
		    next_delivery_date = EXCLUDED.next_delivery_date,
		    reservation_supported = EXCLUDED.reservation_supported,
		    freshness_observed_at = EXCLUDED.freshness_observed_at,
		    source_max_stale_seconds = EXCLUDED.source_max_stale_seconds,
		    last_successful_sync_id = EXCLUDED.last_successful_sync_id,
		    raw = EXCLUDED.raw,
		    updated_at = now()
		 RETURNING `+supplierAvailabilityColumns,
		a.TenantID, a.SupplierID, a.SupplierProductID, a.ProductID, a.WarehouseExternalID,
		a.SourceQuantity, a.AvailabilityType, a.MinHandlingDays, a.MaxHandlingDays, a.NextDeliveryDate,
		a.ReservationSupported, a.FreshnessObservedAt, a.SourceMaxStaleSeconds, a.LastSuccessfulSyncID, string(raw)))
	if err != nil {
		return nil, fmt.Errorf("upsert supplier availability: %w", err)
	}
	return out, nil
}

// ListSnapshotsByProduct returns all warehouse snapshots for a product (tenant-scoped).
func (r *SupplierAvailabilityRepository) ListSnapshotsByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]model.SupplierAvailability, error) {
	rows, err := tx.Query(ctx, `SELECT `+supplierAvailabilityColumns+`
		FROM supplier_availability WHERE product_id = $1 ORDER BY warehouse_external_id`, productID)
	if err != nil {
		return nil, fmt.Errorf("list supplier availability by product: %w", err)
	}
	defer rows.Close()
	out := []model.SupplierAvailability{}
	for rows.Next() {
		a, e := scanSupplierAvailabilityRows(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

const supplierPolicyColumns = `id, tenant_id, scope, supplier_id, product_id, listing_id, channel,
	mode, safety_buffer, freshness_window_seconds, max_lead_time_days, override_quantity,
	allow_channel_increase, require_reservation, require_preflight, created_at, updated_at`

// ListPoliciesForContext loads every policy row that could apply to a (supplier, product,
// listing, channel) context — the caller orders them least->most specific and folds with
// model.ResolvePolicyChain. listingID/channel may be nil/empty when not applicable.
func (r *SupplierAvailabilityRepository) ListPoliciesForContext(ctx context.Context, tx pgx.Tx, supplierID, productID uuid.UUID, listingID *uuid.UUID, channel *string) ([]model.SupplierAvailabilityPolicy, error) {
	rows, err := tx.Query(ctx, `SELECT `+supplierPolicyColumns+`
		FROM supplier_availability_policy
		WHERE (scope = 'supplier' AND supplier_id = $1)
		   OR (scope = 'product'  AND supplier_id = $1 AND product_id = $2)
		   OR (scope = 'listing'  AND listing_id = $3)
		   OR (scope = 'channel'  AND channel = $4)`,
		supplierID, productID, listingID, channel)
	if err != nil {
		return nil, fmt.Errorf("list policies for context: %w", err)
	}
	defer rows.Close()
	out := []model.SupplierAvailabilityPolicy{}
	for rows.Next() {
		p, e := scanSupplierPolicyRows(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// UpsertPolicy inserts or updates a single scope policy (keyed by the partial unique index
// for its scope). Idempotent per scope target.
func (r *SupplierAvailabilityRepository) UpsertPolicy(ctx context.Context, tx pgx.Tx, p model.SupplierAvailabilityPolicy) (*model.SupplierAvailabilityPolicy, error) {
	out, err := scanSupplierPolicy(tx.QueryRow(ctx,
		`INSERT INTO supplier_availability_policy
		   (tenant_id, scope, supplier_id, product_id, listing_id, channel, mode, safety_buffer,
		    freshness_window_seconds, max_lead_time_days, override_quantity, allow_channel_increase,
		    require_reservation, require_preflight)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING `+supplierPolicyColumns,
		p.TenantID, p.Scope, p.SupplierID, p.ProductID, p.ListingID, p.Channel, p.Mode, p.SafetyBuffer,
		p.FreshnessWindowSecs, p.MaxLeadTimeDays, p.OverrideQuantity, p.AllowChannelIncrease,
		p.RequireReservation, p.RequirePreflight))
	if err != nil {
		return nil, fmt.Errorf("upsert supplier availability policy: %w", err)
	}
	return out, nil
}

func scanSupplierAvailability(row pgx.Row) (*model.SupplierAvailability, error) {
	var a model.SupplierAvailability
	if err := row.Scan(&a.ID, &a.TenantID, &a.SupplierID, &a.SupplierProductID, &a.ProductID,
		&a.WarehouseExternalID, &a.SourceQuantity, &a.AvailabilityType, &a.MinHandlingDays, &a.MaxHandlingDays,
		&a.NextDeliveryDate, &a.ReservationSupported, &a.FreshnessObservedAt, &a.SourceMaxStaleSeconds,
		&a.LastSuccessfulSyncID, &a.Raw, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan supplier availability: %w", err)
	}
	return &a, nil
}

func scanSupplierAvailabilityRows(rows pgx.Rows) (*model.SupplierAvailability, error) {
	return scanSupplierAvailability(rows)
}

func scanSupplierPolicy(row pgx.Row) (*model.SupplierAvailabilityPolicy, error) {
	var p model.SupplierAvailabilityPolicy
	if err := row.Scan(&p.ID, &p.TenantID, &p.Scope, &p.SupplierID, &p.ProductID, &p.ListingID, &p.Channel,
		&p.Mode, &p.SafetyBuffer, &p.FreshnessWindowSecs, &p.MaxLeadTimeDays, &p.OverrideQuantity,
		&p.AllowChannelIncrease, &p.RequireReservation, &p.RequirePreflight, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan supplier policy: %w", err)
	}
	return &p, nil
}

func scanSupplierPolicyRows(rows pgx.Rows) (*model.SupplierAvailabilityPolicy, error) {
	return scanSupplierPolicy(rows)
}
