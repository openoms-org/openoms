package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProviderVersionRepository persists versioned provider configurations.
type ProviderVersionRepository struct {
	pool *pgxpool.Pool
}

// NewProviderVersionRepository creates the repository backed by the pool.
func NewProviderVersionRepository(pool *pgxpool.Pool) *ProviderVersionRepository {
	return &ProviderVersionRepository{pool: pool}
}

const providerVersionColumns = `id, provider_definition_id, version, publication_state, changelog,
	compatibility_notes, created_by, published_by, published_at, created_at, updated_at`

func scanProviderVersion(row interface{ Scan(dest ...any) error }) (*model.ProviderVersion, error) {
	var v model.ProviderVersion
	if err := row.Scan(&v.ID, &v.ProviderDefinitionID, &v.Version, &v.PublicationState, &v.Changelog,
		&v.CompatibilityNotes, &v.CreatedBy, &v.PublishedBy, &v.PublishedAt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, err
	}
	return &v, nil
}

// Create inserts a new version (caller sets the initial state, typically research).
func (r *ProviderVersionRepository) Create(ctx context.Context, q Querier, v model.ProviderVersion) (*model.ProviderVersion, error) {
	state := v.PublicationState
	if state == "" {
		state = model.ProviderStateResearch
	}
	out, err := scanProviderVersion(q.QueryRow(ctx,
		`INSERT INTO provider_versions (provider_definition_id, version, publication_state, changelog, compatibility_notes, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+providerVersionColumns,
		v.ProviderDefinitionID, v.Version, state, v.Changelog, v.CompatibilityNotes, v.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("create provider version: %w", err)
	}
	return out, nil
}

// GetByID returns a version, or pgx.ErrNoRows.
func (r *ProviderVersionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ProviderVersion, error) {
	return scanProviderVersion(r.pool.QueryRow(ctx,
		`SELECT `+providerVersionColumns+` FROM provider_versions WHERE id = $1`, id))
}

// ListByDefinition returns all versions for a definition, newest first.
func (r *ProviderVersionRepository) ListByDefinition(ctx context.Context, defID uuid.UUID) ([]model.ProviderVersion, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+providerVersionColumns+` FROM provider_versions WHERE provider_definition_id = $1 ORDER BY created_at DESC`, defID)
	if err != nil {
		return nil, fmt.Errorf("list provider versions: %w", err)
	}
	defer rows.Close()
	result := []model.ProviderVersion{}
	for rows.Next() {
		v, err := scanProviderVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider version: %w", err)
		}
		result = append(result, *v)
	}
	return result, rows.Err()
}

// UpdateMetadata updates the editable changelog/compatibility fields. State is
// not changed here (use UpdateState). Immutability is enforced in the service,
// which passes its transaction so the freeze check and the write are atomic.
func (r *ProviderVersionRepository) UpdateMetadata(ctx context.Context, q Querier, id uuid.UUID, changelog, compat string) (*model.ProviderVersion, error) {
	out, err := scanProviderVersion(q.QueryRow(ctx,
		`UPDATE provider_versions SET changelog = $2, compatibility_notes = $3, updated_at = now()
		  WHERE id = $1 RETURNING `+providerVersionColumns,
		id, changelog, compat))
	if err != nil {
		return nil, fmt.Errorf("update provider version metadata: %w", err)
	}
	return out, nil
}

// UpdateState transitions a version's publication state. publishedBy/publishedAt
// are set only when provided (i.e. when entering 'available').
func (r *ProviderVersionRepository) UpdateState(ctx context.Context, q Querier, id uuid.UUID, state string, publishedBy *uuid.UUID, publishedAt *time.Time) (*model.ProviderVersion, error) {
	out, err := scanProviderVersion(q.QueryRow(ctx,
		`UPDATE provider_versions
		    SET publication_state = $2,
		        published_by = COALESCE($3, published_by),
		        published_at = COALESCE($4, published_at),
		        updated_at = now()
		  WHERE id = $1 RETURNING `+providerVersionColumns,
		id, state, publishedBy, publishedAt))
	if err != nil {
		return nil, fmt.Errorf("update provider version state: %w", err)
	}
	return out, nil
}

// LatestAvailableExcluding returns the most recently published version still in
// 'available' state for a definition, excluding the given version (used to roll
// back the definition's latest-published pointer on emergency disable). Returns
// nil (and no error) when there is no other available version.
func (r *ProviderVersionRepository) LatestAvailableExcluding(ctx context.Context, q Querier, defID, excludeVersionID uuid.UUID) (*model.ProviderVersion, error) {
	v, err := scanProviderVersion(q.QueryRow(ctx,
		`SELECT `+providerVersionColumns+`
		   FROM provider_versions
		  WHERE provider_definition_id = $1 AND id <> $2 AND publication_state = 'available'
		  ORDER BY published_at DESC NULLS LAST, created_at DESC
		  LIMIT 1`, defID, excludeVersionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find latest available version: %w", err)
	}
	return v, nil
}
