package testutil

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"

	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/repository"
)

// --- MockOrderRepo ---

type MockOrderRepo struct{ mock.Mock }

func (m *MockOrderRepo) List(ctx context.Context, tx pgx.Tx, filter model.OrderListFilter) ([]model.Order, int, error) {
	args := m.Called(ctx, tx, filter)
	return args.Get(0).([]model.Order), args.Int(1), args.Error(2)
}

func (m *MockOrderRepo) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderRepo) Create(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	return m.Called(ctx, tx, order).Error(0)
}

func (m *MockOrderRepo) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateOrderRequest) error {
	return m.Called(ctx, tx, id, req).Error(0)
}

func (m *MockOrderRepo) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, shippedAt, deliveredAt *time.Time) error {
	return m.Called(ctx, tx, id, status, shippedAt, deliveredAt).Error(0)
}

func (m *MockOrderRepo) FindByExternalID(ctx context.Context, tx pgx.Tx, source, externalID string) (*model.Order, error) {
	args := m.Called(ctx, tx, source, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *MockOrderRepo) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return m.Called(ctx, tx, id).Error(0)
}

// --- MockUserRepo ---

type MockUserRepo struct{ mock.Mock }

func (m *MockUserRepo) FindForAuth(ctx context.Context, email string, tenantID uuid.UUID) (*repository.UserWithPassword, error) {
	args := m.Called(ctx, email, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.UserWithPassword), args.Error(1)
}

func (m *MockUserRepo) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepo) List(ctx context.Context, tx pgx.Tx) ([]model.User, error) {
	args := m.Called(ctx, tx)
	return args.Get(0).([]model.User), args.Error(1)
}

func (m *MockUserRepo) Create(ctx context.Context, tx pgx.Tx, user *model.User, passwordHash string) error {
	return m.Called(ctx, tx, user, passwordHash).Error(0)
}

func (m *MockUserRepo) UpdateRole(ctx context.Context, tx pgx.Tx, id uuid.UUID, role string) error {
	return m.Called(ctx, tx, id, role).Error(0)
}

func (m *MockUserRepo) UpdateName(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string) error {
	return m.Called(ctx, tx, id, name).Error(0)
}

func (m *MockUserRepo) UpdateLastLogin(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return m.Called(ctx, tx, id).Error(0)
}

func (m *MockUserRepo) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return m.Called(ctx, tx, id).Error(0)
}

func (m *MockUserRepo) CountByRole(ctx context.Context, tx pgx.Tx, role string) (int, error) {
	args := m.Called(ctx, tx, role)
	return args.Int(0), args.Error(1)
}

// --- MockTenantRepo ---

type MockTenantRepo struct{ mock.Mock }

func (m *MockTenantRepo) FindBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Tenant), args.Error(1)
}

func (m *MockTenantRepo) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Tenant, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Tenant), args.Error(1)
}

func (m *MockTenantRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

func (m *MockTenantRepo) Create(ctx context.Context, tx pgx.Tx, tenant *model.Tenant) error {
	return m.Called(ctx, tx, tenant).Error(0)
}

func (m *MockTenantRepo) GetSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID) (json.RawMessage, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(json.RawMessage), args.Error(1)
}

func (m *MockTenantRepo) ListAllTenantIDs(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	args := m.Called(ctx, pool)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockTenantRepo) UpdateSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID, settings json.RawMessage) error {
	return m.Called(ctx, tx, id, settings).Error(0)
}

// --- MockAuditRepo ---

type MockAuditRepo struct{ mock.Mock }

func (m *MockAuditRepo) Log(ctx context.Context, tx pgx.Tx, entry model.AuditEntry) error {
	return m.Called(ctx, tx, entry).Error(0)
}

func (m *MockAuditRepo) ListByEntity(ctx context.Context, tx pgx.Tx, entityType string, entityID uuid.UUID) ([]model.AuditLogEntry, error) {
	args := m.Called(ctx, tx, entityType, entityID)
	return args.Get(0).([]model.AuditLogEntry), args.Error(1)
}

func (m *MockAuditRepo) List(ctx context.Context, tx pgx.Tx, filter model.AuditListFilter) ([]model.AuditLogEntry, int, error) {
	args := m.Called(ctx, tx, filter)
	return args.Get(0).([]model.AuditLogEntry), args.Int(1), args.Error(2)
}

// --- MockShipmentRepo ---

type MockShipmentRepo struct{ mock.Mock }

func (m *MockShipmentRepo) List(ctx context.Context, tx pgx.Tx, filter model.ShipmentListFilter) ([]model.Shipment, int, error) {
	args := m.Called(ctx, tx, filter)
	return args.Get(0).([]model.Shipment), args.Int(1), args.Error(2)
}

func (m *MockShipmentRepo) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Shipment, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Shipment), args.Error(1)
}

func (m *MockShipmentRepo) Create(ctx context.Context, tx pgx.Tx, shipment *model.Shipment) error {
	return m.Called(ctx, tx, shipment).Error(0)
}

func (m *MockShipmentRepo) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateShipmentRequest) error {
	return m.Called(ctx, tx, id, req).Error(0)
}

func (m *MockShipmentRepo) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error {
	return m.Called(ctx, tx, id, status).Error(0)
}

func (m *MockShipmentRepo) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return m.Called(ctx, tx, id).Error(0)
}

// --- MockProductRepo ---

type MockProductRepo struct{ mock.Mock }

func (m *MockProductRepo) List(ctx context.Context, tx pgx.Tx, filter model.ProductListFilter) ([]model.Product, int, error) {
	args := m.Called(ctx, tx, filter)
	return args.Get(0).([]model.Product), args.Int(1), args.Error(2)
}

func (m *MockProductRepo) FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Product, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductRepo) Create(ctx context.Context, tx pgx.Tx, product *model.Product) error {
	return m.Called(ctx, tx, product).Error(0)
}

func (m *MockProductRepo) Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateProductRequest) error {
	return m.Called(ctx, tx, id, req).Error(0)
}

func (m *MockProductRepo) Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	return m.Called(ctx, tx, id).Error(0)
}

// --- MockWebhookDeliveryRepo ---

type MockWebhookDeliveryRepo struct{ mock.Mock }

func (m *MockWebhookDeliveryRepo) Create(ctx context.Context, tx pgx.Tx, delivery *model.WebhookDelivery) error {
	return m.Called(ctx, tx, delivery).Error(0)
}

func (m *MockWebhookDeliveryRepo) List(ctx context.Context, tx pgx.Tx, filter model.WebhookDeliveryFilter) ([]model.WebhookDelivery, int, error) {
	args := m.Called(ctx, tx, filter)
	return args.Get(0).([]model.WebhookDelivery), args.Int(1), args.Error(2)
}
