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

// AutomationRuleRepo defines the interface for automation rule persistence.
type AutomationRuleRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.AutomationRuleListFilter) ([]model.AutomationRule, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.AutomationRule, error)
	FindByTenantAndEvent(ctx context.Context, tx pgx.Tx, event string) ([]model.AutomationRule, error)
	Create(ctx context.Context, tx pgx.Tx, rule *model.AutomationRule) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateAutomationRuleRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	IncrementFireCount(ctx context.Context, tx pgx.Tx, id uuid.UUID, firedAt time.Time) error
}

// AutomationRuleLogRepo defines the interface for automation rule log persistence.
type AutomationRuleLogRepo interface {
	Create(ctx context.Context, tx pgx.Tx, log *model.AutomationRuleLog) error
	ListByRuleID(ctx context.Context, tx pgx.Tx, ruleID uuid.UUID, limit, offset int) ([]model.AutomationRuleLog, int, error)
}

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
	FindBySKU(ctx context.Context, tx pgx.Tx, sku string) (*model.Product, error)
	FindByEAN(ctx context.Context, tx pgx.Tx, ean string) (*model.Product, error)
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
	GetMostCommonCurrency(ctx context.Context, tx pgx.Tx) (string, error)
	GetTopProducts(ctx context.Context, tx pgx.Tx, limit int) ([]model.TopProduct, error)
	GetRevenueBySource(ctx context.Context, tx pgx.Tx, days int) ([]model.SourceRevenue, error)
	GetOrderTrends(ctx context.Context, tx pgx.Tx, days int) ([]model.DailyOrderTrend, error)
	GetPaymentMethodStats(ctx context.Context, tx pgx.Tx) (map[string]int, error)
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
	List(ctx context.Context, tx pgx.Tx, filter model.SyncJobListFilter) ([]*model.SyncJob, int, error)
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

// InvoiceRepo defines the interface for invoice persistence operations.
type InvoiceRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.InvoiceListFilter) ([]model.Invoice, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Invoice, error)
	FindByOrderID(ctx context.Context, tx pgx.Tx, orderID uuid.UUID) ([]model.Invoice, error)
	Create(ctx context.Context, tx pgx.Tx, inv *model.Invoice) error
	Update(ctx context.Context, tx pgx.Tx, inv *model.Invoice) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
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

// WarehouseRepo defines the interface for warehouse persistence operations.
type WarehouseRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.WarehouseListFilter) ([]model.Warehouse, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Warehouse, error)
	Create(ctx context.Context, tx pgx.Tx, warehouse *model.Warehouse) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateWarehouseRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
}

// WarehouseStockRepo defines the interface for warehouse stock persistence operations.
type WarehouseStockRepo interface {
	ListByWarehouse(ctx context.Context, tx pgx.Tx, warehouseID uuid.UUID, filter model.WarehouseStockListFilter) ([]model.WarehouseStock, int, error)
	ListByProduct(ctx context.Context, tx pgx.Tx, productID uuid.UUID) ([]model.WarehouseStock, error)
	Upsert(ctx context.Context, tx pgx.Tx, stock *model.WarehouseStock) error
	AdjustQuantity(ctx context.Context, tx pgx.Tx, warehouseID, productID uuid.UUID, variantID *uuid.UUID, delta int) error
}

// CustomerRepo defines the interface for customer persistence operations.
type CustomerRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.CustomerListFilter) ([]model.Customer, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.Customer, error)
	FindByEmail(ctx context.Context, tx pgx.Tx, email string) (*model.Customer, error)
	Create(ctx context.Context, tx pgx.Tx, customer *model.Customer) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateCustomerRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	IncrementOrderStats(ctx context.Context, tx pgx.Tx, id uuid.UUID, amount float64) error
	ListOrdersByCustomerID(ctx context.Context, tx pgx.Tx, customerID uuid.UUID, filter model.OrderListFilter) ([]model.Order, int, error)
}

// VariantRepo defines the interface for product variant persistence operations.
type VariantRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.VariantListFilter) ([]model.ProductVariant, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.ProductVariant, error)
	FindBySKU(ctx context.Context, tx pgx.Tx, sku string) ([]model.ProductVariant, error)
	FindByEAN(ctx context.Context, tx pgx.Tx, ean string) ([]model.ProductVariant, error)
	Create(ctx context.Context, tx pgx.Tx, variant *model.ProductVariant) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdateVariantRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	CountByProductID(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (int, error)
}

// PriceListRepo defines the interface for price list persistence operations.
type PriceListRepo interface {
	List(ctx context.Context, tx pgx.Tx, filter model.PriceListListFilter) ([]model.PriceList, int, error)
	FindByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*model.PriceList, error)
	Create(ctx context.Context, tx pgx.Tx, pl *model.PriceList) error
	Update(ctx context.Context, tx pgx.Tx, id uuid.UUID, req model.UpdatePriceListRequest) error
	Delete(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	ListItems(ctx context.Context, tx pgx.Tx, priceListID uuid.UUID, limit, offset int) ([]model.PriceListItem, int, error)
	CreateItem(ctx context.Context, tx pgx.Tx, item *model.PriceListItem) error
	DeleteItem(ctx context.Context, tx pgx.Tx, itemID uuid.UUID) error
	FindItemsByProduct(ctx context.Context, tx pgx.Tx, priceListID, productID uuid.UUID, variantID *uuid.UUID, quantity int) ([]model.PriceListItem, error)
}
