package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// OrchestrationRepository persists the transactional outbox + attempts. Outbox
// rows are tenant-scoped (RLS): EnqueueEvent runs inside database.WithTenant
// (part of a state-change tx), while the worker's claim/finish methods take a
// Querier — the privileged cross-tenant pool — to process all tenants' due rows.
type OrchestrationRepository struct{}

// NewOrchestrationRepository creates an OrchestrationRepository.
func NewOrchestrationRepository() *OrchestrationRepository { return &OrchestrationRepository{} }

const orchestrationOutboxColumns = `id, tenant_id, process_id, unit_id, event_type, payload, status,
	idempotency_key, attempts, max_attempts, next_attempt_at, claimed_at, last_error, created_at, updated_at`

func scanOutbox(row interface{ Scan(...any) error }) (*model.OrchestrationOutboxEvent, error) {
	var e model.OrchestrationOutboxEvent
	var payload []byte
	if err := row.Scan(&e.ID, &e.TenantID, &e.ProcessID, &e.UnitID, &e.EventType, &payload, &e.Status,
		&e.IdempotencyKey, &e.Attempts, &e.MaxAttempts, &e.NextAttemptAt, &e.ClaimedAt, &e.LastError, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	unmarshalMeta(payload, &e.Payload)
	return &e, nil
}

// EnqueueEvent inserts an outbox event idempotently (deduped on tenant +
// idempotency_key). Returns created=false when the event already existed.
// Runs inside the caller's tenant transaction (RLS-scoped).
func (r *OrchestrationRepository) EnqueueEvent(ctx context.Context, q Querier, e model.OrchestrationOutboxEvent) (bool, *model.OrchestrationOutboxEvent, error) {
	maxAttempts := e.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	out, err := scanOutbox(q.QueryRow(ctx,
		`INSERT INTO orchestration_outbox (tenant_id, process_id, unit_id, event_type, payload, idempotency_key, max_attempts)
		 VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7)
		 ON CONFLICT (tenant_id, idempotency_key) DO NOTHING
		 RETURNING `+orchestrationOutboxColumns,
		e.TenantID, e.ProcessID, e.UnitID, e.EventType, marshalMeta(e.Payload), e.IdempotencyKey, maxAttempts))
	if err == nil {
		return true, out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, nil, fmt.Errorf("enqueue outbox event: %w", err)
	}
	// Conflict: the event already exists — return the existing row. Filter by
	// tenant_id too (the unique key is (tenant_id, idempotency_key)) so this is
	// correct even on a privileged pool that bypasses RLS.
	existing, err := scanOutbox(q.QueryRow(ctx,
		`SELECT `+orchestrationOutboxColumns+` FROM orchestration_outbox WHERE tenant_id = $1 AND idempotency_key = $2`,
		e.TenantID, e.IdempotencyKey))
	if err != nil {
		return false, nil, fmt.Errorf("load existing outbox event: %w", err)
	}
	return false, existing, nil
}

// ClaimDue atomically claims up to limit due (pending, next_attempt_at<=now)
// outbox rows across tenants using FOR UPDATE SKIP LOCKED, so concurrent workers
// never claim the same row. Runs on the privileged cross-tenant pool.
func (r *OrchestrationRepository) ClaimDue(ctx context.Context, q Querier, limit int) ([]model.OrchestrationOutboxEvent, error) {
	rows, err := q.Query(ctx,
		`UPDATE orchestration_outbox SET status = 'claimed', claimed_at = now(), updated_at = now()
		  WHERE id IN (
		      SELECT id FROM orchestration_outbox
		       WHERE status = 'pending' AND next_attempt_at <= now()
		       ORDER BY next_attempt_at
		       LIMIT $1
		       FOR UPDATE SKIP LOCKED
		  )
		 RETURNING `+orchestrationOutboxColumns, limit)
	if err != nil {
		return nil, fmt.Errorf("claim due outbox events: %w", err)
	}
	defer rows.Close()
	result := []model.OrchestrationOutboxEvent{}
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		result = append(result, *e)
	}
	return result, rows.Err()
}

// CountPending returns the number of pending outbox rows across all tenants. It is
// used by the orchestration worker to publish a queue-depth gauge and, like ClaimDue,
// runs on the privileged cross-tenant pool (not RLS-scoped).
func (r *OrchestrationRepository) CountPending(ctx context.Context, q Querier) (int, error) {
	var n int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM orchestration_outbox WHERE status = 'pending'`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count pending outbox events: %w", err)
	}
	return n, nil
}

// MarkSucceeded marks an outbox row as succeeded.
func (r *OrchestrationRepository) MarkSucceeded(ctx context.Context, q Querier, id uuid.UUID) error {
	_, err := q.Exec(ctx, `UPDATE orchestration_outbox SET status='succeeded', updated_at=now() WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("mark outbox succeeded: %w", err)
	}
	return nil
}

// MarkFailedRetry records a retryable failure: increments attempts, re-queues
// the row (status pending) with a future next_attempt_at, and clears the claim.
func (r *OrchestrationRepository) MarkFailedRetry(ctx context.Context, q Querier, id uuid.UUID, errMsg string, nextAttemptAt time.Time) error {
	_, err := q.Exec(ctx,
		`UPDATE orchestration_outbox
		    SET status='pending', attempts = attempts + 1, next_attempt_at = $2::timestamptz,
		        last_error = $3, claimed_at = NULL, updated_at = now()
		  WHERE id = $1`, id, nextAttemptAt, errMsg)
	if err != nil {
		return fmt.Errorf("mark outbox retry: %w", err)
	}
	return nil
}

