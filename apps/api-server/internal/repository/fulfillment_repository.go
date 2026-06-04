package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// FulfillmentRepository persists the canonical fulfillment model (processes,
// units, steps, blockers). These tables are tenant-scoped with RLS, so every
// method takes a pgx.Tx obtained from database.WithTenant — the tenant context
// (app.current_tenant_id) both filters reads and constrains writes.
type FulfillmentRepository struct{}

// NewFulfillmentRepository creates a FulfillmentRepository.
func NewFulfillmentRepository() *FulfillmentRepository { return &FulfillmentRepository{} }

func marshalMeta(m any) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func unmarshalMeta(raw []byte, dst *any) {
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, dst)
	}
}

// ---- Processes ----

const fulfillmentProcessColumns = `id, tenant_id, order_id, aggregate_status, health_status, metadata, created_at, updated_at`

func scanProcess(row interface{ Scan(...any) error }) (*model.FulfillmentProcess, error) {
	var p model.FulfillmentProcess
	var meta []byte
	if err := row.Scan(&p.ID, &p.TenantID, &p.OrderID, &p.AggregateStatus, &p.HealthStatus, &meta, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	unmarshalMeta(meta, &p.Metadata)
	return &p, nil
}

// CreateProcess inserts a fulfillment process. AggregateStatus/HealthStatus
// default when empty.
func (r *FulfillmentRepository) CreateProcess(ctx context.Context, tx pgx.Tx, p model.FulfillmentProcess) (*model.FulfillmentProcess, error) {
	if p.AggregateStatus == "" {
		p.AggregateStatus = model.ProcessStatusNew
	}
	if p.HealthStatus == "" {
		p.HealthStatus = model.ProcessHealthOK
	}
	out, err := scanProcess(tx.QueryRow(ctx,
		`INSERT INTO fulfillment_processes (tenant_id, order_id, aggregate_status, health_status, metadata)
		 VALUES ($1,$2,$3,$4,$5::jsonb) RETURNING `+fulfillmentProcessColumns,
		p.TenantID, p.OrderID, p.AggregateStatus, p.HealthStatus, marshalMeta(p.Metadata)))
	if err != nil {
		return nil, fmt.Errorf("create fulfillment process: %w", err)
	}
	return out, nil
}

// GetProcess returns a process by id (RLS-scoped), or pgx.ErrNoRows.
func (r *FulfillmentRepository) GetProcess(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.FulfillmentProcess, error) {
	return scanProcess(tx.QueryRow(ctx, `SELECT `+fulfillmentProcessColumns+` FROM fulfillment_processes WHERE id = $1`, id))
}

// GetProcessByOrder returns the process for an order (RLS-scoped), or pgx.ErrNoRows.
func (r *FulfillmentRepository) GetProcessByOrder(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) (*model.FulfillmentProcess, error) {
	return scanProcess(tx.QueryRow(ctx,
		`SELECT `+fulfillmentProcessColumns+` FROM fulfillment_processes WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`, orderID))
}

// ListProcesses returns the tenant's processes, newest first.
func (r *FulfillmentRepository) ListProcesses(ctx context.Context, tx pgx.Tx) ([]model.FulfillmentProcess, error) {
	rows, err := tx.Query(ctx, `SELECT `+fulfillmentProcessColumns+` FROM fulfillment_processes ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment processes: %w", err)
	}
	defer rows.Close()
	result := []model.FulfillmentProcess{}
	for rows.Next() {
		p, err := scanProcess(rows)
		if err != nil {
			return nil, fmt.Errorf("scan process: %w", err)
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

// UpdateProcessStatus updates a process's aggregate + health status.
func (r *FulfillmentRepository) UpdateProcessStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, aggregate, health string) (*model.FulfillmentProcess, error) {
	out, err := scanProcess(tx.QueryRow(ctx,
		`UPDATE fulfillment_processes SET aggregate_status = $2, health_status = $3, updated_at = now()
		 WHERE id = $1 RETURNING `+fulfillmentProcessColumns,
		id, aggregate, health))
	if err != nil {
		return nil, fmt.Errorf("update process status: %w", err)
	}
	return out, nil
}

// ---- Units ----

const fulfillmentUnitColumns = `id, tenant_id, process_id, parent_unit_id, unit_type, status, metadata, created_at, updated_at`

func scanUnit(row interface{ Scan(...any) error }) (*model.FulfillmentUnit, error) {
	var u model.FulfillmentUnit
	var meta []byte
	if err := row.Scan(&u.ID, &u.TenantID, &u.ProcessID, &u.ParentUnitID, &u.UnitType, &u.Status, &meta, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	unmarshalMeta(meta, &u.Metadata)
	return &u, nil
}

// CreateUnit inserts a fulfillment unit.
func (r *FulfillmentRepository) CreateUnit(ctx context.Context, tx pgx.Tx, u model.FulfillmentUnit) (*model.FulfillmentUnit, error) {
	if u.Status == "" {
		u.Status = model.FulfillmentStatusPending
	}
	out, err := scanUnit(tx.QueryRow(ctx,
		`INSERT INTO fulfillment_units (tenant_id, process_id, parent_unit_id, unit_type, status, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb) RETURNING `+fulfillmentUnitColumns,
		u.TenantID, u.ProcessID, u.ParentUnitID, u.UnitType, u.Status, marshalMeta(u.Metadata)))
	if err != nil {
		return nil, fmt.Errorf("create fulfillment unit: %w", err)
	}
	return out, nil
}

// ListUnits returns a process's units.
func (r *FulfillmentRepository) ListUnits(ctx context.Context, tx pgx.Tx, processID uuid.UUID) ([]model.FulfillmentUnit, error) {
	rows, err := tx.Query(ctx, `SELECT `+fulfillmentUnitColumns+` FROM fulfillment_units WHERE process_id = $1 ORDER BY created_at`, processID)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment units: %w", err)
	}
	defer rows.Close()
	result := []model.FulfillmentUnit{}
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan unit: %w", err)
		}
		result = append(result, *u)
	}
	return result, rows.Err()
}

// UpdateUnitStatus updates a unit's status.
func (r *FulfillmentRepository) UpdateUnitStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) (*model.FulfillmentUnit, error) {
	out, err := scanUnit(tx.QueryRow(ctx,
		`UPDATE fulfillment_units SET status = $2, updated_at = now() WHERE id = $1 RETURNING `+fulfillmentUnitColumns,
		id, status))
	if err != nil {
		return nil, fmt.Errorf("update unit status: %w", err)
	}
	return out, nil
}

// ---- Steps ----

const fulfillmentStepColumns = `id, tenant_id, unit_id, step_key, status, attempts, metadata, created_at, updated_at`

func scanStep(row interface{ Scan(...any) error }) (*model.FulfillmentStep, error) {
	var s model.FulfillmentStep
	var meta []byte
	if err := row.Scan(&s.ID, &s.TenantID, &s.UnitID, &s.StepKey, &s.Status, &s.Attempts, &meta, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	unmarshalMeta(meta, &s.Metadata)
	return &s, nil
}

// CreateStep inserts a fulfillment step.
func (r *FulfillmentRepository) CreateStep(ctx context.Context, tx pgx.Tx, s model.FulfillmentStep) (*model.FulfillmentStep, error) {
	if s.Status == "" {
		s.Status = model.FulfillmentStatusPending
	}
	out, err := scanStep(tx.QueryRow(ctx,
		`INSERT INTO fulfillment_steps (tenant_id, unit_id, step_key, status, attempts, metadata)
		 VALUES ($1,$2,$3,$4,$5,$6::jsonb) RETURNING `+fulfillmentStepColumns,
		s.TenantID, s.UnitID, s.StepKey, s.Status, s.Attempts, marshalMeta(s.Metadata)))
	if err != nil {
		return nil, fmt.Errorf("create fulfillment step: %w", err)
	}
	return out, nil
}

// ListSteps returns a unit's steps.
func (r *FulfillmentRepository) ListSteps(ctx context.Context, tx pgx.Tx, unitID uuid.UUID) ([]model.FulfillmentStep, error) {
	rows, err := tx.Query(ctx, `SELECT `+fulfillmentStepColumns+` FROM fulfillment_steps WHERE unit_id = $1 ORDER BY created_at`, unitID)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment steps: %w", err)
	}
	defer rows.Close()
	result := []model.FulfillmentStep{}
	for rows.Next() {
		s, err := scanStep(rows)
		if err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		result = append(result, *s)
	}
	return result, rows.Err()
}

// UpdateStepStatus updates a step's status and attempt count.
func (r *FulfillmentRepository) UpdateStepStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, attempts int) (*model.FulfillmentStep, error) {
	out, err := scanStep(tx.QueryRow(ctx,
		`UPDATE fulfillment_steps SET status = $2, attempts = $3, updated_at = now() WHERE id = $1 RETURNING `+fulfillmentStepColumns,
		id, status, attempts))
	if err != nil {
		return nil, fmt.Errorf("update step status: %w", err)
	}
	return out, nil
}

// ---- Blockers ----

const fulfillmentBlockerColumns = `id, tenant_id, process_id, unit_id, code, category, status, description, created_at, updated_at, resolved_at`

func scanBlocker(row interface{ Scan(...any) error }) (*model.FulfillmentBlocker, error) {
	var b model.FulfillmentBlocker
	if err := row.Scan(&b.ID, &b.TenantID, &b.ProcessID, &b.UnitID, &b.Code, &b.Category, &b.Status, &b.Description, &b.CreatedAt, &b.UpdatedAt, &b.ResolvedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBlocker inserts a blocker; its category is derived from the code.
func (r *FulfillmentRepository) CreateBlocker(ctx context.Context, tx pgx.Tx, b model.FulfillmentBlocker) (*model.FulfillmentBlocker, error) {
	category := b.Category
	if category == "" {
		category = model.BlockerCategory(b.Code)
	}
	out, err := scanBlocker(tx.QueryRow(ctx,
		`INSERT INTO fulfillment_blockers (tenant_id, process_id, unit_id, code, category, description)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING `+fulfillmentBlockerColumns,
		b.TenantID, b.ProcessID, b.UnitID, b.Code, category, b.Description))
	if err != nil {
		return nil, fmt.Errorf("create fulfillment blocker: %w", err)
	}
	return out, nil
}

// ListBlockers returns a process's blockers, newest first.
func (r *FulfillmentRepository) ListBlockers(ctx context.Context, tx pgx.Tx, processID uuid.UUID) ([]model.FulfillmentBlocker, error) {
	rows, err := tx.Query(ctx, `SELECT `+fulfillmentBlockerColumns+` FROM fulfillment_blockers WHERE process_id = $1 ORDER BY created_at DESC`, processID)
	if err != nil {
		return nil, fmt.Errorf("list fulfillment blockers: %w", err)
	}
	defer rows.Close()
	result := []model.FulfillmentBlocker{}
	for rows.Next() {
		b, err := scanBlocker(rows)
		if err != nil {
			return nil, fmt.Errorf("scan blocker: %w", err)
		}
		result = append(result, *b)
	}
	return result, rows.Err()
}

// ResolveBlocker updates a blocker's status, stamping resolved_at when resolved.
func (r *FulfillmentRepository) ResolveBlocker(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) (*model.FulfillmentBlocker, error) {
	var resolvedAt *time.Time
	if status == model.BlockerStatusResolved {
		now := time.Now().UTC()
		resolvedAt = &now
	}
	out, err := scanBlocker(tx.QueryRow(ctx,
		`UPDATE fulfillment_blockers SET status = $2, resolved_at = $3::timestamptz, updated_at = now()
		 WHERE id = $1 RETURNING `+fulfillmentBlockerColumns,
		id, status, resolvedAt))
	if err != nil {
		return nil, fmt.Errorf("resolve blocker: %w", err)
	}
	return out, nil
}
