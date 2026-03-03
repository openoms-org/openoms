# Frontend Hooks Audit

**Total: 79 hook files, 500+ exported hooks, 463+ API endpoints covered**

---

## CRUD HOOK FACTORY

**File:** `create-crud-hooks.ts`

`createCrudHooks<TEntity, TCreate, TUpdate, TParams>()` generates:
- `useList(params)` — GET with search params
- `useGet(id)` — GET single entity
- `useCreate()` — POST
- `useUpdate(id)` — PATCH/PUT
- `useDelete()` — DELETE

Auto-invalidates React Query cache on mutations.

---

## AUTH & SESSION

### use-auth.ts
- POST `/v1/auth/login` — `login()`
- POST `/v1/auth/2fa/login` — `verify2FALogin()`
- POST `/v1/auth/register` — `register()`
- POST `/v1/auth/logout` — `logout()`

### use-public-config.ts
- GET `/v1/config/public` — `usePublicConfig()`

### use-billing.ts
- GET `/v1/billing/subscription` — `useSubscription()` (stale: 5min)

---

## CORE RESOURCES (CRUD factory)

### use-orders.ts
- CRUD: `useOrders`, `useOrder`, `useCreateOrder`, `useUpdateOrder`, `useDeleteOrder`
- POST `/v1/orders/{id}/status` — `useTransitionOrderStatus()`
- POST `/v1/orders/bulk-status` — `useBulkTransitionStatus()`
- POST `/v1/orders/{id}/duplicate` — `useDuplicateOrder()`
- GET `/v1/orders/{id}/audit` — `useOrderAudit()`
- `exportOrdersCSV()` — download CSV

### use-products.ts
- CRUD: `useProducts`, `useProduct`, `useCreateProduct`, `useUpdateProduct`, `useDeleteProduct`

### use-customers.ts
- CRUD + `useCustomerOrders(customerId)`

### use-shipments.ts
- CRUD + `useOrderShipments()`, `useCreateOrderShipment()`
- `useTransitionShipmentStatus()`, `useGenerateLabel()`, `useShipmentTracking()` (refetch 60s)
- `useBatchLabels()` (ZIP), `useCreateDispatchOrder()`

### use-returns.ts
- CRUD + `useTransitionReturnStatus()`

### use-variants.ts
- Full CRUD scoped to product: `useVariants(productId)`, etc.

### use-bundles.ts
- `useBundleComponents()`, `useBundleStock()`
- `useAddBundleComponent()`, `useUpdateBundleComponent()`, `useRemoveBundleComponent()`

### use-roles.ts
- CRUD + `usePermissionGroups()`

### use-users.ts
- `useUsers()`, `useCreateUser()`, `useUpdateUser()`, `useDeleteUser()`

### use-warehouses.ts
- CRUD + `useWarehouseStock()`, `useUpsertWarehouseStock()`, `useProductStock()`

### use-warehouse-documents.ts
- CRUD + `useConfirmWarehouseDocument()`, `useCancelWarehouseDocument()`

### use-stocktakes.ts
- CRUD + `useStartStocktake()`, `useCompleteStocktake()`, `useCancelStocktake()`
- `useStocktakeItems()`, `useRecordCount()`

---

## INVENTORY & STOCK SYNC

### use-stock-sync.ts
- Channels CRUD: `useStockSyncChannels`, `useCreateStockSyncChannel`, etc.
- Push operations: `usePushAllStock()`, `usePushChannelStock()`, `usePushProductStock()`
- `useReconcileProductStock()`, `useStockSyncEvents()`, `useStockSyncDashboard()` (refetch 30s)
- `useStockAllocations()`

---

## INTEGRATIONS

### use-integrations.ts
- CRUD + `useIntegrationsByCategory()`

---

## ALLEGRO (Comprehensive Suite)

### use-allegro-account.ts (~25 hooks)
- Account: `useAllegroAccount()`, `useAllegroBilling()`
- Return Policies: CRUD
- Warranties: CRUD
- Size Tables: CRUD + delete
- Promotions: CRUD + `useAllegroPromoBadges()`
- Delivery: `useAllegroDeliverySettings()`, shipping rates CRUD, `useAutoGenerateShippingRate()`, `useAllegroDeliveryMethods()`

### use-allegro-catalog.ts (~6 hooks)
- `useAllegroCategories()`, `useAllegroCategorySearch()`, `useAllegroCategoryParams()`
- `useAllegroProductSearch()`, `useAllegroFees()`, `useAllegroCommissions()`

