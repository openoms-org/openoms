package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/database"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// Invitation sentinel errors.
var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationUsed     = errors.New("invitation has already been used")
)

// InvitationService manages invitation lifecycle.
type InvitationService struct {
	invRepo   repository.InvitationRepo
	auditRepo repository.AuditRepo
	pool      *pgxpool.Pool
}

// NewInvitationService creates a new InvitationService.
func NewInvitationService(invRepo repository.InvitationRepo, auditRepo repository.AuditRepo, pool *pgxpool.Pool) *InvitationService {
	return &InvitationService{
		invRepo:   invRepo,
		auditRepo: auditRepo,
		pool:      pool,
	}
}

// Create generates a new invitation token and stores it.
func (s *InvitationService) Create(ctx context.Context, tenantID, createdBy uuid.UUID, email, role string) (*model.InvitationResponse, error) {
	if email == "" {
		return nil, NewValidationError(errors.New("email is required"))
	}
	switch role {
	case "owner", "admin", "member":
	default:
		return nil, NewValidationError(errors.New("role must be one of: owner, admin, member"))
	}

	token, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	inv := &model.Invitation{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Email:     email,
		Token:     token,
		Role:      role,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedBy: &createdBy,
	}

	err = database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.invRepo.Create(ctx, tx, inv); err != nil {
			return fmt.Errorf("create invitation: %w", err)
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     createdBy,
			Action:     "invitation.created",
			EntityType: "invitation",
			EntityID:   inv.ID,
		})
	})
	if err != nil {
		return nil, err
	}

	return &model.InvitationResponse{
		Invitation:  *inv,
		InviteToken: token,
	}, nil
}

// List returns paginated invitations for a tenant.
func (s *InvitationService) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.Invitation, int, error) {
	var invitations []model.Invitation
	var total int
	err := database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var listErr error
		invitations, total, listErr = s.invRepo.List(ctx, tx, limit, offset)
		return listErr
	})
	return invitations, total, err
}

// Delete removes an invitation.
func (s *InvitationService) Delete(ctx context.Context, tenantID, id, deletedBy uuid.UUID) error {
	return database.WithTenant(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.invRepo.Delete(ctx, tx, id); err != nil {
			return err
		}
		return s.auditRepo.Log(ctx, tx, model.AuditEntry{
			TenantID:   tenantID,
			UserID:     deletedBy,
			Action:     "invitation.deleted",
			EntityType: "invitation",
			EntityID:   id,
		})
	})
}

// ValidateToken checks an invitation token and returns tenant info if valid.
// Uses SECURITY DEFINER function (no RLS context needed).
func (s *InvitationService) ValidateToken(ctx context.Context, token string) (*model.Invitation, error) {
	inv, err := s.invRepo.FindByToken(ctx, s.pool, token)
	if err != nil {
		return nil, fmt.Errorf("find invitation: %w", err)
	}
	if inv == nil {
		return nil, ErrInvitationNotFound
	}
	if inv.UsedAt != nil {
		return nil, ErrInvitationUsed
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}
	return inv, nil
}

// MarkUsed marks an invitation as used after successful registration.
func (s *InvitationService) MarkUsed(ctx context.Context, token string) error {
	return s.invRepo.MarkUsed(ctx, s.pool, token)
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
