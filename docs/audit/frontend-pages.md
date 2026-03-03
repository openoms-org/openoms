# Frontend Pages Audit

**Total: 133 page.tsx files**

---

## AUTH (4 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/login` | LoginPage | POST `/v1/auth/login`, POST `/v1/auth/2fa/verify` |
| `/register` | RegisterPage | GET `/v1/billing/plans`, POST `/v1/billing/checkout` |
| `/register/invite` | InviteRegisterForm | POST `/v1/auth/register` (invite/license token) |
| `/register/complete` | CompleteRegistrationForm | GET `/v1/billing/checkout/{sessionId}`, POST `/v1/auth/register` |

## ONBOARDING (1 page)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/onboarding` | OnboardingPage | PUT `/v1/settings/company`, POST `/v1/warehouses`, POST `/v1/integrations`, POST `/v1/invitations` |

## DASHBOARD (1 page)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/` | DashboardPage | GET `/v1/stats/*`, KPI cards, revenue chart, order status/source charts |

## PUBLIC (4 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/track/[tenant_slug]` | TrackingPage | GET `/v1/tracking/{tenantSlug}/{orderId}` |
| `/supplier-portal` | SupplierPortalContent | GET/POST `/v1/supplier-portal/*` |
| `/return-request` | PublicReturnForm | POST `/v1/public/returns` |
| `/return-request/[token]` | PublicReturnStatusPage | GET `/v1/public/returns/{token}/status` |

## ORDERS (4 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/orders` | OrdersPage | GET `/v1/orders/`, bulk-status, merge, batch-labels, kanban |
| `/orders/new` | NewOrderPage | POST `/v1/orders` |
| `/orders/import` | ImportOrdersPage | POST `/v1/orders/import/preview`, POST `/v1/orders/import` |
| `/orders/[id]` | OrderDetailPage | Full order lifecycle — status, shipments, returns, split/merge, print, Allegro fulfillment |

## PRODUCTS (6 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/products` | ProductsPage (2 tabs) | GET `/v1/products/`, AI bulk-categorize, redownload images |
| `/products/new` | NewProductPage | POST `/v1/products` |
| `/products/[id]` | ProductDetailPage | PATCH/DELETE `/v1/products/{id}`, bundles, AI, bg-removal, repricing |
| `/products/[id]/variants` | ProductVariantsPage | CRUD `/v1/products/{id}/variants` |
| `/products/[id]/listings` | ProductListingsPage | Marketplace listings per product |
| `/products/import` | ImportProductsPage | Product CSV import |

## SHIPMENTS (3 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/shipments` | ShipmentsPage | GET `/v1/shipments` |
| `/shipments/new` | NewShipmentPage | POST `/v1/shipments` |
| `/shipments/[id]` | ShipmentDetailPage | Detail, label, tracking |

## RETURNS (3 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/returns` | ReturnsPage | GET `/v1/returns` |
| `/returns/new` | NewReturnPage | POST `/v1/returns` |
| `/returns/[id]` | ReturnDetailPage | Detail, status transitions |

## CUSTOMERS (5 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/customers` | CustomersPage | GET `/v1/customers` |
| `/customers/new` | NewCustomerPage | POST `/v1/customers` |
| `/customers/[id]` | CustomerDetailPage | Detail, order history |
| `/customers/import` | ImportCustomersPage | Customer CSV import |
| `/customers/segments` | CustomerSegmentsPage | CRM segmentation |

## INTEGRATIONS (3 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/integrations` | IntegrationsPage | GET `/v1/integrations` |
| `/integrations/new` | NewIntegrationPage | OAuth/API key setup |
| `/integrations/[id]` | IntegrationDetailPage | Config, sync triggers |

## MARKETPLACES (19 pages)

| Route | Component | Purpose |
|-------|-----------|---------|
| `/marketplaces` | MarketplacesPage | Overview |
| `/marketplaces/new` | NewMarketplaceIntegrationPage | Setup wizard |
| `/marketplaces/allegro` | AllegroOverviewPage | Allegro dashboard |
| `/marketplaces/allegro/offers` | AllegroOffersPage | Active listings |
| `/marketplaces/allegro/catalog` | AllegroCatalogPage | Category management |
| `/marketplaces/allegro/import` | AllegroImportPage | Import orders |
| `/marketplaces/allegro/messages` | AllegroMessagesPage | Buyer messages |
| `/marketplaces/allegro/returns` | AllegroReturnsPage | Return requests |
| `/marketplaces/allegro/disputes` | AllegroDisputesPage | Disputes |
| `/marketplaces/allegro/ratings` | AllegroRatingsPage | Seller ratings |
| `/marketplaces/allegro/shipments` | AllegroShipmentsPage | Fulfillment |
| `/marketplaces/allegro/delivery` | AllegroDeliveryPage | Shipping methods |
| `/marketplaces/allegro/promotions` | AllegroPromotionsPage | Promotions |
| `/marketplaces/allegro/policies` | AllegroPoliciesPage | Return/payment policies |
| `/marketplaces/allegro/finance` | AllegroFinancePage | Financial reports |
| `/marketplaces/amazon` | Amazon integration | |
| `/marketplaces/shopify` | Shopify integration | |
| `/marketplaces/prestashop` | PrestaShop integration | |
| `/marketplaces/shoper` | Shoper integration | |