### use-allegro-listings.ts (~10 hooks)
- `useAllegroOffers()`, `useDeactivateAllegroOffer()`, `useActivateAllegroOffer()`
- `useUpdateAllegroOfferStock()`, `useUpdateAllegroOfferPrice()`
- `useProductListings()`, `useCreateProductListing()`, `useDeleteProductListing()`
- `useSyncProductListing()`, `useUpdateListingSyncMode()`, `useForcePushListing()`

### use-allegro-fulfillment.ts (~10 hooks)
- `useAllegroCarriers()`, `useAllegroFulfillment()`, `useAllegroTracking()`
- `useAllegroSync()`, `useAllegroDeliveryServices()`
- `useCreateAllegroShipment()`, `useAllegroLabel()`, `useCancelAllegroShipment()`
- `useAllegroPickupProposals()`, `useScheduleAllegroPickup()`

### use-allegro-messaging.ts (~15 hooks)
- Messages: `useAllegroThreads()`, `useAllegroMessages()`, `useSendAllegroMessage()`
- Returns: `useAllegroReturns()`, `useAllegroReturn()`, `useRejectAllegroReturn()`
- Refunds: `useCreateAllegroRefund()`, `useAllegroRefunds()`
- Disputes: `useAllegroDisputes()`, `useAllegroDisputeMessages()`, `useSendAllegroDisputeMessage()`
- Ratings: `useAllegroRatings()`, `useAllegroRatingAnswer()`, `useCreateAllegroRatingAnswer()`, `useDeleteAllegroRatingAnswer()`, `useRequestAllegroRatingRemoval()`

### use-allegro-import.ts
- `useImportAllegroOffers()`

---

## SETTINGS

### use-settings.ts
- Email: `useEmailSettings()`, `useUpdateEmailSettings()`, `useSendTestEmail()`
- Company: `useCompanySettings()`, `useUpdateCompanySettings()`
- Order Statuses: `useUpdateOrderStatuses()`
- Custom Fields: `useUpdateCustomFields()`
- Inventory: `useInventorySettings()`, `useUpdateInventorySettings()`

### use-order-statuses.ts
- `useOrderStatuses()` (stale: 5min) + `statusesToMap()`

### use-custom-fields.ts
- `useCustomFields()` (stale: 5min)

### use-sms-settings.ts
- `useSMSSettings()`, `useUpdateSMSSettings()`, `useSendTestSMS()`

### use-product-categories.ts
- `useProductCategories()`, `useUpdateProductCategories()`

### use-feed-settings.ts
- `useFeedSettings()`, `useUpdateFeedSettings()`, `useRegenerateFeedToken()`

---

## INVOICING & ACCOUNTING

### use-invoices.ts
- CRUD + `useCancelInvoice()`, `useOrderInvoices()`
- `useInvoicingSettings()`, `useUpdateInvoicingSettings()`
- `useAccountingSettings()`, `useUpdateAccountingSettings()`, `useTestAccountingConnection()`

### use-ksef.ts
- `useKSeFSettings()`, `useUpdateKSeFSettings()`, `useTestKSeFConnection()`
- `useSendToKSeF()`, `useCheckKSeFStatus()`, `useBulkSendToKSeF()`

### use-vat-oss.ts
- `useVATRates()`, `useVATCountryRates()`, `useCalculateVAT()`
- `useOSSConfig()`, `useUpdateOSSConfig()`, `useOSSReport()`, `useOSSThreshold()`
- `useDownloadOSSReportCSV()`

---

## AUTOMATION & WORKFLOWS

### use-automation.ts
- CRUD + `useAutomationRuleLogs()`, `useTestAutomationRule()`, `useDelayedActions()`

---

## PRICING

### use-price-lists.ts
- CRUD + `usePriceListItems()`, `useCreatePriceListItem()`, `useDeletePriceListItem()`

### use-categories.ts
- CRUD + `useCategoryTree()`

### use-exchange-rates.ts
- CRUD + `useFetchNBPRates()`, `useConvertAmount()`

---

## SUPPLIERS & DROPSHIP

