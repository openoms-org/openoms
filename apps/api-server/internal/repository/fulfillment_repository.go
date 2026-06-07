package repository

import (
	"context"
	"encoding/json"
	"errors"
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
	// ON CONFLICT (tenant_id, order_id) DO NOTHING makes process creation idempotent and
	// race-safe against the uq_fulfillment_processes_tenant_order unique index (migration
	// 000040): all callers GetProcessByOrder-first, but under READ COMMITTED two concurrent
	// creators can both miss the existing row and both insert. ON CONFLICT collapses that to
	// a single row WITHOUT aborting the surrounding transaction (a bare unique-violation would
	// abort it). On a conflict the INSERT returns no row, so we re-fetch the winner.
	out, err := scanProcess(tx.QueryRow(ctx,
		`INSERT INTO fulfillment_processes (tenant_id, order_id, aggregate_status, health_status, metadata)
		 VALUES ($1,$2,$3,$4,$5::jsonb)
		 ON CONFLICT (tenant_id, order_id) DO NOTHING
		 RETURNING `+fulfillmentProcessColumns,
		p.TenantID, p.OrderID, p.AggregateStatus, p.HealthStatus, marshalMeta(p.Metadata)))
	if errors.Is(err, pgx.ErrNoRows) {
		// Lost the create race — a process already exists for this order; return it.
		return r.GetProcessByOrder(ctx, tx, p.OrderID)
	}
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

// ListOrderIDsMissingProcess returns up to limit order ids (oldest first) for the
// current tenant whose status is NOT terminal and which have NO fulfillment process
// yet (RLS-scoped via tx). It is the eligibility query for the OPE-423a backfill:
// the LEFT JOIN ... WHERE fp.id IS NULL selects exactly the still-missing set, so
// the backfill is RESUMABLE by construction — already-processed orders fall out of
// the result on the next call. terminalStatuses must be the caller's terminal order
// status list (model.TerminalOrderStatuses); an empty list matches every status.
func (r *FulfillmentRepository) ListOrderIDsMissingProcess(ctx context.Context, tx pgx.Tx, terminalStatuses []string, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 100
	}
	// The explicit o.tenant_id predicate is defense-in-depth: RLS (FORCE ROW LEVEL
	// SECURITY on orders + fulfillment_processes) already scopes this to the tx's
	// app.current_tenant_id, but the predicate keeps the backfill tenant-safe even if
	// the worker pool role were ever misconfigured with BYPASSRLS.
	rows, err := tx.Query(ctx,
		`SELECT o.id
		   FROM orders o
		   LEFT JOIN fulfillment_processes fp ON fp.order_id = o.id
		  WHERE fp.id IS NULL
		    AND NOT (o.status = ANY($1))
		    AND o.tenant_id = current_setting('app.current_tenant_id', true)::uuid
		  ORDER BY o.created_at
		  LIMIT $2`, terminalStatuses, limit)
	if err != nil {
		return nil, fmt.Errorf("list orders missing fulfillment process: %w", err)
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan order id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CountNonTerminalOrders returns how many of the tenant's orders are NOT in a
// terminal status (RLS-scoped via tx). This is the LEGACY "active" order population
// — every order still flowing through fulfillment — and the denominator for the
// OPE-423 parity report's process coverage. terminalStatuses must be the caller's
// terminal order status list (model.TerminalOrderStatuses); an empty list counts
// every order. The explicit tenant_id predicate is defense-in-depth alongside RLS,
// mirroring ListOrderIDsMissingProcess.
func (r *FulfillmentRepository) CountNonTerminalOrders(ctx context.Context, tx pgx.Tx, terminalStatuses []string) (int, error) {
	var n int
	err := tx.QueryRow(ctx,
		`SELECT count(*)
		   FROM orders o
		  WHERE NOT (o.status = ANY($1))
		    AND o.tenant_id = current_setting('app.current_tenant_id', true)::uuid`,
		terminalStatuses).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count non-terminal orders: %w", err)
	}
	return n, nil
}

// CountOrdersMissingProcess returns how many of the tenant's non-terminal orders
// have NO fulfillment process yet (RLS-scoped via tx). It is the COUNT analogue of
// ListOrderIDsMissingProcess (same LEFT JOIN ... fp.id IS NULL eligibility set) and
// feeds the OPE-423 parity report: this number must trend to 0 after the OPE-423a
// backfill completes. terminalStatuses must be model.TerminalOrderStatuses; an empty
// list matches every status.
func (r *FulfillmentRepository) CountOrdersMissingProcess(ctx context.Context, tx pgx.Tx, terminalStatuses []string) (int, error) {
	var n int
	err := tx.QueryRow(ctx,
		`SELECT count(*)
		   FROM orders o
		   LEFT JOIN fulfillment_processes fp ON fp.order_id = o.id
		  WHERE fp.id IS NULL
		    AND NOT (o.status = ANY($1))
		    AND o.tenant_id = current_setting('app.current_tenant_id', true)::uuid`,
		terminalStatuses).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count orders missing fulfillment process: %w", err)
	}
	return n, nil
}

