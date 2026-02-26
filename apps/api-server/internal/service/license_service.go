package service

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

var (
	// ErrLicenseTokenExpired is returned when the license token has expired.
	ErrLicenseTokenExpired = errors.New("license token has expired")
	// ErrLicenseTokenUsed is returned when the license token has already been consumed.
	ErrLicenseTokenUsed = errors.New("license token has already been used")
	// ErrLicenseTokenInvalid is returned when the license token fails validation.
	ErrLicenseTokenInvalid = errors.New("invalid license token")
)

// LicenseService verifies Ed25519-signed license JWT tokens.
type LicenseService struct {
	publicKey   ed25519.PublicKey
	licenseRepo repository.LicenseRepo
	pool        *pgxpool.Pool
}

// NewLicenseService creates a new LicenseService with the given public key.
// Pass nil publicKey to disable license verification (community/self-hosted mode).
func NewLicenseService(publicKey ed25519.PublicKey, pool *pgxpool.Pool) *LicenseService {
	return &LicenseService{
		publicKey: publicKey,
		pool:      pool,
	}
}

// SetRepository sets the license repository for token usage tracking.
func (s *LicenseService) SetRepository(repo repository.LicenseRepo) {
	s.licenseRepo = repo
}

// IsDisabled returns true when no public key is configured (community mode).
func (s *LicenseService) IsDisabled() bool {
	return s.publicKey == nil
}

// ParseAndVerify parses a license token string, verifies its Ed25519 signature,
// and validates claims (issuer, subject, plan, email, JTI).
func (s *LicenseService) ParseAndVerify(tokenStr string) (*model.LicenseClaims, error) {
	if s.IsDisabled() {
		return nil, fmt.Errorf("%w: license system not configured", ErrLicenseTokenInvalid)
	}

	claims := &model.LicenseClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrLicenseTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrLicenseTokenInvalid, err)
	}
	if !token.Valid {
		return nil, ErrLicenseTokenInvalid
	}

	issuer, _ := claims.GetIssuer()
	subject, _ := claims.GetSubject()
	if issuer != model.LicenseIssuer || subject != model.LicenseSubject {
		return nil, fmt.Errorf("%w: unexpected issuer or subject", ErrLicenseTokenInvalid)
	}

	if err := claims.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLicenseTokenInvalid, err)
	}

	return claims, nil
}

// VerifyToken performs full verification of a license token, including
// signature validation, claims checks, and database lookup for replay protection.
func (s *LicenseService) VerifyToken(ctx context.Context, tokenStr string) (*model.LicenseClaims, error) {
	claims, err := s.ParseAndVerify(tokenStr)
	if err != nil {
		return nil, err
	}

	if s.licenseRepo != nil && s.pool != nil {
		used, err := s.licenseRepo.IsTokenUsed(ctx, s.pool, claims.JTI)
		if err != nil {
			return nil, fmt.Errorf("check license token: %w", err)
		}
		if used {
			return nil, ErrLicenseTokenUsed
		}
	}

	return claims, nil
}

// MarkUsed records a license token JTI as consumed after successful registration.
func (s *LicenseService) MarkUsed(ctx context.Context, jti, tenantID uuid.UUID, email, plan string) error {
	if s.licenseRepo == nil || s.pool == nil {
		return nil
	}
	return s.licenseRepo.MarkTokenUsed(ctx, s.pool, jti, tenantID, email, plan)
}
