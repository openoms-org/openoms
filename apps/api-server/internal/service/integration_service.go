package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/crypto"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	ErrIntegrationNotFound = errors.New("integration not found")
	ErrDuplicateProvider   = errors.New("integration for this provider already exists in this tenant")
)

type IntegrationService struct {
	integrationRepo repository.IntegrationRepo
	auditRepo       repository.AuditRepo
	pool            *pgxpool.Pool
	encryptionKey   []byte
}

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

func (s *IntegrationService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateIntegrationRequest, actorID uuid.UUID, ip string) (*model.Integration, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
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

func (s *IntegrationService) Update(ctx context.Context, tenantID, integrationID uuid.UUID, req model.UpdateIntegrationRequest, actorID uuid.UUID, ip string) (*model.Integration, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	var encryptedCreds *string
	if req.Credentials != nil {
		encrypted, err := crypto.Encrypt(*req.Credentials, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
		encryptedCreds = &encrypted
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
