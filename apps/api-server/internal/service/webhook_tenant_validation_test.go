package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
	"github.com/stretchr/testify/assert"
)

// fakeTenantRepo satisfies repository.TenantRepo via an embedded (nil) interface
// and overrides only FindByID, which is all ensureTenantExists uses.
type fakeTenantRepo struct {
	repository.TenantRepo
	tenant *model.Tenant
	err    error
}

func (f *fakeTenantRepo) FindByID(_ context.Context, _ pgx.Tx, _ uuid.UUID) (*model.Tenant, error) {
	return f.tenant, f.err
}

func TestEnsureTenantExists_UnknownTenantRejected(t *testing.T) {
	s := &WebhookService{tenantRepo: &fakeTenantRepo{tenant: nil}}

	err := s.ensureTenantExists(context.Background(), nil, uuid.New())

	assert.ErrorIs(t, err, ErrUnknownTenant,
		"a webhook for a non-existent tenant_id must be rejected")
}

func TestEnsureTenantExists_KnownTenantAccepted(t *testing.T) {
	s := &WebhookService{tenantRepo: &fakeTenantRepo{tenant: &model.Tenant{ID: uuid.New()}}}

	err := s.ensureTenantExists(context.Background(), nil, uuid.New())

	assert.NoError(t, err)
}

func TestEnsureTenantExists_PropagatesRepoError(t *testing.T) {
	repoErr := errors.New("db down")
	s := &WebhookService{tenantRepo: &fakeTenantRepo{err: repoErr}}

	err := s.ensureTenantExists(context.Background(), nil, uuid.New())

	assert.ErrorIs(t, err, repoErr, "repo errors must propagate (fail closed, not treated as unknown tenant)")
}
