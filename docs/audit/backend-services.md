# Backend Services Audit

**Total: 130 files (67 main services, 39 tests, 24 support files)**

---

## AUTH & SECURITY

### AuthService (`auth_service.go`)
- Deps: UserRepo, TenantRepo, AuditRepo, TokenService, PasswordService, LoginLockout, RefreshTokenStore
- `Register(ctx, req, ipAddress) (*TokenResponse, string, error)`
- `Login(ctx, req, ipAddress) (*LoginResult, error)`
- `Verify2FALogin(ctx, tempToken, code) (*TokenResponse, string, error)`
- `Setup2FA(ctx, userID, tenantID, email) (*TwoFASetupResponse, error)`
- `Verify2FA(ctx, userID, tenantID, code) error`
- `Disable2FA(ctx, userID, tenantID, password, code) error`
- `Get2FAStatus(ctx, userID, tenantID) (*TwoFAStatusResponse, error)`
- `ValidateRefreshToken(tokenStr) (*RefreshTokenInfo, error)`
- `Logout(ctx, userID, tenantID) error`
- `LogoutWithRefreshToken(ctx, userID, tenantID, refreshToken) error`
- `Refresh(ctx, refreshToken) (*TokenResponse, string, error)`

### TokenService (`token_service.go`)
- Generate/validate JWT tokens (access, refresh, 2FA pending)

### PasswordService (`password_service.go`)
- Hash, Compare, ValidateStrength

### LoginLockout (`login_lockout.go` + redis/memory impls)
- Track failed login attempts per account

### RefreshTokenStore (`refresh_token_store.go` + redis/memory impls)
- Track refresh token families + reuse detection

---

## USER & TENANT

### UserService (`user_service.go`)
- Deps: UserRepo, RoleRepo, TenantRepo, AuditRepo
- `List`, `Get`, `Create`, `Update`, `Delete`, `ChangePassword`, `GetMe`, `UpdateMe`

### RoleService (`role_service.go`)
- Deps: RoleRepo
- `List`, `Get`, `Create`, `Update`, `Delete`, `ListPermissions`

### InvitationService (`invitation_service.go`)
- Generate/validate user invite tokens

---

## ORDERS & FULFILLMENT

### OrderService (`order_service.go`)
- Deps: OrderRepo, ShipmentRepo, ProductRepo, CustomerRepo, AuditRepo, WarehouseStockRepo, BundleService, WebhookDispatchService
- `List`, `Get`, `Create`, `Update`, `Delete`
- `TransitionStatus`, `Confirm`, `Pack`
- `BulkUpdateStatus`, `DuplicateOrder`, `Merge`, `Split`
- `Export`, `Import`, `AddTag`, `RemoveTag`, `IncrementCustomField`

### ShipmentService (`shipment_service.go`)
- Deps: ShipmentRepo, OrderRepo, ProductRepo, IntegrationService, CarbonService, LabelService, WebhookDispatchService
- `List`, `Get`, `Create`, `CreateMultiple`
- `CreateLabel`, `GetLabel`, `UpdateTracking`, `Cancel`
- `DispatchOrder`, `BatchDispatch`

### ReturnService (`return_service.go`)
- Deps: ReturnRepo, OrderRepo, AuditRepo, WebhookDispatchService
- `List`, `Get`, `Create`, `UpdateStatus`, `PrintLabel`
- `SubmitPublic`, `GetPublicStatus`

---

## PRODUCTS & INVENTORY

### ProductService (`product_service.go`)
- Deps: ProductRepo, VariantRepo, CategoryRepo, AuditRepo, WebhookDispatchService
- `List`, `Get`, `Create`, `Update`, `Delete`, `Export`, `Import`
- `ListStock`, `UpdateStock`, `BulkUpdateStock`

### VariantService (`variant_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`

### BundleService (`bundle_service.go`)
- `ListComponents`, `AddComponent`, `UpdateComponent`, `RemoveComponent`
- `CalculateBundleStock`, `DecrementComponentStock`

