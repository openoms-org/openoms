package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
)

// OrderRepo defines the interface for order persistence operations.
type OrderRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.OrderListFilter) ([]model.Order, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Order, error)
	Create(ctx context.Context, tx pgx.Tx, order *model.Order) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateOrderRequest) error
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, shippedAt, deliveredAt *time.Time) error
	FindByExternalID(ctx context.Context, tx pgx.Tx, source, externalID string) (*model.Order, error)
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// UserRepo defines the interface for user persistence operations.
type UserRepo interface {
	FindForAuth(ctx context.Context, email string, tenantID uuid.UUID) (*UserWithPassword, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.User, error)
	List(ctx context.Context, tx pgx.Tx) ([]model.User, error)
	Create(ctx context.Context, tx pgx.Tx, user *model.User, passwordHash string) error
	UpdateRole(ctx context.Context, tx pgx.Tx, id uuid.UUID, role string) error
	UpdateName(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string) error
	UpdateLastLogin(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	CountByRole(ctx context.Context, tx pgx.Tx, role string) (int, error)
}

// TenantRepo defines the interface for tenant persistence operations.
type TenantRepo interface {
	FindBySlug(ctx context.Context, slug string) (*model.Tenant, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Tenant, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	Create(ctx context.Context, tx pgx.Tx, tenant *model.Tenant) error
	GetSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID) (json.RawMessage, error)
	ListAllTenantIDs(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error)
	UpdateSettings(ctx context.Context, tx pgx.Tx, id uuid.UUID, settings json.RawMessage) error
}

// AuditRepo defines the interface for audit log persistence operations.
type AuditRepo interface {
	Log(ctx context.Context, tx pgx.Tx, entry model.AuditEntry) error
	ListByEntity(ctx context.Context, tx pgx.Tx, entityType string, entityID uuid.UUID) ([]model.AuditLogEntry, error)
	List(ctx context.Context, tx pgx.Tx, filter model.AuditListFilter) ([]model.AuditLogEntry, int, error)
}

// ShipmentRepo defines the interface for shipment persistence operations.
type ShipmentRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.ShipmentListFilter) ([]model.Shipment, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Shipment, error)
	Create(ctx context.Context, tx pgx.Tx, shipment *model.Shipment) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateShipmentRequest) error
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// ProductRepo defines the interface for product persistence operations.
type ProductRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.ProductListFilter) ([]model.Product, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Product, error)
	Create(ctx context.Context, tx pgx.Tx, product *model.Product) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateProductRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// IntegrationRepo defines the interface for integration persistence operations.
type IntegrationRepo interface {
	List(ctx context.Context, tx pgx.Tx) ([]model.IntegrationWithCreds, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.IntegrationWithCreds, error)
	FindByProvider(ctx context.Context, tx pgx.Tx, provider string) (*model.IntegrationWithCreds, error)
	Create(ctx context.Context, tx pgx.Tx, integration *model.Integration, encryptedCreds string) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateIntegrationRequest, encryptedCreds *string) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// ReturnRepo defines the interface for return/RMA persistence operations.
type ReturnRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.ReturnListFilter) ([]model.Return, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Return, error)
	Create(ctx context.Context, tx pgx.Tx, ret *model.Return) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateReturnRequest) error
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// WebhookRepo defines the interface for webhook event persistence operations.
type WebhookRepo interface {
	Create(ctx context.Context, tx pgx.Tx, event *model.WebhookEvent) error
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.WebhookEvent, error)
}

// WebhookDeliveryRepo defines the interface for webhook delivery persistence operations.
type WebhookDeliveryRepo interface {
	Create(ctx context.Context, tx pgx.Tx, delivery *model.WebhookDelivery) error
	List(ctx context.Context, tx pgx.Tx, filter model.WebhookDeliveryFilter) ([]model.WebhookDelivery, int, error)
}

// StatsRepo defines the interface for statistics/analytics persistence operations.
type StatsRepo interface {
	GetOrderCountByStatus(ctx context.Context, tx pgx.Tx) (map[string]int, error)
	GetOrderCountBySource(ctx context.Context, tx pgx.Tx) (map[string]int, error)
	GetTotalRevenue(ctx context.Context, tx pgx.Tx) (float64, error)
	GetDailyRevenue(ctx context.Context, tx pgx.Tx, days int) ([]model.DailyRevenue, error)
	GetRecentOrders(ctx context.Context, tx pgx.Tx, limit int) ([]model.OrderSummary, error)
}

// ProductListingRepo defines the interface for product listing persistence operations.
type ProductListingRepo interface {
	Create(ctx context.Context, tx pgx.Tx, listing *model.ProductListing) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req *model.UpdateProductListingRequest) error
	GetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductListing, error)
	FindByProductAndIntegration(ctx context.Context, tx pgx.Tx, productID, integrationID uuid.UUID) (*model.ProductListing, error)
	ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]*model.ProductListing, error)
	ListByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID) ([]*model.ProductListing, error)
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// SyncJobRepo defines the interface for sync job persistence operations.
type SyncJobRepo interface {
	Create(ctx context.Context, tx pgx.Tx, job *model.SyncJob) error
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status string, itemsProcessed, itemsFailed int, errorMsg *string) error
	GetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.SyncJob, error)
	ListByIntegration(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID, limit int) ([]*model.SyncJob, error)
}

// SupplierRepo defines the interface for supplier persistence operations.
type SupplierRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.SupplierListFilter) ([]model.Supplier, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Supplier, error)
	Create(ctx context.Context, tx pgx.Tx, supplier *model.Supplier) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateSupplierRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	UpdateSyncStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, lastSyncAt time.Time, errorMessage *string) error
}

// SupplierProductRepo defines the interface for supplier product persistence operations.
type SupplierProductRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.SupplierProductListFilter) ([]model.SupplierProduct, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.SupplierProduct, error)
	Create(ctx context.Context, tx pgx.Tx, sp *model.SupplierProduct) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string, ean, sku *string, price *float64, stock int, metadata []byte, syncedAt *time.Time) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	FindByEAN(ctx context.Context, tx pgx.Tx, ean string) (*model.SupplierProduct, error)
	FindBySupplierAndExternalID(ctx context.Context, tx pgx.Tx, supplierID uuid.UUID, externalID string) (*model.SupplierProduct, error)
	UpsertByExternalID(ctx context.Context, tx pgx.Tx, sp *model.SupplierProduct) error
	LinkToProduct(ctx context.Context, tx pgx.Tx, id uuid.UUID, productID uuid.UUID) error
}