// CountLegacyProblemOrders returns how many of the tenant's non-terminal orders the
// LEGACY operations dashboard would flag as "needs attention" (RLS-scoped via tx).
// The legacy heuristic (apps/dashboard/src/lib/operations-dashboard.ts) flags an
// order when it is on_hold OR has a problem shipment, so this counts the DISTINCT
// union of: (a) non-terminal orders whose status is on_hold, and (b) non-terminal
// orders with at least one shipment whose status is in problemShipmentStatuses
// (e.g. {failed, error}). It is the legacy comparison point for the parity report's
// ProcessBackedExceptions. terminalStatuses must be model.TerminalOrderStatuses.
//
// HONEST CAVEAT: the legacy dashboard ALSO surfaces integration-error exceptions,
// but those are NOT order-scoped (an integration in error state is not an order), so
// they are intentionally excluded here — this count is the order-comparable subset.
func (r *FulfillmentRepository) CountLegacyProblemOrders(ctx context.Context, tx pgx.Tx, terminalStatuses, problemShipmentStatuses []string) (int, error) {
	var n int
	err := tx.QueryRow(ctx,
		`SELECT count(DISTINCT o.id)
		   FROM orders o
		  WHERE NOT (o.status = ANY($1))
		    AND o.tenant_id = current_setting('app.current_tenant_id', true)::uuid
		    AND (
		      o.status = 'on_hold'
		      OR EXISTS (
		        SELECT 1 FROM shipments s
		         WHERE s.order_id = o.id AND s.status = ANY($2)
		      )
		    )`,
		terminalStatuses, problemShipmentStatuses).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count legacy problem orders: %w", err)
	}
	return n, nil
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

// ProcessListFilter narrows a paginated process listing. Empty filter fields are
// ignored. AggregateStatuses / HealthStatuses are OR-ed within a field and AND-ed
// across fields. The caller is responsible for validating the status values; only
// known values should reach the repository (invalid ones simply match nothing).
type ProcessListFilter struct {
	AggregateStatuses []string
	HealthStatuses    []string
	Limit             int
	Offset            int
}