### ProductCategoryService (`product_category_service.go`)
- `List`, `Create`, `Update`, `Delete`
- `ResolveMarketplaceCategory`, `GetDescendantIDs`

### WarehouseService (`warehouse_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`
- `GetStock`, `UpdateStock`, `SetDefault`

### StocktakeService (`stocktake_service.go`)
- `List`, `Get`, `Create`, `Start`, `CountItem`, `Complete`, `Cancel`

### WarehouseDocumentService (`warehouse_document_service.go`)
- Create/confirm/cancel PZ/WZ/MM inventory documents

### ProductImportService (`product_import_service.go`)
- `PreviewCSV`, `ImportCSV`

### BaseLinkerProductImportService (`baselinker_product_import_service.go`)
- `PreviewCSV`, `ImportCSV` (BL-specific column mapping, variant grouping)

### ImageDownloadService (`image_download_service.go`)
- `RedownloadImages` — downloads external URLs, uploads to S3, updates product images

---

## CUSTOMERS & CRM

### CustomerService (`customer_service.go`)
- Deps: CustomerRepo, AuditRepo, WebhookDispatchService
- `List`, `Get`, `Create`, `Update`, `Delete`, `ListOrders`, `IncrementOrderStats`

### CustomerImportService (`customer_import_service.go`)
- `PreviewCSV`, `ImportCSV`

### SegmentService (`segment_service.go`)
- Customer segmentation + RFM analysis

### LoyaltyService (`loyalty_service.go`)
- Customer loyalty/rewards program

---

## MARKETPLACE INTEGRATIONS

### IntegrationService (`integration_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`
- `GetDecryptedCredentialsByProvider`, `GetDecryptedCredentialsByID`, `TestConnection`

### AllegroImportService (`allegro_import_service.go`)
- `ImportOffers` — import seller offers as products

### AllegroSyncService (`allegro_sync_service.go`)
- `SyncFulfillmentStatus`, `SyncTracking`

### StockSyncService (`stock_sync_service.go`)
- `SyncAll`, `SyncProductStock`, `OnStockChange`

### ListingSyncService (`listing_sync_service.go`)
- Sync product listings to marketplaces

### ProductListingService
- Create/update/delete marketplace listings

---

## SUPPLIERS & DROPSHIP

### SupplierService (`supplier_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`
- `ListProducts`, `LinkProduct`, `UnlinkProduct`, `SyncCatalog`

### DropshipService (`dropship_service.go`)
- `List`, `Get`, `GetByOrderID`, `Create`, `AutoRouteOrder`, `UpdateStatus`, `Cancel`

### SupplierPortalService (`supplier_portal_service.go`)
- Portal API for suppliers (token-based auth)

---

## INVOICING & BILLING

### InvoiceService (`invoice_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`
- `GeneratePDF`, `SendToKSeF`, `CheckKSeFStatus`, `BulkSendToKSeF`

### KSeFService (`ksef_service.go`)
- Polish KSeF e-invoicing system

### CheckoutService (`checkout_service.go`)
- `ListPlans`, `FindPlan`, `CreateCheckoutSession`, `GetSessionStatus`, `ClaimSession`
- `FinalizeCheckoutClaim`, `GetSubscription`

### StripeWebhookService (`stripe_webhook_service.go`)
- Handle Stripe webhook events

---

## PRICING & PROMOTIONS

### PriceListService (`price_list_service.go`)
- `List`, `Create`, `Update`, `Delete`, `ListItems`, `UpsertItem`

### RepricingService (`repricing_service.go`)
- Auto-reprice products based on rules

---

## AUTOMATION & WORKFLOWS

### AutomationService (`automation_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`
- `GetLogs`, `TestRule`, `ProcessEvent`, `ListDelayed`

### RecurringOrderService (`recurring_order_service.go`)
- Subscription/recurring order management

