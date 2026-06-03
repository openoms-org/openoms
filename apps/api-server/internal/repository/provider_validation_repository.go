package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProviderValidationRepository persists validation probes, runs and results.
// Platform-managed, no RLS.
type ProviderValidationRepository struct {
	pool *pgxpool.Pool
}

// NewProviderValidationRepository creates the repository backed by the pool.
func NewProviderValidationRepository(pool *pgxpool.Pool) *ProviderValidationRepository {
	return &ProviderValidationRepository{pool: pool}
}

const providerProbeColumns = `id, provider_version_id, probe_type, label, destructive, required, config, created_at, updated_at`

func scanProbe(row interface{ Scan(...any) error }) (*model.ProviderValidationProbe, error) {
	var p model.ProviderValidationProbe
	var config []byte
	if err := row.Scan(&p.ID, &p.ProviderVersionID, &p.ProbeType, &p.Label, &p.Destructive, &p.Required, &config, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	if len(config) > 0 {
		_ = json.Unmarshal(config, &p.Config)
	}
	return &p, nil
}

// ReplaceProbes atomically replaces a version's probe set (caller's tx).
func (r *ProviderValidationRepository) ReplaceProbes(ctx context.Context, q Querier, versionID uuid.UUID, probes []model.ProviderValidationProbe) error {
	if _, err := q.Exec(ctx, `DELETE FROM provider_validation_probes WHERE provider_version_id = $1`, versionID); err != nil {
		return fmt.Errorf("clear probes: %w", err)
	}
	for _, p := range probes {
		config, err := json.Marshal(p.Config)
		if err != nil || p.Config == nil {
			config = []byte("{}")
		}
		if _, err := q.Exec(ctx,
			`INSERT INTO provider_validation_probes (provider_version_id, probe_type, label, destructive, required, config)
			 VALUES ($1,$2,$3,$4,$5,$6::jsonb)`,
			versionID, p.ProbeType, p.Label, p.Destructive, p.Required, string(config)); err != nil {
			return fmt.Errorf("insert probe %q: %w", p.Label, err)
		}
	}
	return nil
}

// ListProbes returns a version's probes.
func (r *ProviderValidationRepository) ListProbes(ctx context.Context, versionID uuid.UUID) ([]model.ProviderValidationProbe, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+providerProbeColumns+` FROM provider_validation_probes WHERE provider_version_id = $1 ORDER BY label`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list probes: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderValidationProbe{}
	for rows.Next() {
		p, err := scanProbe(rows)
		if err != nil {
			return nil, fmt.Errorf("scan probe: %w", err)
		}
		result = append(result, *p)
	}
	return result, rows.Err()
}

const providerRunColumns = `id, provider_version_id, environment, verdict, started_by, started_at, finished_at, notes`

func scanRun(row interface{ Scan(...any) error }) (*model.ProviderValidationRun, error) {
	var run model.ProviderValidationRun
	if err := row.Scan(&run.ID, &run.ProviderVersionID, &run.Environment, &run.Verdict, &run.StartedBy, &run.StartedAt, &run.FinishedAt, &run.Notes); err != nil {
		return nil, err
	}
	return &run, nil
}

// CreateRun creates a pending validation run.
func (r *ProviderValidationRepository) CreateRun(ctx context.Context, versionID uuid.UUID, environment string, startedBy *uuid.UUID) (*model.ProviderValidationRun, error) {
	run, err := scanRun(r.pool.QueryRow(ctx,
		`INSERT INTO provider_validation_runs (provider_version_id, environment, started_by)
		 VALUES ($1,$2,$3) RETURNING `+providerRunColumns,
		versionID, environment, startedBy))
	if err != nil {
		return nil, fmt.Errorf("create validation run: %w", err)
	}
	return run, nil
}

// GetRun returns a run header, or pgx.ErrNoRows.
func (r *ProviderValidationRepository) GetRun(ctx context.Context, runID uuid.UUID) (*model.ProviderValidationRun, error) {
	return scanRun(r.pool.QueryRow(ctx, `SELECT `+providerRunColumns+` FROM provider_validation_runs WHERE id = $1`, runID))
}

// ListRuns returns a version's runs, newest first.
func (r *ProviderValidationRepository) ListRuns(ctx context.Context, versionID uuid.UUID) ([]model.ProviderValidationRun, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+providerRunColumns+` FROM provider_validation_runs WHERE provider_version_id = $1 ORDER BY started_at DESC`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderValidationRun{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		result = append(result, *run)
	}
	return result, rows.Err()
}

// FinalizeRun sets a run's verdict and finished_at (caller's tx).
func (r *ProviderValidationRepository) FinalizeRun(ctx context.Context, q Querier, runID uuid.UUID, verdict string, finishedAt time.Time) error {
	_, err := q.Exec(ctx,
		`UPDATE provider_validation_runs SET verdict = $2, finished_at = $3::timestamptz WHERE id = $1`,
		runID, verdict, finishedAt)
	if err != nil {
		return fmt.Errorf("finalize run: %w", err)
	}
	return nil
}

const providerResultColumns = `id, run_id, probe_type, label, status, observation, payload_hash, findings, created_at`

// UpsertResult records (or replaces) a probe result within a run.
func (r *ProviderValidationRepository) UpsertResult(ctx context.Context, result model.ProviderValidationResult) (*model.ProviderValidationResult, error) {
	var out model.ProviderValidationResult
	err := r.pool.QueryRow(ctx,
		`INSERT INTO provider_validation_results (run_id, probe_type, label, status, observation, payload_hash, findings)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (run_id, label) DO UPDATE
		   SET probe_type = EXCLUDED.probe_type, status = EXCLUDED.status, observation = EXCLUDED.observation,
		       payload_hash = EXCLUDED.payload_hash, findings = EXCLUDED.findings
		 RETURNING `+providerResultColumns,
		result.RunID, result.ProbeType, result.Label, result.Status, result.Observation, result.PayloadHash, result.Findings).
		Scan(&out.ID, &out.RunID, &out.ProbeType, &out.Label, &out.Status, &out.Observation, &out.PayloadHash, &out.Findings, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert validation result: %w", err)
	}
	return &out, nil
}

// ListResults returns a run's probe results.
func (r *ProviderValidationRepository) ListResults(ctx context.Context, runID uuid.UUID) ([]model.ProviderValidationResult, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+providerResultColumns+` FROM provider_validation_results WHERE run_id = $1 ORDER BY label`, runID)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderValidationResult{}
	for rows.Next() {
		var res model.ProviderValidationResult
		if err := rows.Scan(&res.ID, &res.RunID, &res.ProbeType, &res.Label, &res.Status, &res.Observation, &res.PayloadHash, &res.Findings, &res.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		result = append(result, res)
	}
	return result, rows.Err()
}