// ListProcessesFiltered returns a tenant's processes (newest first) matching the
// optional aggregate_status / health_status filters, plus the total count of
// matching rows (ignoring limit/offset) for pagination. RLS-scoped via tx.
func (r *FulfillmentRepository) ListProcessesFiltered(ctx context.Context, tx pgx.Tx, f ProcessListFilter) ([]model.FulfillmentProcess, int, error) {
	where := ""
	args := []any{}
	if len(f.AggregateStatuses) > 0 {
		args = append(args, f.AggregateStatuses)
		where += fmt.Sprintf(" AND aggregate_status = ANY($%d)", len(args))
	}
	if len(f.HealthStatuses) > 0 {
		args = append(args, f.HealthStatuses)
		where += fmt.Sprintf(" AND health_status = ANY($%d)", len(args))
	}

	var total int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM fulfillment_processes WHERE 1=1`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fulfillment processes: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := max(f.Offset, 0)
	args = append(args, limit, offset)
	query := `SELECT ` + fulfillmentProcessColumns + ` FROM fulfillment_processes WHERE 1=1` + where +
		fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list fulfillment processes (filtered): %w", err)
	}
	defer rows.Close()
	result := []model.FulfillmentProcess{}
	for rows.Next() {
		p, err := scanProcess(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan process: %w", err)
		}
		result = append(result, *p)
	}
	return result, total, rows.Err()
}

// ListProcessesByAggregateStatuses returns the tenant's processes whose aggregate
// status is one of statuses, newest first (RLS-scoped). Used to build the
// actionable exceptions list (blocked / waiting_external / stuck).
func (r *FulfillmentRepository) ListProcessesByAggregateStatuses(ctx context.Context, tx pgx.Tx, statuses []string, limit int) ([]model.FulfillmentProcess, error) {
	if len(statuses) == 0 {
		return []model.FulfillmentProcess{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := tx.Query(ctx,
		`SELECT `+fulfillmentProcessColumns+`
		   FROM fulfillment_processes
		  WHERE aggregate_status = ANY($1)
		  ORDER BY updated_at DESC
		  LIMIT $2`, statuses, limit)
	if err != nil {
		return nil, fmt.Errorf("list processes by aggregate status: %w", err)
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

// ProcessStatusCount is one (aggregate_status, health_status, count) bucket.
type ProcessStatusCount struct {
	AggregateStatus string
	HealthStatus    string
	Count           int
}

// CountProcessesByStatus returns the per-(aggregate_status, health_status) row
// counts for the tenant (RLS-scoped). The operator summary buckets are derived
// from these in the service layer.
func (r *FulfillmentRepository) CountProcessesByStatus(ctx context.Context, tx pgx.Tx) ([]ProcessStatusCount, error) {
	rows, err := tx.Query(ctx,
		`SELECT aggregate_status, health_status, count(*)
		   FROM fulfillment_processes
		  GROUP BY aggregate_status, health_status`)
	if err != nil {
		return nil, fmt.Errorf("count processes by status: %w", err)
	}
	defer rows.Close()
	result := []ProcessStatusCount{}
	for rows.Next() {
		var c ProcessStatusCount
		if err := rows.Scan(&c.AggregateStatus, &c.HealthStatus, &c.Count); err != nil {
			return nil, fmt.Errorf("scan process status count: %w", err)
		}
		result = append(result, c)
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

// GetUnit returns a unit by id (RLS-scoped), or pgx.ErrNoRows.
func (r *FulfillmentRepository) GetUnit(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.FulfillmentUnit, error) {
	return scanUnit(tx.QueryRow(ctx, `SELECT `+fulfillmentUnitColumns+` FROM fulfillment_units WHERE id = $1`, id))
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

// GetStep returns a step by id (RLS-scoped), or pgx.ErrNoRows.
func (r *FulfillmentRepository) GetStep(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.FulfillmentStep, error) {
	return scanStep(tx.QueryRow(ctx, `SELECT `+fulfillmentStepColumns+` FROM fulfillment_steps WHERE id = $1`, id))
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

// GetBlocker returns a blocker by id (RLS-scoped), or pgx.ErrNoRows.
func (r *FulfillmentRepository) GetBlocker(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.FulfillmentBlocker, error) {
	return scanBlocker(tx.QueryRow(ctx, `SELECT `+fulfillmentBlockerColumns+` FROM fulfillment_blockers WHERE id = $1`, id))
}

// ListOpenBlockersByProcess returns a process's not-yet-resolved blockers (status
// open or acknowledged), newest first (RLS-scoped). Used by the exceptions feed to
// attach the top actionable blocker to a process.
func (r *FulfillmentRepository) ListOpenBlockersByProcess(ctx context.Context, tx pgx.Tx, processID uuid.UUID) ([]model.FulfillmentBlocker, error) {
	rows, err := tx.Query(ctx,
		`SELECT `+fulfillmentBlockerColumns+`
		   FROM fulfillment_blockers
		  WHERE process_id = $1 AND status <> 'resolved'
		  ORDER BY created_at DESC`, processID)
	if err != nil {
		return nil, fmt.Errorf("list open blockers by process: %w", err)
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
