package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

type IntegrationRepository struct{}

func NewIntegrationRepository() *IntegrationRepository {
	return &IntegrationRepository{}
}

func (r *IntegrationRepository) List(ctx context.Context, tx pgx.Tx) ([]model.IntegrationWithCreds, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, provider, label, status, credentials, settings, sync_cursor, error_message, last_sync_at, created_at, updated_at
		 FROM integrations ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()

	var integrations []model.IntegrationWithCreds
	for rows.Next() {
		var i model.IntegrationWithCreds
		var credsJSON json.RawMessage
		if err := rows.Scan(&i.ID, &i.TenantID, &i.Provider, &i.Label, &i.Status,
			&credsJSON, &i.Settings, &i.SyncCursor, &i.ErrorMessage, &i.LastSyncAt, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan integration: %w", err)
		}
		// Extract the encrypted string from the JSON string value
		if len(credsJSON) > 0 {
			if err := json.Unmarshal(credsJSON, &i.EncryptedCredentials); err != nil {
				slog.Warn("failed to unmarshal integration credentials", "error", err, "integration_id", i.ID)
			}
		}
		integrations = append(integrations, i)
	}
	return integrations, rows.Err()
}

func (r *IntegrationRepository) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.IntegrationWithCreds, error) {
	var i model.IntegrationWithCreds
	var credsJSON json.RawMessage
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, provider, label, status, credentials, settings, sync_cursor, error_message, last_sync_at, created_at, updated_at
		 FROM integrations WHERE id = $1`, id,
	).Scan(&i.ID, &i.TenantID, &i.Provider, &i.Label, &i.Status,
		&credsJSON, &i.Settings, &i.SyncCursor, &i.ErrorMessage, &i.LastSyncAt, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find integration by id: %w", err)
	}
	if len(credsJSON) > 0 {
		if err := json.Unmarshal(credsJSON, &i.EncryptedCredentials); err != nil {
			slog.Warn("failed to unmarshal integration credentials", "error", err, "integration_id", i.ID)
		}
	}
	return &i, nil
}

func (r *IntegrationRepository) FindByProvider(ctx context.Context, tx pgx.Tx, provider string) (*model.IntegrationWithCreds, error) {
	var i model.IntegrationWithCreds
	var credsJSON json.RawMessage
	err := tx.QueryRow(ctx,
		`SELECT id, tenant_id, provider, label, status, credentials, settings, sync_cursor, error_message, last_sync_at, created_at, updated_at
		 FROM integrations WHERE provider = $1 LIMIT 1`, provider,
	).Scan(&i.ID, &i.TenantID, &i.Provider, &i.Label, &i.Status,
		&credsJSON, &i.Settings, &i.SyncCursor, &i.ErrorMessage, &i.LastSyncAt, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find integration by provider: %w", err)
	}
	if len(credsJSON) > 0 {
		if err := json.Unmarshal(credsJSON, &i.EncryptedCredentials); err != nil {
			slog.Warn("failed to unmarshal integration credentials", "error", err, "integration_id", i.ID)
		}
	}
	return &i, nil
}

func (r *IntegrationRepository) Create(ctx context.Context, tx pgx.Tx, integration *model.Integration, encryptedCreds string) error {
	// Store encrypted credentials as a JSON string value in the JSONB column
	credsJSON, _ := json.Marshal(encryptedCreds)
	return tx.QueryRow(ctx,
		`INSERT INTO integrations (id, tenant_id, provider, label, status, credentials, settings)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		 RETURNING created_at, updated_at`,
		integration.ID, integration.TenantID, integration.Provider, integration.Label, integration.Status,
		credsJSON, integration.Settings,
	).Scan(&integration.CreatedAt, &integration.UpdatedAt)
}

func (r *IntegrationRepository) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateIntegrationRequest, encryptedCreds *string) error {
	var setClauses []string
	var args []any
	argIdx := 1

	if req.Label != nil {
		setClauses = append(setClauses, fmt.Sprintf("label = $%d", argIdx))
		args = append(args, *req.Label)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}
	if encryptedCreds != nil {
		credsJSON, _ := json.Marshal(*encryptedCreds)
		setClauses = append(setClauses, fmt.Sprintf("credentials = $%d::jsonb", argIdx))
		args = append(args, credsJSON)
		argIdx++
	}
	if req.Settings != nil {
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d", argIdx))
		args = append(args, *req.Settings)
		argIdx++
	}
	if req.SyncCursor != nil {
		setClauses = append(setClauses, fmt.Sprintf("sync_cursor = $%d", argIdx))
		args = append(args, *req.SyncCursor)
		argIdx++
	}
	if req.ErrorMessage != nil {
		setClauses = append(setClauses, fmt.Sprintf("error_message = $%d", argIdx))
		args = append(args, *req.ErrorMessage)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE integrations SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, id)

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("integration not found")
	}
	return nil
}

func (r *IntegrationRepository) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	ct, err := tx.Exec(ctx, "DELETE FROM integrations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("integration not found")
	}
	return nil
}
