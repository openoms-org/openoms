# Backend Handlers Audit

**Total: 87 handler files, 65+ handler structs, 430+ public methods across 15 domains**

---

## 1. AUTH & SECURITY

### AuthHandler (`auth_handler.go`)
- `Register()` — POST `/v1/auth/register`
- `Login()` — POST `/v1/auth/login`
- `TwoFALogin()` — POST `/v1/auth/2fa/login`
- `TwoFASetup()` — POST `/v1/auth/2fa/setup`
- `TwoFAVerify()` — POST `/v1/auth/2fa/verify`
- `TwoFADisable()` — POST `/v1/auth/2fa/disable`
- `TwoFAStatus()` — GET `/v1/auth/2fa/status`
- `Refresh()` — POST `/v1/auth/refresh`
- `Logout()` — POST `/v1/auth/logout`
- `WSTicket()` — POST `/v1/auth/ws-ticket`

### AllegroAuthHandler (`allegro_auth_handler.go`)
- `GetAuthURL()` — GET `/v1/integrations/allegro/auth/url`
- `HandleCallback()` — GET `/v1/integrations/allegro/auth/callback`

### AmazonAuthHandler (`amazon_auth_handler.go`)
- `Setup()` — POST `/v1/integrations/amazon/setup`

### StoreAuthHandler (`store_auth_handler.go`)
- `SetupShoper()` — POST `/v1/integrations/shoper/setup`
- `SetupPrestaShop()` — POST `/v1/integrations/prestashop/setup`
- `SetupShopify()` — POST `/v1/integrations/shopify/setup`

---

## 2. ORDERS

### OrderHandler (`order_handler.go`)
- `List()` — GET `/v1/orders` (filters: status, source, search, payment_status, tag, priority)
- `Get()` — GET `/v1/orders/{id}`
- `Create()` — POST `/v1/orders` (plan limit check)
- `Update()` — PATCH `/v1/orders/{id}`
- `Delete()` — DELETE `/v1/orders/{id}`
- `TransitionStatus()` — POST `/v1/orders/{id}/transition-status`
- `BulkTransitionStatus()` — POST `/v1/orders/bulk-transition-status`
- `GetAudit()` — GET `/v1/orders/{id}/audit`
- `DuplicateOrder()` — POST `/v1/orders/{id}/duplicate`
- `ExportCSV()` — GET `/v1/orders/export/csv`

### OrderGroupHandler (`order_group_handler.go`)
- `MergeOrders()` — POST `/v1/orders/merge`
- `SplitOrder()` — POST `/v1/orders/{id}/split`

---

## 3. PRODUCTS & VARIANTS

### ProductHandler (`product_handler.go`)
- `List()` — GET `/v1/products` (filters: name, sku, tag, category, search, category_id, supplier_id, source, marketplace)
- `Get()` — GET `/v1/products/{id}`
- `Create()` — POST `/v1/products`
- `Update()` — PATCH `/v1/products/{id}`
- `Delete()` — DELETE `/v1/products/{id}`
- `ExportCSV()` — GET `/v1/products/export/csv`
- `ImportPreview()` — POST `/v1/products/import/preview`
- `ImportCSV()` — POST `/v1/products/import`
- `BLImportPreview()` — POST `/v1/products/import/baselinker/preview`
- `BLImportCSV()` — POST `/v1/products/import/baselinker`
- `RedownloadImages()` — POST `/v1/products/redownload-images`

### VariantHandler (`variant_handler.go`)
- `List()` — GET `/v1/products/{id}/variants`
- `Get()` — GET `/v1/variants/{id}`
- `Create()` — POST `/v1/products/{id}/variants`
- `Update()` — PATCH `/v1/variants/{id}`
- `Delete()` — DELETE `/v1/variants/{id}`

### BundleHandler (`bundle_handler.go`)
- `ListComponents()` — GET `/v1/products/{id}/bundle-components`
- `AddComponent()` — POST `/v1/products/{id}/bundle-components`
- `UpdateComponent()` — PATCH `/v1/bundle-components/{id}`
- `RemoveComponent()` — DELETE `/v1/bundle-components/{id}`
- `GetBundleStock()` — GET `/v1/products/{id}/bundle-stock`

### ProductCategoryHandler (`product_category_handler.go`)
- List, Get, Create, Update, Delete (standard CRUD)

---

## 4. SHIPMENTS & LOGISTICS

