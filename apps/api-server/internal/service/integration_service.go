package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// Integration service sentinel errors.
var (
	ErrIntegrationNotFound      = errors.New("integration not found")
	ErrDuplicateProvider        = errors.New("integration for this provider already exists in this tenant")
	ErrIntegrationLimitExceeded = errors.New("integration limit exceeded")
)

// IntegrationService handles business logic for marketplace and carrier integrations.
type IntegrationService struct {
	integrationRepo repository.IntegrationRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	encryptionKey   []byte
}

// NewIntegrationService creates a new IntegrationService.
func NewIntegrationService(
	integrationRepo repository.IntegrationRepo,
	auditRepo repository.AuditRepo,
	pool *pgxpool.Pool,
	encryptionKey []byte,
) *IntegrationService {
	return &IntegrationService{
		integrationRepo: integrationRepo,
		auditRepo:       auditRepo,
		pool:            pool,
		encryptionKey:   encryptionKey,
	}
}

// List returns all integrations for the given tenant.
func (s *IntegrationService) List(ctx context.Context, tenantID uuid.UUID) ([]model.Integration, error) {
	var result []model.Integration
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		withCreds, err := s.integrationRepo.List(ctx, tx)
		if err != nil {
			return err
		}
		result = make([]model.Integration, len(withCreds))
		for i, wc := range withCreds {
			wc.HasCredentials = wc.EncryptedCredentials != ""
			result[i] = wc.Integration
		}
		return nil
	})
	return result, err
}

// Get returns a single integration by ID.
func (s *IntegrationService) Get(ctx context.Context, tenantID, integrationID uuid.UUID) (*model.Integration, error) {
	var result *model.Integration
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		wc, err := s.integrationRepo.FindByID(ctx, tx, integrationID)
		if err != nil {
			return err
		}
		if wc == nil {
			return ErrIntegrationNotFound
		}
		wc.HasCredentials = wc.EncryptedCredentials != ""
		result = &wc.Integration
		return nil
	})
	return result, err
}

// GetDecryptedCredentialsByProvider returns decrypted credentials JSON for a given provider.
func (s *IntegrationService) GetDecryptedCredentialsByProvider(ctx context.Context, tenantID uuid.UUID, provider string) ([]byte, *model.Integration, error) {
	var credJSON []byte
	var result *model.Integration
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		wc, err := s.integrationRepo.FindByProvider(ctx, tx, provider)
		if err != nil {
			return err
		}
		if wc == nil {
			return ErrIntegrationNotFound
		}
		if wc.EncryptedCredentials == "" {
			return errors.New("integration has no credentials")
		}
		decrypted, err := crypto.Decrypt(wc.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypt credentials: %w", err)
		}
		credJSON = decrypted
		wc.HasCredentials = true
		result = &wc.Integration
		return nil
	})
	return credJSON, result, err
}

// GetDecryptedCredentialsByID returns decrypted credentials JSON for a given integration ID.
// Used when a supplier is linked to a specific integration via integration_id.
func (s *IntegrationService) GetDecryptedCredentialsByID(ctx context.Context, tenantID uuid.UUID, integrationID uuid.UUID) ([]byte, error) {
	var credJSON []byte
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		wc, err := s.integrationRepo.FindByID(ctx, tx, integrationID)
		if err != nil {
			return err
		}
		if wc == nil {
			return ErrIntegrationNotFound
		}
		if wc.EncryptedCredentials == "" {
			return errors.New("integration has no credentials")
		}
		decrypted, err := crypto.Decrypt(wc.EncryptedCredentials, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("decrypt credentials: %w", err)
		}
		credJSON = decrypted
		return nil
	})
	return credJSON, err
}

// Create adds a new marketplace or carrier integration for a tenant.
func (s *IntegrationService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateIntegrationRequest, actorID uuid.UUID, ip string) (*model.Integration, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	encrypted, err := crypto.Encrypt(req.Credentials, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}

	settings := req.Settings
	if settings == nil {
		settings = []byte("{}")
	}

	integration := &model.Integration{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Provider:       req.Provider,
		Label:          req.Label,
		Status:         "active",
		HasCredentials: true,
		Settings:       settings,
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Atomic plan limit check inside transaction to prevent TOCTOU race
		if req.MaxIntegrations > 0 {
			count, err := s.integrationRepo.Count(ctx, tx)
			if err != nil {
				return fmt.Errorf("count integrations for limit check: %w", err)
			}
			if count >= req.MaxIntegrations {
				return ErrIntegrationLimitExceeded
			}
		}

		if err := s.integrationRepo.Create(ctx, tx, integration, encrypted); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "integration.created",
			EntityType: "integration",
			EntityID:   integration.ID,
			Changes:    map[string]string{"provider": req.Provider},
			IPAddress:  ip,
		})
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateProvider
		}
		return nil, err
	}
	return integration, nil
}