## INVOICES (2 pages)

| Route | Component | API Endpoints |
|-------|-----------|---------------|
| `/invoices` | InvoicesPage | GET `/v1/invoices`, PDF, KSeF |
| `/invoices/[id]` | InvoiceDetailPage | Detail, KSeF send/status |

## WAREHOUSES & INVENTORY (11 pages)

| Route | Component |
|-------|-----------|
| `/settings/warehouses` | WarehousesPage |
| `/settings/warehouses/[id]` | WarehouseDetailPage |
| `/stocktakes` | StocktakesPage |
| `/stocktakes/new` | NewStocktakePage |
| `/stocktakes/[id]` | StocktakeDetailPage |
| `/stock-sync` | StockSyncPage |
| `/stock-sync/events` | StockSyncEventsPage |
| `/settings/warehouse-documents` | WarehouseDocumentsPage |
| `/settings/warehouse-documents/new` | NewWarehouseDocumentPage |
| `/settings/warehouse-documents/[id]` | DocumentDetailPage |

## SUPPLIERS & DROPSHIP (11 pages)

| Route | Component |
|-------|-----------|
| `/suppliers` | SuppliersPage |
| `/suppliers/new` | NewSupplierPage |
| `/suppliers/new/btp` | BTPortalWizardPage |
| `/suppliers/[id]` | SupplierDetailPage |
| `/suppliers/[id]/products` | SupplierProductsPage |
| `/dropship-orders` | DropshipOrdersPage |
| `/dropship-orders/[id]` | DropshipOrderDetailPage |
| `/purchase-orders` | PurchaseOrdersPage |
| `/purchase-orders/new` | NewPurchaseOrderPage |
| `/purchase-orders/[id]` | PurchaseOrderDetailPage |

## SETTINGS (32 pages)

| Route | Purpose |
|-------|---------|
| `/settings/company` | Company info (name, NIP, address) |
| `/settings/order-statuses` | Custom order status management |
| `/settings/custom-fields` | Dynamic field management |
| `/settings/users` | Team members + invitations |
| `/settings/roles` | RBAC roles |
| `/settings/roles/[id]` | Permission checkboxes |
| `/settings/security` | 2FA, password |
| `/settings/billing` | Subscription, plan, payments |
| `/settings/webhooks` | Outgoing webhook config |
| `/settings/webhooks/deliveries` | Delivery history |
| `/settings/email` | Email provider config |
| `/settings/sms` | SMS gateway config |
| `/settings/ksef` | KSeF e-invoicing |
| `/settings/invoicing` | Invoice template |
| `/settings/print-templates` | Print layout editor |
| `/settings/message-templates` | Email/SMS templates |
| `/settings/price-lists` | B2B pricing |
| `/settings/price-lists/[id]` | Price tier management |
| `/settings/product-categories` | Category tree |
| `/settings/accounting` | Accounting integration |
| `/settings/automation/rules` | Automation rules |
| `/settings/automation/[id]` | Rule detail + test |
| `/settings/feeds` | Product feed management |
| `/settings/currencies` | Exchange rates |
| `/settings/vat-oss` | VAT OSS EU config |
| `/settings/marketing` | Mailchimp integration |
| `/settings/notifications` | Notification preferences |
| `/settings/helpdesk` | Helpdesk config |
| `/settings/sync-jobs` | Background job history |

## SPECIALIZED OPERATIONS (3 pages)

| Route | Component |
|-------|-----------|
| `/packing` | PackingPage |
| `/pick-pack` | PickPackPage |
| `/pick-pack/[id]` | PickPackDetailPage |

## ADVANCED FEATURES (10 pages)

| Route | Component |
|-------|-----------|
| `/repricing` | RepricingPage |
| `/repricing/new` | NewRepricingPage |
| `/repricing/[id]` | RepricingDetailPage |
| `/listing-sync` | ListingSyncPage |
| `/listing-sync/new` | NewListingSyncPage |
| `/listing-sync/[id]` | ListingSyncDetailPage |
| `/recurring-orders` | RecurringOrdersPage |
| `/recurring-orders/new` | NewRecurringOrderPage |
| `/recurring-orders/[id]` | RecurringOrderDetailPage |
| `/reconciliation` | ReconciliationPage |

## REPORTING (4 pages)

| Route | Component |
|-------|-----------|
| `/reports` | ReportsPage |
| `/reports/vat-oss` | VATOSSReportPage |
| `/reports/carbon` | CarbonReportPage |
| `/reports/forecast` | ForecastingPage |

## MISCELLANEOUS (10 pages)

| Route | Component |
|-------|-----------|
| `/audit` | AuditLogPage |
| `/help` | HelpPage |
| `/workflows` | WorkflowsPage |
| `/workflows/new` | NewWorkflowPage |
| `/workflows/new-editor` | NewWorkflowEditorPage |
| `/workflows/[id]` | WorkflowDetailPage |
| `/loyalty` | LoyaltyPage |
| `/loyalty/[id]` | LoyaltyDetailPage |
| `/tools/bg-removal` | BackgroundRemovalPage |