### ShipmentHandler (`shipment_handler.go`)
- `ListByOrder()` — GET `/v1/orders/{id}/shipments`
- `CreateForOrder()` — POST `/v1/orders/{id}/shipments`
- `List()` — GET `/v1/shipments`
- `Get()` — GET `/v1/shipments/{id}`
- `Create()` — POST `/v1/shipments`
- `Update()` — PATCH `/v1/shipments/{id}`
- `Delete()` — DELETE `/v1/shipments/{id}`
- `TransitionStatus()` — POST `/v1/shipments/{id}/transition-status`
- `GenerateLabel()` — POST `/v1/shipments/{id}/generate-label`
- `GetTracking()` — GET `/v1/shipments/{id}/tracking`
- `CreateDispatchOrder()` — POST `/v1/shipments/dispatch-order`
- `BatchLabels()` — POST `/v1/shipments/batch-labels`

### AllegroShipmentHandler (`allegro_shipment_handler.go`)
- `ListDeliveryServices()` — GET `/v1/integrations/allegro/delivery-services`
- `CreateShipment()` — POST `/v1/integrations/allegro/shipments`
- `GetLabel()` — GET `/v1/integrations/allegro/shipments/{id}/label`
- `CancelShipment()` — POST `/v1/integrations/allegro/shipments/{id}/cancel`
- `GetPickupProposals()` — GET `/v1/integrations/allegro/shipments/pickup-proposals`
- `SchedulePickup()` — POST `/v1/integrations/allegro/shipments/pickup`
- `GenerateProtocol()` — POST `/v1/integrations/allegro/shipments/protocol`

### InPostPointHandler (`inpost_point_handler.go`)
- Search, retrieve InPost locker points

---

## 5. RETURNS & RMA

### ReturnHandler (`return_handler.go`)
- `List()` — GET `/v1/returns`
- `Get()` — GET `/v1/returns/{id}`
- `Create()` — POST `/v1/returns`
- `Update()` — PATCH `/v1/returns/{id}`
- `TransitionStatus()` — POST `/v1/returns/{id}/transition-status`
- `Delete()` — DELETE `/v1/returns/{id}`

### PublicReturnHandler (`public_return_handler.go`)
- Public return submission (no auth, rate-limited)

---

## 6. CUSTOMERS & CRM

### CustomerHandler (`customer_handler.go`)
- `List()` — GET `/v1/customers`
- `Get()` — GET `/v1/customers/{id}`
- `Create()` — POST `/v1/customers`
- `Update()` — PATCH `/v1/customers/{id}`
- `Delete()` — DELETE `/v1/customers/{id}`
- `ListOrders()` — GET `/v1/customers/{id}/orders`
- `ImportPreview()` — POST `/v1/customers/import/preview`
- `ImportCSV()` — POST `/v1/customers/import`

### SegmentHandler (`segment_handler.go`)
- `List()` — GET `/v1/segments`
- `Get()` — GET `/v1/segments/{id}`
- `Create()` — POST `/v1/segments`
- `Update()` — PATCH `/v1/segments/{id}`
- `Delete()` — DELETE `/v1/segments/{id}`
- `ListMembers()` — GET `/v1/segments/{id}/members`
- `AddMember()` — POST `/v1/segments/{id}/members`
- `RemoveMember()` — DELETE `/v1/segments/{id}/members/{customer_id}`
- `RunRFMAnalysis()` — POST `/v1/segments/{id}/rfm-analysis`
- `GetCustomerSegments()` — GET `/v1/customers/{id}/segments`

---

## 7. INVOICES & BILLING

### InvoiceHandler (`invoice_handler.go`)
- `List()` — GET `/v1/invoices`
- `Get()` — GET `/v1/invoices/{id}`
- `Create()` — POST `/v1/invoices`
- `Cancel()` — POST `/v1/invoices/{id}/cancel`
- `GetPDF()` — GET `/v1/invoices/{id}/pdf`
- `ListByOrder()` — GET `/v1/orders/{id}/invoices`

### CheckoutHandler (`checkout_handler.go`)
- Stripe checkout flow (pre-registration billing)
- `GetSubscription()` — GET `/v1/billing/subscription`

### KSeFHandler (`ksef_handler.go`)
- KSeF (Polish e-invoicing) integration

---

## 8. WAREHOUSES & INVENTORY