// MarkFailedPermanent marks an outbox row as permanently failed.
func (r *OrchestrationRepository) MarkFailedPermanent(ctx context.Context, q Querier, id uuid.UUID, errMsg string) error {
	_, err := q.Exec(ctx,
		`UPDATE orchestration_outbox SET status='failed', attempts = attempts + 1, last_error=$2, updated_at=now() WHERE id=$1`,
		id, errMsg)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}

const orchestrationAttemptColumns = `id, tenant_id, outbox_id, attempt_number, status, error, started_at, finished_at`

// StartAttempt records the beginning of an execution attempt.
func (r *OrchestrationRepository) StartAttempt(ctx context.Context, q Querier, tenantID, outboxID uuid.UUID, attemptNumber int) (*model.OrchestrationAttempt, error) {
	var a model.OrchestrationAttempt
	err := q.QueryRow(ctx,
		`INSERT INTO orchestration_attempts (tenant_id, outbox_id, attempt_number, status)
		 VALUES ($1,$2,$3,'running') RETURNING `+orchestrationAttemptColumns,
		tenantID, outboxID, attemptNumber).
		Scan(&a.ID, &a.TenantID, &a.OutboxID, &a.AttemptNumber, &a.Status, &a.Error, &a.StartedAt, &a.FinishedAt)
	if err != nil {
		return nil, fmt.Errorf("start attempt: %w", err)
	}
	return &a, nil
}

// FinishAttempt records the outcome of an attempt.
func (r *OrchestrationRepository) FinishAttempt(ctx context.Context, q Querier, id uuid.UUID, status, errMsg string) error {
	_, err := q.Exec(ctx,
		`UPDATE orchestration_attempts SET status=$2, error=$3, finished_at=now() WHERE id=$1`,
		id, status, errMsg)
	if err != nil {
		return fmt.Errorf("finish attempt: %w", err)
	}
	return nil
}

// GetEvent returns an outbox event by id.
func (r *OrchestrationRepository) GetEvent(ctx context.Context, q Querier, id uuid.UUID) (*model.OrchestrationOutboxEvent, error) {
	return scanOutbox(q.QueryRow(ctx, `SELECT `+orchestrationOutboxColumns+` FROM orchestration_outbox WHERE id = $1`, id))
}

// FindPendingByEventAndNonce loads a still-pending event of the given type whose payload
// carries the correlation nonce AND integration id (tenant-scoped). Used by the callback to
// find the one event it may resolve; not-found / already-resolved -> pgx.ErrNoRows.
func (r *OrchestrationRepository) FindPendingByEventAndNonce(ctx context.Context, q Querier, eventType, nonce string, integrationID uuid.UUID) (*model.OrchestrationOutboxEvent, error) {
	// FOR UPDATE locks the matched row so a concurrent worker claim (FOR UPDATE SKIP LOCKED)
	// at the timeout boundary skips it — the callback and a deadline re-dispatch can never
	// interleave on the same event, so a racing callback can't overwrite a just-claimed row.
	row := q.QueryRow(ctx, `SELECT `+orchestrationOutboxColumns+`
		FROM orchestration_outbox
		WHERE status = 'pending' AND event_type = $1
		  AND payload->>'correlation_nonce' = $2
		  AND payload->>'integration_id' = $3
		FOR UPDATE`, eventType, nonce, integrationID.String())
	return scanOutbox(row)
}

// HasDispatchedAttempt reports whether the outbox event carries explicit dispatch evidence:
// an attempt that finished succeeded. The orchestration worker records the succeeded attempt
// in the SAME transaction as the pending-callback park (DeferUntil), and a plainly succeeded
// event never re-enters dispatch — so for a still-pending event this is exactly "the outbound
// POST succeeded earlier and the event is awaiting its callback" (OPE-514). Filters by
// tenant_id like its siblings, so it is correct on both RLS-scoped transactions and the
// privileged pool.
func (r *OrchestrationRepository) HasDispatchedAttempt(ctx context.Context, q Querier, tenantID, outboxID uuid.UUID) (bool, error) {
	var dispatched bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS (
		    SELECT 1 FROM orchestration_attempts
		     WHERE tenant_id = $1 AND outbox_id = $2 AND status = 'succeeded'
		 )`, tenantID, outboxID).Scan(&dispatched); err != nil {
		return false, fmt.Errorf("check dispatched attempt: %w", err)
	}
	return dispatched, nil
}

// SetLatestAttemptExternalExecID records the external engine's execution id on the most
// recent attempt of an outbox event (best-effort).
func (r *OrchestrationRepository) SetLatestAttemptExternalExecID(ctx context.Context, q Querier, outboxID uuid.UUID, execID string) error {
	_, err := q.Exec(ctx, `UPDATE orchestration_attempts SET external_execution_id = $2
		WHERE id = (SELECT id FROM orchestration_attempts WHERE outbox_id = $1 ORDER BY attempt_number DESC LIMIT 1)`,
		outboxID, execID)
	return err
}

// ListByProcess returns a process's outbox events (RLS-scoped; pass a WithTenant tx).
func (r *OrchestrationRepository) ListByProcess(ctx context.Context, q Querier, processID uuid.UUID) ([]model.OrchestrationOutboxEvent, error) {
	rows, err := q.Query(ctx, `SELECT `+orchestrationOutboxColumns+` FROM orchestration_outbox WHERE process_id = $1 ORDER BY created_at`, processID)
	if err != nil {
		return nil, fmt.Errorf("list outbox by process: %w", err)
	}
	defer rows.Close()
	result := []model.OrchestrationOutboxEvent{}
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		result = append(result, *e)
	}
	return result, rows.Err()
}
