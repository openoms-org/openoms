# Backend Models & Repositories Audit

**Models: 68 files, 326+ exported structs, 60+ Validate() methods**
**Repositories: 47 files, 23+ interfaces**

---

## MODELS

### Core Domain

**Order** (`order.go`):
- `Order`, `CreateOrderRequest` (Validate), `UpdateOrderRequest` (Validate)
- `StatusTransitionRequest`, `BulkStatusTransitionRequest`, `BulkStatusResult`
- `OrderListFilter`, `StatusDef`, `OrderStatusConfig` (IsValidStatus, CanTransition, GetStatusDef)
- `CustomFieldDef`, `CustomFieldsConfig`, `CategoryDef`, `ProductCategoriesConfig`
- Functions: `IsValidPriority()`, `IsValidFieldType()`, `DefaultOrderStatusConfig()`

**Product** (`product.go`):
- `Product`, `CreateProductRequest` (Validate), `UpdateProductRequest` (Validate)
- `ProductListFilter`

**Shipment** (`shipment.go`):
- `Shipment`, `CreateShipmentRequest` (Validate), `UpdateShipmentRequest` (Validate)
- `ShipmentStatusTransitionRequest`, `GenerateLabelRequest`, `ShipmentListFilter`
- `CreateDispatchOrderRequest`, `DispatchOrderResponse`, `BatchLabelsRequest`, `BatchLabelResult`

**Return** (`return.go`):
- `Return`, `PublicReturnRequest` (Validate), `CreateReturnRequest` (Validate)
- `UpdateReturnRequest`, `ReturnStatusRequest`, `ReturnListFilter`

**Variant** (`variant.go`):
- `ProductVariant`, `CreateVariantRequest` (Validate), `UpdateVariantRequest` (Validate), `VariantListFilter`

**Bundle** (`bundle.go`):
- `ProductBundle`, `CreateBundleComponentRequest` (Validate), `UpdateBundleComponentRequest` (Validate)

**Customer** (`customer.go`):
- `Customer`, `CreateCustomerRequest` (Validate), `UpdateCustomerRequest` (Validate), `CustomerListFilter`

### Auth & Users

**User/Tenant** (`user.go`):
- `User`, `Tenant`, `EmailSettings` (Validate), `CompanySettings`
- `SMSSettings` (Validate), `InventorySettings`, `OnboardingSettings`
- `AuditEntry`

**Auth** (`auth.go`):
- `AuthClaims`, `LoginRequest` (Validate), `RegisterRequest` (Validate)
- `TokenResponse`, `LoginResponse`
- `TwoFALoginRequest`, `TwoFAVerifyRequest`, `TwoFADisableRequest`
- `TwoFASetupResponse`, `TwoFAStatusResponse`
- `CreateUserRequest` (Validate), `UpdateUserRequest` (Validate)

### Integrations & Marketplaces

**Integration** (`integration.go`):
- `Integration`, `IntegrationWithCreds`
- `CreateIntegrationRequest` (Validate), `UpdateIntegrationRequest` (Validate)

**Product Listing** (`product_listing.go`):
- `ProductListing`, `CreateProductListingRequest` (Validate), `UpdateProductListingRequest` (Validate)
- `AllegroImportResult`, `AllegroImportDetail`, `SyncJob`, `SyncJobListFilter`

**Product Category** (`product_category.go`):
- `ProductCategory`, `CreateCategoryRequest` (Validate), `UpdateCategoryRequest` (Validate)
- `CategoryListFilter`, `MarketplaceCategoryMapping`, `UpsertMarketplaceCategoryMappingRequest` (Validate)

### Suppliers

**Supplier** (`supplier.go`):
- `Supplier`, `SupplierProduct`, `CreateSupplierRequest` (Validate), `UpdateSupplierRequest` (Validate)
- `SupplierListFilter`, `SupplierProductListFilter`
- `ImportSupplierProductsRequest` (Validate), `ImportSupplierProductsResponse`
- `SupplierCategoryMapping`, `UpsertCategoryMappingRequest` (Validate)
- `BulkDeleteSupplierProductsRequest` (Validate), `LinkProductRequest` (Validate)
- BTP Wizard: `BTPWizardStartImportRequest`, `BTPWizardSetAPIKeysRequest`, `BTPWizardSyncSettingsRequest`
- `AllegroParameterMapping`, `UpsertAllegroParameterMappingRequest`, `BulkUpsertAllegroMappingsRequest` (Validate)

**Supplier Portal** (`supplier_portal.go`):
- `SupplierPortalToken`, `SupplierMessage`, `SupplierPortalPO`
- `SupplierConfirmRequest`, `SupplierShipRequest` (Validate)
- `CreateSupplierMessageRequest` (Validate), `SupplierPortalLinkResponse`