### WarehouseHandler (`warehouse_handler.go`)
- `List()` — GET `/v1/warehouses`
- `Get()` — GET `/v1/warehouses/{id}`
- `Create()` — POST `/v1/warehouses`
- `Update()` — PATCH `/v1/warehouses/{id}`
- `Delete()` — DELETE `/v1/warehouses/{id}`
- `ListStock()` — GET `/v1/warehouses/{id}/stock`
- `UpsertStock()` — POST `/v1/warehouses/{id}/stock`
- `ListProductStock()` — GET `/v1/products/{id}/stock`

### WarehouseDocumentHandler (`warehouse_document_handler.go`)
- `List()` — GET `/v1/warehouse-documents`
- `Get()` — GET `/v1/warehouse-documents/{id}`
- `Create()` — POST `/v1/warehouse-documents`
- `Update()` — PATCH `/v1/warehouse-documents/{id}`
- `Delete()` — DELETE `/v1/warehouse-documents/{id}`
- `Confirm()` — POST `/v1/warehouse-documents/{id}/confirm`
- `Cancel()` — POST `/v1/warehouse-documents/{id}/cancel`

### StocktakeHandler (`stocktake_handler.go`)
- `Create()` — POST `/v1/stocktakes`
- `List()` — GET `/v1/stocktakes`
- `Get()` — GET `/v1/stocktakes/{id}`
- `Delete()` — DELETE `/v1/stocktakes/{id}`
- `Start()` — POST `/v1/stocktakes/{id}/start`
- `RecordCount()` — POST `/v1/stocktakes/{id}/items/{product_id}`
- `Complete()` — POST `/v1/stocktakes/{id}/complete`
- `Cancel()` — POST `/v1/stocktakes/{id}/cancel`
- `ListItems()` — GET `/v1/stocktakes/{id}/items`

---

## 9. SETTINGS & CONFIGURATION

### SettingsHandler (`settings_handler.go`) — 35+ methods
- **Email:** `GetEmailSettings()`, `UpdateEmailSettings()`
- **Company:** `GetCompanySettings()`, `UpdateCompanySettings()`
- **Order Statuses:** `GetOrderStatuses()`, `UpdateOrderStatuses()`
- **Custom Fields:** `GetCustomFields()`, `UpdateCustomFields()`
- **Product Categories:** `GetProductCategories()`, `UpdateProductCategories()`
- **Webhooks:** `GetWebhooks()`, `UpdateWebhooks()`
- **Onboarding:** `GetOnboardingSettings()`, `UpdateOnboardingSettings()`, `GetOnboardingStatus()`, `UpdateOnboardingStep()`, `CompleteOnboarding()`
- **Invoicing:** `GetInvoicingSettings()`, `UpdateInvoicingSettings()`
- **SMS:** `GetSMSSettings()`, `UpdateSMSSettings()`, `SendTestSMS()`
- **Inventory:** `GetInventorySettings()`, `UpdateInventorySettings()`
- **Testing:** `SendTestEmail()`
- **Import/Export:** `ExportSettings()`, `ImportSettings()`

---

## 10. INTEGRATIONS & MARKETPLACES

### IntegrationHandler (`integration_handler.go`)
- `List()` — GET `/v1/integrations`
- `Get()` — GET `/v1/integrations/{id}`
- `Create()` — POST `/v1/integrations` (plan limit check)
- `Update()` — PATCH `/v1/integrations/{id}`
- `Delete()` — DELETE `/v1/integrations/{id}`
- `GetGeowidgetToken()` — GET `/v1/integrations/inpost/geowidget-token`

### AllegroHandler (`allegro_handler.go`)
- `UpdateFulfillment()`, `AddTracking()`, `ListCarriers()`, `SyncOrders()`, `ImportOffers()`

### AllegroAccountHandler (`allegro_account_handler.go`)
- `GetAccount()`, `GetBilling()`, `ListOffers()`, `DeactivateOffer()`, `ActivateOffer()`, `UpdateOfferStock()`, `UpdateOfferPrice()`

### AllegroCatalogHandler (`allegro_catalog_handler.go`)
- `ListCategories()`, `GetCategory()`, `GetCategoryParameters()`, `SearchCategories()`, `SearchProducts()`, `SearchListing()`, `GetProduct()`, `GetFeePreview()`, `GetCommissions()`

### AllegroListingsHandler (`allegro_listings_handler.go`)
- `CreateListing()`, `ListByProduct()`, `GetListing()`, `UpdateListing()`, `DeleteListing()`, `SyncListing()`

