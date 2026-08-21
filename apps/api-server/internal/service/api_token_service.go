package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

var (
	// ErrAPITokenOwnerRequired is returned when a non-owner tries to manage API tokens.
	ErrAPITokenOwnerRequired = errors.New("only an organization owner can manage API tokens")
	// ErrAPITokenInvalid is returned when a bearer token does not match an active API token.
	ErrAPITokenInvalid = errors.New("invalid API token")
	// ErrAPITokenNotFound is returned when revoke targets a missing token.
	ErrAPITokenNotFound = errors.New("API token not found")
)

// APITokenStore persists hashed owner API tokens.
type APITokenStore interface {
	Insert(ctx context.Context, tenantID uuid.UUID, token *model.APIToken) error
	ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]model.APIToken, error)
	Revoke(ctx context.Context, tenantID, userID, id uuid.UUID) (bool, error)
	FindActiveByHash(ctx context.Context, tokenHash string) (*model.APIToken, error)
	TouchLastUsed(ctx context.Context, tenantID, id uuid.UUID) error
}

// APITokenUserLookup loads the user an API token authenticates as.
type APITokenUserLookup interface {
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.User, error)
}

// APITokenRoleLookup resolves custom-role permissions for token auth.
type APITokenRoleLookup interface {
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*model.Role, error)
}

// APITokenService issues, lists, revokes, and authenticates hashed API tokens.
type APITokenService struct {
	store APITokenStore
	users APITokenUserLookup
	roles APITokenRoleLookup
}

// NewAPITokenService constructs an APITokenService.
func NewAPITokenService(store APITokenStore, users APITokenUserLookup, roles APITokenRoleLookup) *APITokenService {
	return &APITokenService{store: store, users: users, roles: roles}
}

// Create mints a raw token, stores only its hash, and returns the secret once.
func (s *APITokenService) Create(ctx context.Context, tenantID, actorID uuid.UUID, actorRole string, req model.CreateAPITokenRequest, _ uuid.UUID, _ string) (*model.CreatedAPIToken, error) {
	if err := requireOwner(actorRole); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, NewValidationError(err)
	}

	raw, err := generateAPIToken()
	if err != nil {
		return nil, fmt.Errorf("generate API token: %w", err)
	}

	token := &model.APIToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    actorID,
		Name:      strings.TrimSpace(req.Name),
		TokenHash: model.HashAPIToken(raw),
	}
	if err := s.store.Insert(ctx, tenantID, token); err != nil {
		return nil, fmt.Errorf("store API token: %w", err)
	}

	return &model.CreatedAPIToken{APIToken: *token, Token: raw}, nil
}

// List returns the caller's active tokens without hashes or secrets.
func (s *APITokenService) List(ctx context.Context, tenantID, actorID uuid.UUID, actorRole string) ([]model.APIToken, error) {
	if err := requireOwner(actorRole); err != nil {
		return nil, err
	}
	tokens, err := s.store.ListByUser(ctx, tenantID, actorID)
	if err != nil {
		return nil, fmt.Errorf("list API tokens: %w", err)
	}
	for i := range tokens {
		tokens[i].TokenHash = ""
	}
	return tokens, nil
}

// Revoke marks a token unusable. Subsequent Bearer auth fails.
func (s *APITokenService) Revoke(ctx context.Context, tenantID, actorID uuid.UUID, actorRole string, id uuid.UUID, _ uuid.UUID, _ string) error {
	if err := requireOwner(actorRole); err != nil {
		return err
	}
	ok, err := s.store.Revoke(ctx, tenantID, actorID, id)
	if err != nil {
		return fmt.Errorf("revoke API token: %w", err)
	}
	if !ok {
		return ErrAPITokenNotFound
	}
	return nil
}

// AuthenticateAPIToken resolves a raw Bearer token to the same claims a session JWT carries.
func (s *APITokenService) AuthenticateAPIToken(ctx context.Context, rawToken string) (*model.AuthClaims, error) {
	if s == nil || s.store == nil {
		return nil, ErrAPITokenInvalid
	}
	record, err := s.store.FindActiveByHash(ctx, model.HashAPIToken(rawToken))
	if err != nil {
		return nil, fmt.Errorf("lookup API token: %w", err)
	}
	if record == nil {
		return nil, ErrAPITokenInvalid
	}

	var user *model.User
	if s.users != nil {
		user, err = s.users.FindByID(ctx, record.TenantID, record.UserID)
		if err != nil {
			return nil, fmt.Errorf("load API token user: %w", err)
		}
	}
	if user == nil {
		return nil, ErrAPITokenInvalid
	}

	permissions := model.SystemPermissionsForRole(user.Role)
	if user.RoleID != nil && s.roles != nil {
		role, roleErr := s.roles.FindByID(ctx, user.TenantID, *user.RoleID)
		if roleErr != nil {
			return nil, fmt.Errorf("load API token role: %w", roleErr)
		}
		if role != nil && role.Permissions != nil {
			permissions = append([]string(nil), role.Permissions...)
		} else {
			permissions = []string{}
		}
	}

	_ = s.store.TouchLastUsed(ctx, record.TenantID, record.ID)

	roleID := uuid.Nil
	if user.RoleID != nil {
		roleID = *user.RoleID
	}

	return &model.AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: user.ID.String()},
		TenantID:         user.TenantID,
		Email:            user.Email,
		Role:             user.Role,
		RoleID:           roleID,
		Permissions:      permissions,
		Type:             "access",
	}, nil
}

func requireOwner(role string) error {
	if role != "owner" {
		return ErrAPITokenOwnerRequired
	}
	return nil
}

func generateAPIToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return model.APITokenPrefix + hex.EncodeToString(raw), nil
}