### Warehouses & Inventory

**Warehouse** (`warehouse.go`):
- `Warehouse`, `WarehouseStock`, `CreateWarehouseRequest` (Validate), `UpdateWarehouseRequest` (Validate)
- `WarehouseListFilter`, `UpsertWarehouseStockRequest` (Validate), `WarehouseStockListFilter`

**Warehouse Document** (`warehouse_document.go`):
- `WarehouseDocument`, `WarehouseDocItem`
- `CreateWarehouseDocumentRequest` (Validate), `UpdateWarehouseDocumentRequest` (Validate)
- `WarehouseDocumentListFilter`

**Stocktake** (`stocktake.go`):
- `Stocktake`, `StocktakeItem`, `StocktakeStats`
- `CreateStocktakeRequest` (Validate), `UpdateStocktakeItemRequest` (Validate)
- `StocktakeListFilter`, `StocktakeItemListFilter`

**Stock Sync** (`stock_sync.go`):
- `StockSyncChannel`, `StockSyncEvent`, `StockAllocation`, `StockSyncDashboard`, `ChannelSummary`
- `CreateStockSyncChannelRequest` (Validate), `UpdateStockSyncChannelRequest` (Validate)
- `StockSyncChannelListFilter`, `StockSyncEventListFilter`, `ManualPushRequest`

**Listing Sync** (`listing_sync.go`):
- `ListingSyncConfig`, `ListingSyncLog`
- `CreateListingSyncConfigRequest` (Validate), `UpdateListingSyncConfigRequest` (Validate)
- `ListingSyncConfigFilter`, `ListingSyncLogFilter`

### Finance & Billing

**Invoice** (`invoice.go`):
- `Invoice`, `CreateInvoiceRequest` (Validate), `InvoiceListFilter`

**Billing** (`billing.go`):
- `CheckoutSessionRequest` (Validate), `CheckoutSessionResponse`, `CheckoutSessionStatus`
- `BillingCheckoutSession`, `BillingCustomer`, `BillingSubscription`
- `SubscriptionStatus`, `PublicPlanInfo`

**Payment** (`payment.go`):
- `PaymentSettlement`, `PaymentSettlementWithTransactions`, `PaymentTransaction`
- `CreateSettlementRequest`, `CreateTransactionRequest`, `ManualMatchRequest`
- `ReconciliationSummary`, `MatchResult`, `AutoMatchResponse`

**Price List** (`price_list.go`):
- `PriceList`, `PriceListItem`
- `CreatePriceListRequest` (Validate), `UpdatePriceListRequest` (Validate)
- `CreatePriceListItemRequest` (Validate), `PriceListListFilter`, `CalculatePriceResponse`

**Exchange Rate** (`exchange_rate.go`):
- `ExchangeRate`, `CreateExchangeRateRequest` (Validate), `UpdateExchangeRateRequest` (Validate)
- `ConvertAmountRequest` (Validate), `ConvertAmountResponse`, `ExchangeRateListFilter`

### Automation

**Automation** (`automation.go`):
- `AutomationRule`, `AutomationCondition`, `AutomationAction`, `DelayedAction`
- `CreateAutomationRuleRequest` (Validate), `UpdateAutomationRuleRequest` (Validate)
- `AutomationRuleLog`, `AutomationRuleListFilter`
- `TestAutomationRuleRequest`, `TestAutomationRuleResponse`, `ConditionResult`

**Repricing** (`repricing.go`):
- `RepricingRule`, `RepricingLog`
- `MarginParams`, `CompetitiveParams`, `TimeBasedParams`, `StockBasedParams`
- `CreateRepricingRuleRequest` (Validate), `UpdateRepricingRuleRequest` (Validate)
- `RepricingRuleListFilter`, `RepricingSummary`, `SimulationResult`, `ApplyResult`

### Orders Extended

**Purchase Order** (`purchase_order.go`):
- `PurchaseOrder`, `PurchaseOrderItem`
- `CreatePurchaseOrderRequest` (Validate), `UpdatePurchaseOrderRequest` (Validate)
- `ReceiveItemsRequest` (Validate), `PurchaseOrderListFilter`

**Dropship** (`dropship.go`):
- `DropshipOrder`, `DropshipOrderItem`
- `CreateDropshipOrderRequest` (Validate), `UpdateDropshipStatusRequest` (Validate)
- `DropshipOrderListFilter`

**Recurring Order** (`recurring_order.go`):
- `RecurringOrder`, `RecurringOrderItem`
- `CreateRecurringOrderRequest` (Validate), `UpdateRecurringOrderRequest` (Validate)
- `RecurringOrderListFilter`