### AllegroPoliciesHandler (`allegro_policies_handler.go`)
- Return Policies: `ListReturnPolicies()`, `GetReturnPolicy()`, `CreateReturnPolicy()`, `UpdateReturnPolicy()`
- Warranties: `ListWarranties()`, `GetWarranty()`, `CreateWarranty()`, `UpdateWarranty()`
- Size Tables: `ListSizeTables()`, `GetSizeTable()`, `CreateSizeTable()`, `UpdateSizeTable()`, `DeleteSizeTable()`

### AllegroPromotionsHandler (`allegro_promotions_handler.go`)
- `ListPromotions()`, `GetPromotion()`, `CreatePromotion()`, `UpdatePromotion()`, `DeletePromotion()`, `ListBadges()`

### AllegroDeliveryHandler (`allegro_delivery_handler.go`)
- `GetDeliverySettings()`, `UpdateDeliverySettings()`, `ListShippingRates()`, `GetShippingRate()`, `CreateShippingRate()`, `UpdateShippingRate()`, `ListDeliveryMethods()`, `AutoGenerateShippingRate()`

### AllegroCommsHandler (`allegro_comms_handler.go`)
- `ListThreads()`, `GetMessages()`, `SendMessage()`, `ListAllegroReturns()`, `GetAllegroReturn()`, `RejectAllegroReturn()`, `CreateRefund()`, `ListRefunds()`

### AllegroDisputesHandler (`allegro_disputes_handler.go`)
- `ListDisputes()`, `GetDispute()`, `ListDisputeMessages()`, `SendDisputeMessage()`

### AllegroRatingsHandler (`allegro_ratings_handler.go`)
- `ListRatings()`, `GetAnswer()`, `CreateAnswer()`, `DeleteAnswer()`, `RequestRemoval()`

### WooCommerceListingsHandler (`woocommerce_listings_handler.go`)
- `CreateListing()` — POST `/v1/products/{id}/listings/woocommerce`

### ErliListingsHandler (`erli_listings_handler.go`)
- Erli marketplace listing management

---

## 11. SUPPLIERS & SOURCING

### SupplierHandler (`supplier_handler.go`) — 30+ methods
- **CRUD:** `List()`, `Get()`, `Create()`, `Update()`, `Delete()`
- **Sync:** `Sync()`, `ListSourceCategories()`, `ListProducts()`, `DeleteProduct()`, `BulkDeleteProducts()`, `UnlinkProduct()`, `LinkProduct()`
- **Category Mapping:** `ListCategoryMappings()`, `UpsertCategoryMapping()`, `DeleteCategoryMapping()`
- **Allegro Mapping:** `ListAllegroMappings()`, `BulkUpsertAllegroMappings()`, `DeleteAllegroMapping()`
- **BTP Wizard:** `BTPWizardStartImport()`, `BTPWizardImportProgress()`, `BTPWizardSetAPIKeys()`, `BTPWizardCompleteSyncSettings()`
- **Portal:** `SupplierLink()`, `ImportProducts()`, `ImportSingleProduct()`, `ListAllSupplierProducts()`

### SupplierPortalHandler (`supplier_portal_handler.go`)
- `GenerateLink()`, `RevokeAccess()`, `GetPortalStatus()`, `ListOrders()`, `GetOrder()`, `ConfirmOrder()`, `ShipOrder()`, `ListMessages()`, `AddMessage()`

---

## 12. AUTOMATION & WORKFLOWS

### AutomationHandler (`automation_handler.go`)
- `List()`, `Get()`, `Create()`, `Update()`, `Delete()`, `GetLogs()`, `ListDelayed()`, `TestRule()`

### WorkflowHandler (`workflow_handler.go`)
- `ListTemplates()`, `Validate()`, `Convert()`, `GetWorkflowForRule()`

---

## 13. ANALYTICS & REPORTING

### StatsHandler (`stats_handler.go`)
- `GetDashboard()`, `GetTopProducts()`, `GetRevenueBySource()`, `GetOrderTrends()`, `GetPaymentMethodStats()`

### ForecastHandler (`forecast_handler.go`)
- Demand forecasting, stock predictions

### VATOSSHandler (`vat_oss_handler.go`)
- `GetAllRates()`, `GetCountryRates()`, `Calculate()`, `GetConfig()`, `UpdateConfig()`, `GetReport()`, `GetReportCSV()`, `GetThreshold()`

### ReconciliationHandler (`reconciliation_handler.go`)
- Order/shipment reconciliation

### CarbonHandler (`carbon_handler.go`)
- Carbon footprint/sustainability tracking

---

## 14. ADMINISTRATIVE & SYSTEM