### OrderGroupService (`order_group_service.go`)
- Order groups for merge/split tracking

---

## AI & CONTENT

### AIService (`ai_service.go`)
- Deps: ProductRepo, TenantRepo, ProductCategoryRepo, OpenAI API
- `IsConfigured`, `SuggestCategories`, `SuggestTags`, `Categorize`, `Describe`
- `GenerateEnhancedDescription`, `ImproveDescription`, `TranslateDescription`

### BGRemovalService (`bg_removal_service.go`)
- `IsConfigured`, `RemoveBackground`

---

## REPORTING & ANALYTICS

### StatsService (`stats_service.go`)
- Dashboard KPIs, revenue trends, top products, payment methods

### CarbonService (`carbon_service.go`)
- `GetCarbonStats`, `GetCarbonCSVData`, `EstimateCarbon`, `EstimateDistance`

### ForecastService (`forecast_service.go`)
- `ForecastDemand`, `ForecastAll`, `GetReorderRecommendations`
- `GetSeasonalityAnalysis`, `GetProductVelocity`, `GetForecastConfig`, `UpdateForecastConfig`

### VATOSSService (`vat_oss_service.go`)
- VAT OSS compliance

### ReconciliationService (`reconciliation_service.go`)
- Bank/payment reconciliation

---

## COMMUNICATION

### EmailService (`email_service.go`)
- `SendOrderStatusEmail`, `SendTestEmail`

### SMSService (`sms_service.go`)
- Send SMS via SMSAPI

### WebhookDispatchService (`webhook_dispatch_service.go`)
- `Dispatch` — fire outgoing webhooks

### WebhookService (`webhook_service.go`)
- `List`, `Create`, `Update`, `Delete`, `Test`

### MessageTemplateService (`message_template_service.go`)
- Email/SMS template management

### MailchimpService (`mailchimp_service.go`)
- Mailchimp integration

### FreshdeskService (`freshdesk_service.go`)
- Freshdesk helpdesk integration

---

## CONFIGURATION & DATA

### ExchangeRateService (`exchange_rate_service.go`)
- `List`, `Get`, `Create`, `Update`, `Delete`, `ConvertAmount`, `FetchNBPRates`

### FeedService (`feed_service.go`)
- `GetFeedConfig`, `UpdateFeedConfig`, `RegenerateToken`, `ValidateFeedToken`
- `GenerateCeneoFeed`, `GenerateGoogleFeed`

### LicenseService (`license_service.go`)
- Validate Ed25519-signed license tokens

### PlanCacheService (`plan_cache.go`)
- In-memory cache for billing plans

---

## UTILITIES

### BarcodeService (`barcode_service.go`)
- `Lookup`, `PackOrder`

### RateService (`rate_service.go`)
- Compare shipping rates across carriers

### LabelService (`label_service.go`)
- Generate carrier labels (PDF/ZPL)

### TrackingService (`tracking_service.go`)
- Poll shipment tracking from carriers

### PickPackService (`pick_pack_service.go`)
- Pick/pack workflow operations

### PurchaseOrderService (`purchase_order_service.go`)
- Purchase order management

### BaseLinkerImportService (`baselinker_import_service.go`)
- `PreviewOrders`, `ImportOrders` (BL order CSV with row grouping)

### WSTicketService (`ws_ticket_service.go`)
- WebSocket connection ticket management

---

## KEY PATTERNS

1. **Multi-Tenancy:** Every service uses `database.WithTenant(ctx, pool, tenantID, callback)`
2. **Audit Trail:** Most CRUD operations log to AuditRepo
3. **Dependency Injection:** Via constructor function parameters
4. **Error Handling:** Custom `ErrXxx` sentinels + `NewValidationError(err)`
5. **Async Operations:** Heavy I/O uses `asyncutil.SafeGo()`
6. **Repository Pattern:** All data access through repos
7. **Event-Driven:** Automation, webhooks, sync triggered by service events