// Update modifies an existing integration.
func (s *IntegrationService) Update(ctx context.Context, tenantID, integrationID uuid.UUID, req model.UpdateIntegrationRequest, actorID uuid.UUID, ip string) (*model.Integration, error) {
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	var result *model.Integration
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		wc, err := s.integrationRepo.FindByID(ctx, tx, integrationID)
		if err != nil {
			return err
		}
		if wc == nil {
			return ErrIntegrationNotFound
		}

		var encryptedCreds *string
		if req.Credentials != nil {
			existingCredentials := []byte("{}")
			if wc.EncryptedCredentials != "" {
				existingCredentials, err = crypto.Decrypt(wc.EncryptedCredentials, s.encryptionKey)
				if err != nil {
					return fmt.Errorf("decrypt existing credentials: %w", err)
				}
			}

			mergedCredentials, err := mergeCredentialUpdate(existingCredentials, *req.Credentials)
			if err != nil {
				return err
			}
			encrypted, err := crypto.Encrypt(mergedCredentials, s.encryptionKey)
			if err != nil {
				return fmt.Errorf("encrypt credentials: %w", err)
			}
			encryptedCreds = &encrypted
		}

		if err := s.integrationRepo.Update(ctx, tx, integrationID, req, encryptedCreds); err != nil {
			return err
		}

		// Re-fetch to get updated fields
		wc, err = s.integrationRepo.FindByID(ctx, tx, integrationID)
		if err != nil {
			return err
		}
		wc.HasCredentials = wc.EncryptedCredentials != ""
		result = &wc.Integration

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "integration.updated",
			EntityType: "integration",
			EntityID:   integrationID,
			IPAddress:  ip,
		})
	})
	return result, err
}

func mergeCredentialUpdate(existing, update []byte) ([]byte, error) {
	var existingValues map[string]any
	if len(existing) == 0 {
		existingValues = map[string]any{}
	} else if err := json.Unmarshal(existing, &existingValues); err != nil {
		return nil, fmt.Errorf("decode existing credentials: %w", err)
	}
	if existingValues == nil {
		existingValues = map[string]any{}
	}

	var updateValues map[string]any
	if err := json.Unmarshal(update, &updateValues); err != nil {
		return nil, fmt.Errorf("decode credential update: %w", err)
	}

	for key, value := range updateValues {
		// An explicit JSON null clears/removes a credential field. This is the
		// supported path to rotate a secret to empty or drop a field entirely —
		// distinct from an omitted/empty field, which preserves the stored value.
		if value == nil {
			delete(existingValues, key)
			continue
		}
		if key == "webhook_secret" {
			secret, ok := value.(string)
			if !ok || strings.TrimSpace(secret) == "" {
				// Empty (not null) webhook_secret means "unchanged": the UI does
				// not echo the stored secret back, so an empty value must not wipe
				// it. Send JSON null to explicitly clear, or a new value to rotate.
				continue
			}
		}
		existingValues[key] = value
	}

	merged, err := json.Marshal(existingValues)
	if err != nil {
		return nil, fmt.Errorf("encode merged credentials: %w", err)
	}
	return merged, nil
}

// UpdateCredentialsByProvider encrypts and persists new credential JSON for a given provider.
// Used by OAuth token refresh callbacks to persist refreshed tokens.
func (s *IntegrationService) UpdateCredentialsByProvider(ctx context.Context, tenantID uuid.UUID, provider string, credentialsJSON []byte) error {
	encrypted, err := crypto.Encrypt(credentialsJSON, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		wc, err := s.integrationRepo.FindByProvider(ctx, tx, provider)
		if err != nil {
			return err
		}
		if wc == nil {
			return ErrIntegrationNotFound
		}
		return s.integrationRepo.Update(ctx, tx, wc.ID, model.UpdateIntegrationRequest{}, &encrypted)
	})
}

// Delete removes an integration by ID.
func (s *IntegrationService) Delete(ctx context.Context, tenantID, integrationID uuid.UUID, actorID uuid.UUID, ip string) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		wc, err := s.integrationRepo.FindByID(ctx, tx, integrationID)
		if err != nil {
			return err
		}
		if wc == nil {
			return ErrIntegrationNotFound
		}

		if err := s.integrationRepo.Delete(ctx, tx, integrationID); err != nil {
			return err
		}

		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     actorID,
			Action:     "integration.deleted",
			EntityType: "integration",
			EntityID:   integrationID,
			Changes:    map[string]string{"provider": wc.Provider},
			IPAddress:  ip,
		})
	})
}