### UserHandler (`user_handler.go`)
- `Me()`, `List()`, `Create()`, `Update()`, `Delete()`

### RoleHandler (`role_handler.go`)
- `List()`, `Get()`, `Create()`, `Update()`, `Delete()`, `ListPermissions()`

### AuditHandler (`audit_handler.go`)
- `List()` — GET `/v1/audit-log`

### WebhookHandler (`webhook_handler.go`)
- `Receive()` — POST `/v1/webhooks/{provider}/{tenant_id}`

### WebhookDeliveryHandler (`webhook_delivery_handler.go`)
- `List()` — GET `/v1/webhook-deliveries`

### SyncJobHandler (`sync_job_handler.go`)
- `List()`, `Get()`

### StripeWebhookHandler (`stripe_webhook_handler.go`)
- `HandleWebhook()` — POST `/v1/webhooks/stripe`

### InPostWebhookHandler (`inpost_webhook_handler.go`)
- Webhook receivers for InPost events

---

## 15. UTILITIES & MISC

### UploadHandler (`upload_handler.go`)
- `Upload()` — POST `/v1/uploads`

### BarcodeHandler (`barcode_handler.go`)
- `Lookup()`, `PackOrder()`

### TrackingHandler (`tracking_handler.go`)
- `TrackOrder()` — GET `/v1/orders/{id}/track`

### ExchangeRateHandler (`exchange_rate_handler.go`)
- Currency rate management (NBP integration)

### RateHandler (`rate_handler.go`)
- `GetRates()` — POST `/v1/shipping/rates`

### PickPackHandler (`pick_pack_handler.go`)
- Pick/pack workflow

### PriceListHandler (`price_list_handler.go`)
- B2B pricing tiers

### PrintHandler (`print_handler.go`)
- Print label/document generation

### MarketplaceCategoryMappingHandler (`marketplace_category_mapping_handler.go`)
- Category mapping between local and marketplace taxonomies

### MessageTemplateHandler (`message_template_handler.go`)
- Email/SMS template management

### HelpDeskHandler (`helpdesk_handler.go`)
- Support ticket management

### ListingSyncHandler (`listing_sync_handler.go`)
- Marketplace listing synchronization

### StockSyncHandler (`stock_sync_handler.go`)
- `ListChannels()`, `GetChannel()`, `CreateChannel()`, `UpdateChannel()`, `DeleteChannel()`
- `PushAll()`, `PushChannel()`, `PushProduct()`, `PushListing()`
- `ReconcileProduct()`, `ListEvents()`, `GetDashboard()`, `GetAllocations()`

### RecurringOrderHandler (`recurring_order_handler.go`)
- Subscription/recurring order management

### RepricingHandler (`repricing_handler.go`)
- Dynamic pricing rules, `GetSummary()`

### MarketingHandler (`marketing_handler.go`)
- Mailchimp sync, campaign management

### LoyaltyHandler (`loyalty_handler.go`)
- Customer loyalty programs

### ImportHandler (`import_handler.go`)
- Generic import orchestration

### BGRemovalHandler (`bg_removal_handler.go`)
- `RemoveBackground()`, `RemoveProductImageBackground()`, `Status()`

### AIHandler (`ai_handler.go`)
- `Categorize()`, `Describe()`, `Improve()`, `Translate()`, `BulkCategorize()`

### AccountingHandler (`accounting_handler.go`)
- `GetAccountingSettings()`, `UpdateAccountingSettings()`, `TestConnection()`

### PurchaseOrderHandler (`purchase_order_handler.go`)
- Purchase order management

### DropshipHandler (`dropship_handler.go`)
- Dropshipping supplier workflows

### ConfigHandler (`config_handler.go`)
- Application configuration

### DocsHandler (`docs_handler.go`)
- OpenAPI/Swagger documentation

### HealthHandler (`health.go`)
- Health check endpoint

### InvitationHandler (`invitation_handler.go`)
- User invitation tokens

### WSHandler (`ws_handler.go`)
- `ServeWS()` — WebSocket `/v1/ws`

### FeedHandler (`feed_handler.go`)
- Product feed generation (Ceneo XML, Google RSS)

### Response Helpers (`response.go`)
- `writeJSON()`, `writeError()`, `writeServerError()`, `writeCSVHeaders()`, `clientIP()`, `isValidationError()`

### OAuth State (`oauth_state_store.go`, `oauth_state_store_redis.go`, `oauth_state_store_memory.go`)
- OAuth state token management for Allegro/Amazon flows