**Order Group** (`order_group.go`):
- `OrderGroup`, `MergeOrdersRequest` (Validate), `SplitSpec`, `SplitOrderRequest` (Validate)

### Analytics & Reporting

**Stats** (`stats.go`):
- `DashboardStats`, `OrderCounts`, `Revenue`, `DailyRevenue`, `OrderSummary`
- `TopProduct`, `SourceRevenue`, `DailyOrderTrend`

**Forecast** (`forecast.go`):
- `Forecast`, `ReorderRecommendation`, `SeasonalityData`
- `DayOfWeekSales`, `MonthSales`, `ProductVelocity`, `ForecastConfig`

**Carbon** (`carbon.go`):
- `CarbonStats`, `CarrierCarbonStats`, `MonthlyCarbonStat`

**VAT/OSS** (`vat_oss.go`):
- `VATRateSet`, `VATCalculation`, `OSSConfig`, `OSSReport`, `OSSCountryReport`, `ThresholdStatus`

### Customer Engagement

**Segments & Loyalty** (`customer_segment.go`):
- `CustomerSegment`, `SegmentMember`, `RFMScores`, `CustomerRFM`, `SegmentRules`
- `CreateSegmentRequest` (Validate), `UpdateSegmentRequest` (Validate)
- `LoyaltyProgram`, `CustomerLoyalty`, `LeaderboardEntry`
- `CreateLoyaltyProgramRequest` (Validate), `AwardPointsRequest` (Validate), `RedeemPointsRequest` (Validate)

### Communications

**Message Template** (`message_template.go`):
- `MessageTemplate`, `CreateMessageTemplateRequest` (Validate), `UpdateMessageTemplateRequest` (Validate)

**Webhook** (`webhook.go`, `webhook_subscription.go`):
- `WebhookEvent`, `WebhookConfig`, `WebhookEndpoint`, `WebhookDelivery`, `WebhookDeliveryFilter`

### Fulfillment

**Pick & Pack** (`pick_pack.go`):
- `PickPackSession`, `PickPackItem`, `PickPackStats`
- `CreatePickPackSessionRequest` (Validate), `ScanItemRequest` (Validate), `MarkPackedRequest` (Validate)

**Barcode** (`barcode.go`):
- `BarcodeLookupResponse`, `ScannedItem`, `PackOrderRequest` (Validate), `PackOrderResponse`

**Tracking** (`tracking.go`):
- `TrackingResponse`, `TrackingItem`, `TrackingShipment`, `TrackingEvent`

### Access Control

**Role** (`role.go`):
- `Role`, `CreateRoleRequest` (Validate), `UpdateRoleRequest` (Validate), `RoleListFilter`

**License** (`license.go`):
- `LicenseLimits`, `LicenseClaims` (Validate)

**Invitation** (`invitation.go`):
- `Invitation`, `CreateInvitationRequest`, `InvitationResponse`

### Infrastructure

**Pagination** (`pagination.go`):
- `ListResponse[T]` (generic), `PaginationParams`
- `ParsePagination()`, `BuildOrderByClause()`

**Validation** (`validation.go`):
- `ValidateEmail()`, `ValidateSlug()`, `ValidatePassword()`

**Workflow** (`workflow.go`):
- `WorkflowNode`, `WorkflowEdge`, `WorkflowDefinition`, `WorkflowTemplate`
- `ValidateWorkflowRequest`, `ConvertWorkflowRequest`

**Import** (`import.go`):
- `ImportColumnMapping`, `ImportPreviewRow`, `ImportPreviewResponse`, `ImportResult`, `ImportError`

**Feed** (`feed.go`):
- `ProductFeedConfig`

**Audit** (`audit.go`):
- `AuditLogEntry`, `AuditListFilter`

---

## REPOSITORIES

### Core Interfaces (from `interfaces.go`)

