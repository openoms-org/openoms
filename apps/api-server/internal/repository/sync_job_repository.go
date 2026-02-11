package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type SyncJobRepository struct{}

func NewSyncJobRepository() *SyncJobRepository {
	return &SyncJobRepository{}
}

func (r *SyncJobRepository) Create(ctx context.Context, tx pgx.Tx, job *model.SyncJob) error {
	return tx.QueryRow(ctx,
		`INSERT INTO sync_jobs (
			id, tenant_id, integration_id, job_type, status,
			started_at, items_processed, items_failed, error_message, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at`,
		job.ID, job.TenantID, job.IntegrationID, job.JobType, job.Status,
		job.StartedAt, job.ItemsProcessed, job.ItemsFailed, job.ErrorMessage, job.Metadata,
	).Scan(&job.CreatedAt)
}

func (r *SyncJobRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, itemsProcessed, itemsFailed int, errorMsg *string) error {
	ct, err := tx.Exec(ctx,
		`UPDATE sync_jobs
		 SET status = $1, items_processed = $2, items_failed = $3, error_message = $4,
		     finished_at = CASE WHEN $1 IN ('completed', 'failed') THEN NOW() ELSE finished_at END
		 WHERE id = $5`,
		status, itemsProcessed, itemsFailed, errorMsg, id,
	)
	if err != nil {
		return fmt.Errorf("update sync job status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("sync job not found")
	}
	return nil
}

func (r *SyncJobRepository) GetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.SyncJob, error) {
	var j model.SyncJob
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, integration_id, job_type, status,
		        started_at, finished_at, items_processed, items_failed,
		        error_message, metadata, created_at
		 FROM sync_jobs WHERE id = $1`, id,
	).Scan(
		&j.ID, &j.TenantID, &j.IntegrationID, &j.JobType, &j.Status,
		&j.StartedAt, &j.FinishedAt, &j.ItemsProcessed, &j.ItemsFailed,
		&j.ErrorMessage, &j.Metadata, &j.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find sync job by id: %w", err)
	}
	return &j, nil
}

func (r *SyncJobRepository) ListByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID, limit int) ([]*model.SyncJob, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, integration_id, job_type, status,
		        started_at, finished_at, items_processed, items_failed,
		        error_message, metadata, created_at
		 FROM sync_jobs WHERE integration_id = $1
		 ORDER BY created_at DESC LIMIT $2`, integrationID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list sync jobs by integration: %w", err)
	}
	defer rows.Close()

	var jobs []*model.SyncJob
	for rows.Next() {
		var j model.SyncJob
		if err := rows.Scan(
			&j.ID, &j.TenantID, &j.IntegrationID, &j.JobType, &j.Status,
			&j.StartedAt, &j.FinishedAt, &j.ItemsProcessed, &j.ItemsFailed,
			&j.ErrorMessage, &j.Metadata, &j.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sync job: %w", err)
		}
		jobs = append(jobs, &j)
	}
	return jobs, rows.Err()
}

func (r *SyncJobRepository) List(ctx context.Context, tx pgx.Tx, filter model.SyncJobListFilter) ([]*model.SyncJob, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.IntegrationID != nil {
		conditions = append(conditions, fmt.Sprintf("integration_id = $%d", argIdx))
		args = append(args, *filter.IntegrationID)
		argIdx++
	}
	if filter.JobType != nil {
		conditions = append(conditions, fmt.Sprintf("job_type = $%d", argIdx))
		args = append(args, *filter.JobType)
		argIdx++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM sync_jobs %s", where)
	var total int
	if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sync jobs: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	query := fmt.Sprintf(
		`SELECT id, tenant_id, integration_id, job_type, status,
		        started_at, finished_at, items_processed, items_failed,
		        error_message, metadata, created_at
		 FROM sync_jobs
		 %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, limit, filter.Offset)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sync jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.SyncJob
	for rows.Next() {
		var j model.SyncJob
		if err := rows.Scan(
			&j.ID, &j.TenantID, &j.IntegrationID, &j.JobType, &j.Status,
			&j.StartedAt, &j.FinishedAt, &j.ItemsProcessed, &j.ItemsFailed,
			&j.ErrorMessage, &j.Metadata, &j.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan sync job: %w", err)
		}
		jobs = append(jobs, &j)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}