### use-suppliers.ts (~30 hooks)
- CRUD
- Sync: `useSyncSupplier()`
- Products: `useSupplierProducts()`, `useLinkSupplierProduct()`, `useUnlinkSupplierProduct()`, `useDeleteSupplierProduct()`, `useBulkDeleteSupplierProducts()`, `useImportSupplierProducts()`, `useImportSingleProduct()`
- Portal: `useSupplierPortalStatus()`, `useGeneratePortalLink()`, `useRevokePortalAccess()`
- Category Mappings: `useSupplierCategoryMappings()`, `useUpsertCategoryMapping()`, `useDeleteCategoryMapping()`
- Allegro Mappings: `useAllegroParameterMappings()`, `useBulkUpsertAllegroMappings()`, `useDeleteAllegroParameterMapping()`
- BTP Wizard: `useBTPWizardStartImport()`, `useBTPWizardImportProgress()`, `useBTPWizardSetAPIKeys()`, `useBTPWizardCompleteSyncSettings()`
- `useAllSupplierProducts()`, `useProductSupplierLink()`, `useSupplierAttributes()`, `useAllegroMappingCategories()`

### use-dropship-orders.ts
- `useDropshipOrders()`, `useDropshipOrder()`, `useOrderDropshipOrders()`
- `useCreateDropshipOrder()`, `useAutoRouteDropship()`, `useUpdateDropshipStatus()`, `useCancelDropshipOrder()`

---

## ORDER MANAGEMENT

### use-order-groups.ts
- `useOrderGroups()`, `useMergeOrders()`, `useSplitOrder()`

### use-order-import.ts
- `useImportPreview()`, `useImportOrders()`

### use-product-import.ts
- `useProductImportPreview()`, `useProductImport()`

### use-customer-import.ts
- `useCustomerImportPreview()`, `useCustomerImport()`

### use-pick-pack.ts
- Sessions CRUD + `useScanItem()`, `useMoveToPacking()`, `useMarkItemPacked()`
- `useCompletePickPackSession()`, `useCancelPickPackSession()`

### use-recurring-orders.ts
- CRUD + `usePauseRecurringOrder()`, `useResumeRecurringOrder()`, `useCancelRecurringOrder()`

---

## REPRICING & LISTING SYNC

### use-repricing.ts
- Rules CRUD + `useSimulateRepricingRule()`, `useApplyRepricingRules()`
- `useRepricingLog()`, `useRepricingSummary()`

### use-listing-sync.ts
- Configs CRUD + `useTriggerSync()`, `useTriggerSyncPrices()`, `useTriggerSyncStock()`
- `useListingSyncLogs()`

---

## ANALYTICS & REPORTING

### use-dashboard-stats.ts
- `useDashboardStats()`

### use-reports.ts
- `useTopProducts()`, `useRevenueBySource()`, `useOrderTrends()`, `usePaymentMethodStats()`

### use-segments.ts
- CRUD + `useSegmentMembers()`, `useAddSegmentMember()`, `useRemoveSegmentMember()`
- `useRunRFMAnalysis()`, `useCustomerSegments()`

---

## LOGISTICS

### use-shipping-rates.ts
- `useShippingRates()`

### use-inpost-points.ts
- `useInPostPointSearch()`, `useGeowidgetToken()`

---

## COMMUNICATION

### use-webhooks.ts
- `useWebhookConfig()`, `useUpdateWebhookConfig()`, `useWebhookDeliveries()`

### use-message-templates.ts
- CRUD factory

### use-marketing.ts
- `useMarketingStatus()`, `useSyncCustomers()`, `useCreateCampaign()`

### use-helpdesk.ts
- `useOrderTickets()`, `useAllTickets()`, `useCreateOrderTicket()`

---

## ONBOARDING

### use-onboarding.ts
- Multi-endpoint query: settings, company, integrations, stats
- Returns: `{steps, completedCount, allCompleted, isVisible, dismiss}`

### use-onboarding-wizard.ts
- `useOnboardingStatus()`, `useUpdateOnboardingStep()`, `useCompleteOnboarding()`, `useDismissOnboarding()`

---

## REAL-TIME

### use-websocket.ts
- POST `/v1/auth/ws-ticket` — obtain ticket
- WebSocket `/v1/ws?ticket=...` — auto-reconnect with exponential backoff (max 30s)
- Broadcasts events to invalidate React Query caches via `EVENT_INVALIDATION_MAP`

---

## SYSTEM

### use-audit.ts
- `useAuditLog(params)`

### use-sync-jobs.ts
- `useSyncJobs()`, `useSyncJob()`

### use-marketplace-category-mappings.ts
- `useMarketplaceCategoryMappings()`, `useUpsertMarketplaceCategoryMapping()`, `useDeleteMarketplaceCategoryMapping()`

---

## UTILITY (non-API)

- `use-service-worker.ts` — Service worker registration
- `use-keyboard-shortcuts.ts` — Keyboard navigation
- `use-group-expansion.ts` — UI state management
