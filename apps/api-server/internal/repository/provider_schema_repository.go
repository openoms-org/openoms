package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// ProviderSchemaRepository persists the per-version credential/settings field
// schema. Platform-managed, no RLS; queries run directly on the pool.
type ProviderSchemaRepository struct {
	pool *pgxpool.Pool
}

// NewProviderSchemaRepository creates the repository backed by the pool.
func NewProviderSchemaRepository(pool *pgxpool.Pool) *ProviderSchemaRepository {
	return &ProviderSchemaRepository{pool: pool}
}

// schemaPayload is the jsonb shape stored in provider_field_schemas.schema.
type schemaPayload struct {
	Groups []model.ProviderFieldGroup `json:"groups"`
}

func decodeSchemaRow(id, versionID uuid.UUID, raw []byte, s *model.ProviderFieldSchema) error {
	s.ID = id
	s.ProviderVersionID = versionID
	s.Groups = []model.ProviderFieldGroup{}
	if len(raw) > 0 {
		var p schemaPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("decode field schema: %w", err)
		}
		if p.Groups != nil {
			s.Groups = p.Groups
		}
	}
	return nil
}

// Upsert stores (creating or replacing) the field schema for a version.
func (r *ProviderSchemaRepository) Upsert(ctx context.Context, versionID uuid.UUID, groups []model.ProviderFieldGroup) (*model.ProviderFieldSchema, error) {
	if groups == nil {
		groups = []model.ProviderFieldGroup{}
	}
	payload, err := json.Marshal(schemaPayload{Groups: groups})
	if err != nil {
		return nil, fmt.Errorf("encode field schema: %w", err)
	}
	var s model.ProviderFieldSchema
	var raw []byte
	err = r.pool.QueryRow(ctx,
		`INSERT INTO provider_field_schemas (provider_version_id, schema)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (provider_version_id)
		 DO UPDATE SET schema = EXCLUDED.schema, updated_at = now()
		 RETURNING id, provider_version_id, schema, created_at, updated_at`,
		versionID, string(payload)).
		Scan(&s.ID, &s.ProviderVersionID, &raw, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert field schema: %w", err)
	}
	if err := decodeSchemaRow(s.ID, s.ProviderVersionID, raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByVersion returns the field schema for a version, or pgx.ErrNoRows if none.
func (r *ProviderSchemaRepository) GetByVersion(ctx context.Context, versionID uuid.UUID) (*model.ProviderFieldSchema, error) {
	var s model.ProviderFieldSchema
	var raw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, provider_version_id, schema, created_at, updated_at
		   FROM provider_field_schemas WHERE provider_version_id = $1`, versionID).
		Scan(&s.ID, &s.ProviderVersionID, &raw, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := decodeSchemaRow(s.ID, s.ProviderVersionID, raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
