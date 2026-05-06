package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// TenantRepository handles persistence for tenant records and settings.
type TenantRepository struct {
	pool          *pgxpool.Pool
	encryptionKey []byte
}

// NewTenantRepository creates a new TenantRepository backed by the given pool.
func NewTenantRepository(pool *pgxpool.Pool, encryptionKey ...[]byte) *TenantRepository {
	var key []byte
	if len(encryptionKey) > 0 && len(encryptionKey[0]) > 0 {
		key = append([]byte(nil), encryptionKey[0]...)
	}
	return &TenantRepository{pool: pool, encryptionKey: key}
}

// FindBySlug uses SECURITY DEFINER function (bypasses RLS).
func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.pool.QueryRow(ctx,
		"SELECT id, name, slug, plan, settings, created_at, updated_at FROM find_tenant_by_slug($1)", slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find tenant by slug: %w", err)
	}
	if err := r.decryptTenantSettings(&t); err != nil {
		return nil, fmt.Errorf("find tenant by slug: %w", err)
	}
	return &t, nil
}

// FindByID finds a tenant by ID within a WithTenant transaction.
func (r *TenantRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	err := tx.QueryRow(ctx,
		`SELECT id, name, slug, plan, settings, created_at, updated_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find tenant by id: %w", err)
	}
	if err := r.decryptTenantSettings(&t); err != nil {
		return nil, fmt.Errorf("find tenant by id: %w", err)
	}
	return &t, nil
}

// SlugExists checks if a tenant slug is already taken.
func (r *TenantRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	t, err := r.FindBySlug(ctx, slug)
	if err != nil {
		return false, err
	}
	return t != nil, nil
}

// Create inserts a new tenant within a WithTenant transaction.
func (r *TenantRepository) Create(ctx context.Context, tx pgx.Tx, tenant *model.Tenant) error {
	return tx.QueryRow(ctx,
		`INSERT INTO tenants (id, name, slug, plan)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Plan,
	).Scan(&tenant.CreatedAt, &tenant.UpdatedAt)
}

// GetSettings returns the application-readable JSON settings blob for a tenant.
func (r *TenantRepository) GetSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID) (json.RawMessage, error) {
	var settings json.RawMessage
	err := tx.QueryRow(ctx, "SELECT settings FROM tenants WHERE id = $1", id).Scan(&settings)
	if err != nil {
		return nil, fmt.Errorf("get tenant settings: %w", err)
	}
	return r.decryptSettings(settings)
}

// GetSettingsForUpdate returns application-readable settings with a row-level lock
// (FOR UPDATE) to prevent concurrent read-modify-write races on the settings JSONB blob.
func (r *TenantRepository) GetSettingsForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (json.RawMessage, error) {
	var settings json.RawMessage
	err := tx.QueryRow(ctx, "SELECT settings FROM tenants WHERE id = $1 FOR UPDATE", id).Scan(&settings)
	if err != nil {
		return nil, fmt.Errorf("get tenant settings for update: %w", err)
	}
	return r.decryptSettings(settings)
}

// ListAllTenantIDs returns all tenant IDs (cross-tenant, runs on pool directly).
func (r *TenantRepository) ListAllTenantIDs(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, "SELECT id FROM tenants")
	if err != nil {
		return nil, fmt.Errorf("list all tenant ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan tenant id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// UpdateSettings persists updated settings for a tenant.
func (r *TenantRepository) UpdateSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID, settings json.RawMessage) error {
	encryptedSettings, err := r.encryptSettings(settings)
	if err != nil {
		return fmt.Errorf("encrypt tenant settings: %w", err)
	}

	ct, err := tx.Exec(ctx, "UPDATE tenants SET settings = $1::jsonb, updated_at = NOW() WHERE id = $2", encryptedSettings, id)
	if err != nil {
		return fmt.Errorf("update tenant settings: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

// BackfillSettingsSecretEncryption encrypts legacy plaintext secrets in tenants.settings.
// It must run with a pool that can read and update tenants across tenant boundaries.
func (r *TenantRepository) BackfillSettingsSecretEncryption(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	if len(r.encryptionKey) == 0 {
		return 0, nil
	}
	if pool == nil {
		pool = r.pool
	}

	ids, err := r.ListAllTenantIDs(ctx, pool)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, id := range ids {
		changed, err := r.backfillTenantSettingsSecretEncryption(ctx, pool, id)
		if err != nil {
			return updated, err
		}
		if changed {
			updated++
		}
	}
	return updated, nil
}

func (r *TenantRepository) backfillTenantSettingsSecretEncryption(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tenant settings backfill tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var settings json.RawMessage
	err = tx.QueryRow(ctx, "SELECT settings FROM tenants WHERE id = $1 FOR UPDATE", id).Scan(&settings)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("load tenant settings for backfill: %w", err)
	}

	encrypted, changed, err := crypto.EncryptTenantSettingsSecrets(settings, r.encryptionKey)
	if err != nil {
		return false, fmt.Errorf("encrypt tenant settings for backfill: %w", err)
	}
	if !changed {
		return false, tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, "UPDATE tenants SET settings = $1::jsonb, updated_at = NOW() WHERE id = $2", encrypted, id); err != nil {
		return false, fmt.Errorf("persist encrypted tenant settings: %w", err)
	}
	return true, tx.Commit(ctx)
}

func (r *TenantRepository) encryptSettings(settings json.RawMessage) (json.RawMessage, error) {
	encrypted, _, err := crypto.EncryptTenantSettingsSecrets(settings, r.encryptionKey)
	return encrypted, err
}

func (r *TenantRepository) decryptSettings(settings json.RawMessage) (json.RawMessage, error) {
	decrypted, _, err := crypto.DecryptTenantSettingsSecrets(settings, r.encryptionKey)
	return decrypted, err
}

func (r *TenantRepository) decryptTenantSettings(tenant *model.Tenant) error {
	settings, err := r.decryptSettings(tenant.Settings)
	if err != nil {
		return err
	}
	tenant.Settings = settings
	return nil
}