| Interface | Methods | Key Features |
|-----------|---------|--------------|
| `OrderRepo` | List, FindByID, Create, Update, UpdateStatus, FindByExternalID, Delete, CountThisMonth | External ID lookup for dedup |
| `UserRepo` | FindForAuth, FindByID, List, Count, Create, UpdateRole, UpdateName, Delete, TOTP methods | TOTP secret management |
| `TenantRepo` | FindBySlug, FindByID, SlugExists, Create, GetSettings, ListAllTenantIDs, UpdateSettings | Cross-tenant listing for workers |
| `ProductRepo` | List, FindByID, FindByIDs, FindBySKU, FindByEAN, Create, Update, Delete | SKU/EAN lookup |
| `ShipmentRepo` | List, FindByID, FindByExternalID, CountByOrder, Create, Update, UpdateStatus, Delete | External ID lookup |
| `ReturnRepo` | List, FindByID, FindByToken, Create, Update, UpdateStatus, Delete | Token-based public lookup |
| `CustomerRepo` | List, FindByID, FindByEmail, Create, Update, Delete, IncrementOrderStats, ListOrders | Email dedup |
| `IntegrationRepo` | List, Count, FindByID, FindByProvider, Create, Update, Delete | Provider lookup, count for limits |
| `InvoiceRepo` | List, FindByID, FindByOrderID, Create, Update, Delete, FindPendingKSeF, UpdateKSeFStatus | KSeF status management |
| `WarehouseRepo` | List, FindByID, FindDefault, Create, Update, Delete | Default warehouse |
| `WarehouseStockRepo` | ListByWarehouse, ListByProduct, Upsert, AdjustQuantity | Delta stock adjustment |
| `ProductListingRepo` | Create, Update, GetByID, FindByProductAndIntegration, ListByProduct, ListByIntegration, FindByExternalID, Delete | Multi-index lookup |
| `StatsRepo` | GetOrderCountByStatus, GetTotalRevenue, GetDailyRevenue, GetRecentOrders, GetTopProducts, GetRevenueBySource, GetOrderTrends, GetPaymentMethodStats | Read-only analytics |
| `AuditRepo` | Log, ListByEntity, List | Write + read audit trail |
| `BillingRepo` | CreateCheckoutSession, CompleteCheckoutSession, GetCheckoutSession, ClaimCheckoutSession, CreateBillingCustomer, UpsertSubscription, GetSubscriptionByTenant, SyncTenantPlan | Atomic checkout state machine |
| `AutomationRuleRepo` | List, FindByID, FindByTenantAndEvent, Create, Update, Delete, IncrementFireCount | Event-based lookup |
| `DelayedActionRepo` | Create, ListPending, MarkExecuted, ListPendingByTenant | Pending action polling |
| `RoleRepo` | List, FindByID, FindByName, Create, Update, Delete | |
| `VariantRepo` | List, FindByID, FindBySKU, FindByEAN, Create, Update, Delete, CountByProductID | |
| `ExchangeRateRepo` | List, FindByID, GetRate, Create, Upsert, Update, Delete | Currency pair lookup |
| `SupplierRepo` | List, FindByID, Create, Update, Delete, UpdateSyncStatus, UpdateLastFullSync | Sync tracking |
| `ProductCategoryRepo` | List, FindByID, FindBySlug, Create, Update, Delete, GetDescendantIDs, FuzzyMatch | Hierarchical queries |
| `LicenseRepo` | IsTokenUsed, MarkTokenUsed, UpdateClaimedTenant | JTI replay protection |

### Additional Repository Files (24)

- `supplier_category_mapping_repository.go` — ListBySupplier, FindBySourceCategory, Upsert, Delete
- `marketplace_category_mapping_repository.go` — ListByIntegration, FindByExternalID, Upsert, Delete
- `warehouse_document_repository.go` — CRUD + Confirm, Cancel, NextDocumentNumber
- `stocktake_repository.go` — CRUD + Start/Complete lifecycle, item counting
- `purchase_order_repository.go` — CRUD + GeneratePONumber, item management
- `dropship_repository.go` — CRUD + FindByOrderID
- `recurring_order_repository.go` — CRUD + FindDue, UpdateAfterCreation
- `repricing_repository.go` — Rules CRUD + log management, GetSummary
- `stock_sync_repository.go` — Channels CRUD + events, GetAvailableStock
- `listing_sync_repository.go` — Configs CRUD + logs
- `customer_segment_repository.go` — Segments CRUD + members, loyalty programs
- `allegro_parameter_mapping_repository.go` — BulkUpsert, ListBySupplierAndCategory
- `supplier_portal_repository.go` — Portal PO and messaging
- `pick_pack_repository.go` — Sessions CRUD + scanning/marking
- `loyalty_repository.go` — Programs CRUD + points management
- `payment_repository.go` — Settlements + transactions
- `message_template_repository.go` — Templates CRUD
- `bundle_repository.go` — Bundle components CRUD
- `sync_job_repository.go` — Sync job tracking
- `webhook_delivery_repository.go` — Delivery logs
- `invitation_repository.go` — Invite token management
- `price_list_repository.go` — Price lists + items
- `billing_repository.go` — Checkout sessions, subscriptions, customers
- `audit_repository.go` — Audit log write + read

### Patterns

- All accept `context.Context` + `pgx.Tx` (or `*pgxpool.Pool` for SECURITY DEFINER)
- Pagination via `model.XxxListFilter` with Limit/Offset
- RLS context set via `set_config('app.current_tenant_id')` in WithTenant wrapper
- `queryutil.go` for dynamic SQL construction
