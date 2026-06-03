package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProviderDefinitionRepository persists provider-family definitions. The
// provider_* tables are platform-managed, NOT tenant-scoped, and have no RLS;
// queries run directly on the pool outside any tenant context.
type ProviderDefinitionRepository struct {
	pool *pgxpool.Pool
}

// NewProviderDefinitionRepository creates the repository backed by the pool.
func NewProviderDefinitionRepository(pool *pgxpool.Pool) *ProviderDefinitionRepository {
	return &ProviderDefinitionRepository{pool: pool}
}

const providerDefinitionColumns = `id, provider_key, display_name, provider_type, regions, business_domains,
	owner, source_links, notes, latest_published_version_id, created_at, updated_at`

func scanProviderDefinition(row interface {
	Scan(dest ...any) error
}) (*model.ProviderDefinition, error) {
	var d model.ProviderDefinition
	var sourceLinks []byte
	if err := row.Scan(&d.ID, &d.ProviderKey, &d.DisplayName, &d.ProviderType, &d.Regions, &d.BusinessDomains,
		&d.Owner, &sourceLinks, &d.Notes, &d.LatestPublishedVersionID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	if len(sourceLinks) > 0 {
		if err := json.Unmarshal(sourceLinks, &d.SourceLinks); err != nil {
			return nil, fmt.Errorf("decode source_links: %w", err)
		}
	}
	if d.Regions == nil {
		d.Regions = []string{}
	}
	if d.BusinessDomains == nil {
		d.BusinessDomains = []string{}
	}
	if d.SourceLinks == nil {
		d.SourceLinks = []string{}
	}
	return &d, nil
}

// Create inserts a new provider definition (state implied by its versions).
func (r *ProviderDefinitionRepository) Create(ctx context.Context, d model.ProviderDefinition) (*model.ProviderDefinition, error) {
	sourceLinks, err := json.Marshal(orEmptySlice(d.SourceLinks))
	if err != nil {
		return nil, fmt.Errorf("encode source_links: %w", err)
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO provider_definitions (provider_key, display_name, provider_type, regions, business_domains, owner, source_links, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
		 RETURNING `+providerDefinitionColumns,
		d.ProviderKey, d.DisplayName, d.ProviderType, orEmptySlice(d.Regions), orEmptySlice(d.BusinessDomains),
		d.Owner, string(sourceLinks), d.Notes)
	def, err := scanProviderDefinition(row)
	if err != nil {
		return nil, fmt.Errorf("create provider definition: %w", err)
	}
	return def, nil
}

// GetByID returns a provider definition, or pgx.ErrNoRows.
func (r *ProviderDefinitionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ProviderDefinition, error) {
	return scanProviderDefinition(r.pool.QueryRow(ctx,
		`SELECT `+providerDefinitionColumns+` FROM provider_definitions WHERE id = $1`, id))
}

// GetByKey returns a provider definition by its provider_key, or pgx.ErrNoRows.
func (r *ProviderDefinitionRepository) GetByKey(ctx context.Context, key string) (*model.ProviderDefinition, error) {
	return scanProviderDefinition(r.pool.QueryRow(ctx,
		`SELECT `+providerDefinitionColumns+` FROM provider_definitions WHERE provider_key = $1`, key))
}

// List returns all provider definitions, newest first.
func (r *ProviderDefinitionRepository) List(ctx context.Context) ([]model.ProviderDefinition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+providerDefinitionColumns+` FROM provider_definitions ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list provider definitions: %w", err)
	}
	defer rows.Close()

	result := []model.ProviderDefinition{}
	for rows.Next() {
		d, err := scanProviderDefinition(rows)
		if err != nil {
			return nil, fmt.Errorf("scan provider definition: %w", err)
		}
		result = append(result, *d)
	}
	return result, rows.Err()
}

// UpdateMetadata updates editable definition metadata.
func (r *ProviderDefinitionRepository) UpdateMetadata(ctx context.Context, d model.ProviderDefinition) (*model.ProviderDefinition, error) {
	sourceLinks, err := json.Marshal(orEmptySlice(d.SourceLinks))
	if err != nil {
		return nil, fmt.Errorf("encode source_links: %w", err)
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE provider_definitions
		    SET display_name = $2, provider_type = $3, regions = $4, business_domains = $5,
		        owner = $6, source_links = $7::jsonb, notes = $8, updated_at = now()
		  WHERE id = $1
		 RETURNING `+providerDefinitionColumns,
		d.ID, d.DisplayName, d.ProviderType, orEmptySlice(d.Regions), orEmptySlice(d.BusinessDomains),
		d.Owner, string(sourceLinks), d.Notes)
	def, err := scanProviderDefinition(row)
	if err != nil {
		return nil, fmt.Errorf("update provider definition: %w", err)
	}
	return def, nil
}

// SetLatestPublishedVersion repoints (or clears, when versionID is nil) the
// definition's currently-available version.
func (r *ProviderDefinitionRepository) SetLatestPublishedVersion(ctx context.Context, q Querier, defID uuid.UUID, versionID *uuid.UUID) error {
	_, err := q.Exec(ctx,
		`UPDATE provider_definitions SET latest_published_version_id = $2, updated_at = now() WHERE id = $1`,
		defID, versionID)
	if err != nil {
		return fmt.Errorf("set latest published version: %w", err)
	}
	return nil
}

func orEmptySlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
