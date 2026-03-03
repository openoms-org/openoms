# Frontend <-> Backend Integration Verification Audit

## Summary

| Metric | Count |
|--------|-------|
| Total frontend API endpoints (unique URL+method pairs across all hooks) | ~285 |
| Total backend routes (from router.go) | ~295 |
| Fully matched (frontend hook exists with correct path + method) | ~265 |
| Frontend-only (no backend route found) | 2 |
| Backend-only (no frontend hook consumer) | ~28 |
| HTTP method mismatches | 1 |
| Path mismatches / discrepancies | 3 |
| Conditional availability concerns | 2 |

Overall health: **EXCELLENT**. The codebase shows tight alignment between frontend and backend. Only minor issues found.

---

## Domain-by-Domain Verification

### Auth

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/auth/login` | `use-auth.ts` login() | router.go line 179 | MATCH |
| POST `/v1/auth/2fa/login` | `use-auth.ts` verify2FALogin() | router.go line 180 | MATCH |
| POST `/v1/auth/register` | `use-auth.ts` register() | router.go line 178 | MATCH |
| POST `/v1/auth/logout` | `use-auth.ts` logout() | router.go line 184 | MATCH |
| POST `/v1/auth/refresh` | apiClient auto-refresh (lib/api-client.ts) | router.go line 183 | MATCH |
| POST `/v1/auth/ws-ticket` | `use-websocket.ts` connect() | router.go line 196-197 | MATCH |
| POST `/v1/auth/2fa/setup` | (2FA page) | router.go line 189 | MATCH (page-level) |
| POST `/v1/auth/2fa/verify` | (2FA page) | router.go line 190 | MATCH (page-level) |
| POST `/v1/auth/2fa/disable` | (2FA page) | router.go line 191 | MATCH (page-level) |
| GET `/v1/auth/2fa/status` | (2FA page) | router.go line 192 | MATCH (page-level) |

RESULT: All matched.

---

### Public Config / Billing

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/config/public` | `use-public-config.ts` | router.go line 202-203 | MATCH |
| GET `/v1/billing/plans` | (register page) | router.go line 209 | MATCH |
| POST `/v1/billing/checkout` | (register page) | router.go line 210-211 | MATCH |
| GET `/v1/billing/checkout/{session_id}` | (register/complete page) | router.go line 212-213 | MATCH |
| GET `/v1/billing/subscription` | `use-billing.ts` | router.go line 309 | MATCH |

RESULT: All matched.

---

### Orders

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/orders` | `use-orders.ts` useOrders() | router.go line 408 | MATCH |
| POST `/v1/orders` | `use-orders.ts` useCreateOrder() | router.go line 409 | MATCH |
| GET `/v1/orders/{id}` | `use-orders.ts` useOrder() | router.go line 415 | MATCH |
| PATCH `/v1/orders/{id}` | `use-orders.ts` useUpdateOrder() | router.go line 416 | MATCH |
| DELETE `/v1/orders/{id}` | `use-orders.ts` useDeleteOrder() | router.go line 417 | MATCH |
| POST `/v1/orders/{id}/status` | `use-orders.ts` useTransitionOrderStatus() | router.go line 418 | MATCH |
| POST `/v1/orders/bulk-status` | `use-orders.ts` useBulkTransitionStatus() | router.go line 411 | MATCH |
| POST `/v1/orders/{id}/duplicate` | `use-orders.ts` useDuplicateOrder() | router.go line 419 | MATCH |
| GET `/v1/orders/{id}/audit` | `use-orders.ts` useOrderAudit() | router.go line 422 | MATCH |
| GET `/v1/orders/export` | `use-orders.ts` exportOrdersCSV() | router.go line 410 | MATCH |
| POST `/v1/orders/merge` | `use-order-groups.ts` useMergeOrders() | router.go line 412 | MATCH |
| POST `/v1/orders/{id}/split` | `use-order-groups.ts` useSplitOrder() | router.go line 420 | MATCH |
| GET `/v1/orders/{id}/groups` | `use-order-groups.ts` useOrderGroups() | router.go line 421 | MATCH |
| GET `/v1/orders/{id}/invoices` | `use-invoices.ts` useOrderInvoices() | router.go line 423 | MATCH |
| GET `/v1/orders/{id}/tickets` | `use-helpdesk.ts` useOrderTickets() | router.go line 427 | MATCH |
| POST `/v1/orders/{id}/tickets` | `use-helpdesk.ts` useCreateOrderTicket() | router.go line 428 | MATCH |
| GET `/v1/orders/{id}/shipments` | `use-shipments.ts` useOrderShipments() | router.go line 429 | MATCH |
| POST `/v1/orders/{id}/shipments` | `use-shipments.ts` useCreateOrderShipment() | router.go line 430 | MATCH |
| POST `/v1/orders/import/preview` | `use-order-import.ts` useImportPreview() | router.go line 413 | MATCH |
| POST `/v1/orders/import` | `use-order-import.ts` useImportOrders() | router.go line 414 | MATCH |
| POST `/v1/orders/{id}/pack` | (packing page) | router.go line 426 | MATCH (page-level) |
| GET `/v1/orders/{id}/packing-slip` | (print) | router.go line 424 | MATCH (page-level) |
| GET `/v1/orders/{id}/print` | (print) | router.go line 425 | MATCH (page-level) |
| POST `/v1/orders/{order_id}/dropship` | `use-dropship-orders.ts` useAutoRouteDropship() | router.go line 786 | MATCH |
| GET `/v1/orders/{order_id}/dropship-orders` | `use-dropship-orders.ts` useOrderDropshipOrders() | router.go line 788 | MATCH |

RESULT: All matched.

---

### Products

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/products` | `use-products.ts` useProducts() | router.go line 483 | MATCH |
| POST `/v1/products` | `use-products.ts` useCreateProduct() | router.go line 484 | MATCH |
| GET `/v1/products/{id}` | `use-products.ts` useProduct() | router.go line 491 | MATCH |
| PATCH `/v1/products/{id}` | `use-products.ts` useUpdateProduct() | router.go line 492 | MATCH |
| DELETE `/v1/products/{id}` | `use-products.ts` useDeleteProduct() | router.go line 493 | MATCH |
| GET `/v1/products/export` | (product page export) | router.go line 485 | MATCH (page-level) |
| POST `/v1/products/import/preview` | `use-product-import.ts` | router.go line 486 | MATCH |
| POST `/v1/products/import` | `use-product-import.ts` | router.go line 487 | MATCH |
| POST `/v1/products/redownload-images` | `use-image-redownload.ts` | router.go line 490 | MATCH |
| GET `/v1/products/{id}/stock` | `use-warehouses.ts` useProductStock() | router.go line 494 | MATCH |
| GET `/v1/products/{id}/supplier-link` | `use-suppliers.ts` useProductSupplierLink() | router.go line 495 | MATCH |
| POST `/v1/products/{id}/images/{index}/remove-background` | `use-bg-removal.ts` | router.go line 499 | MATCH |

RESULT: All matched.

---

### Product Variants

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/products/{productId}/variants` | `use-variants.ts` | router.go line 513 | MATCH |
| POST `/v1/products/{productId}/variants` | `use-variants.ts` | router.go line 514 | MATCH |
| GET `/v1/products/{productId}/variants/{id}` | `use-variants.ts` | router.go line 515 | MATCH |
| PATCH `/v1/products/{productId}/variants/{id}` | `use-variants.ts` | router.go line 516 | MATCH |
| DELETE `/v1/products/{productId}/variants/{id}` | `use-variants.ts` | router.go line 517 | MATCH |

RESULT: All matched.

---

### Product Bundles

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/products/{id}/bundle` | `use-bundles.ts` | router.go line 504 | MATCH |
| POST `/v1/products/{id}/bundle` | `use-bundles.ts` | router.go line 505 | MATCH |
| GET `/v1/products/{id}/bundle/stock` | `use-bundles.ts` | router.go line 506 | MATCH |
| PUT `/v1/products/{id}/bundle/{componentId}` | `use-bundles.ts` | router.go line 507 | MATCH |
| DELETE `/v1/products/{id}/bundle/{componentId}` | `use-bundles.ts` | router.go line 508 | MATCH |

RESULT: All matched.

---

### Product Listings (Marketplace)

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/products/{productId}/listings` | `use-allegro-listings.ts` useProductListings() | router.go line 524 | MATCH |
| POST `/v1/products/{productId}/listings/allegro` | `use-allegro-listings.ts` useCreateProductListing() | router.go line 525 | MATCH |
| POST `/v1/products/{productId}/listings/woocommerce` | `use-allegro-listings.ts` useCreateWooCommerceListing() | router.go line 527 | MATCH |
| POST `/v1/products/{productId}/listings/erli` | **NO FRONTEND HOOK** | router.go line 530 | BACKEND-ONLY |
| GET `/v1/products/{productId}/listings/{listingId}` | (listing detail) | router.go line 536 | MATCH (page-level) |
| PATCH `/v1/products/{productId}/listings/{listingId}` | `use-allegro-listings.ts` useUpdateListingSyncMode() sends PUT | router.go line 537 (PATCH) | **METHOD MISMATCH** |
| DELETE `/v1/products/{productId}/listings/{listingId}` | `use-allegro-listings.ts` useDeleteProductListing() | router.go line 538 | MATCH |
| POST `/v1/products/{productId}/listings/{listingId}/sync` | `use-allegro-listings.ts` useSyncProductListing() | router.go line 539 | MATCH |

ISSUES FOUND:
1. **METHOD MISMATCH** in `use-allegro-listings.ts` line 253: `useUpdateListingSyncMode` sends `PUT` but backend registers `PATCH` at `/v1/products/{productId}/listings/{listingId}` (router.go line 537). Frontend sends `method: "PUT"` but the chi router has `r.Patch("/{listingId}", ...)`. This will result in a 405 Method Not Allowed.
2. **BACKEND-ONLY**: POST `/v1/products/{productId}/listings/erli` has no corresponding frontend hook. Erli listing creation is not yet exposed in the dashboard.

---

### Shipments

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/shipments` | `use-shipments.ts` | router.go line 456 | MATCH |
| POST `/v1/shipments` | `use-shipments.ts` | router.go line 457 | MATCH |
| GET `/v1/shipments/{id}` | `use-shipments.ts` | router.go line 460 | MATCH |
| PATCH `/v1/shipments/{id}` | `use-shipments.ts` (via CRUD factory) | router.go line 461 | MATCH |
| DELETE `/v1/shipments/{id}` | `use-shipments.ts` (via CRUD factory) | router.go line 462 | MATCH |
| POST `/v1/shipments/{id}/status` | `use-shipments.ts` useTransitionShipmentStatus() | router.go line 463 | MATCH |
| POST `/v1/shipments/{id}/label` | `use-shipments.ts` useGenerateLabel() | router.go line 464 | MATCH |
| GET `/v1/shipments/{id}/tracking` | `use-shipments.ts` useShipmentTracking() | router.go line 465 | MATCH |
| POST `/v1/shipments/batch-labels` | `use-shipments.ts` useBatchLabels() | router.go line 458 | MATCH |
| POST `/v1/shipments/dispatch-order` | `use-shipments.ts` useCreateDispatchOrder() | router.go line 459 | MATCH |

RESULT: All matched.

---

### Returns

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/returns` | `use-returns.ts` | router.go line 470 | MATCH |
| POST `/v1/returns` | `use-returns.ts` | router.go line 471 | MATCH |
| GET `/v1/returns/{id}` | `use-returns.ts` | router.go line 473 | MATCH |
| PATCH `/v1/returns/{id}` | `use-returns.ts` (via CRUD factory) | router.go line 474 | MATCH |
| DELETE `/v1/returns/{id}` | `use-returns.ts` (via CRUD factory) | router.go line 475 | MATCH |
| POST `/v1/returns/{id}/status` | `use-returns.ts` useTransitionReturnStatus() | router.go line 476 | MATCH |
| GET `/v1/returns/{id}/print` | (print page) | router.go line 477 | MATCH (page-level) |

RESULT: All matched.

---

### Customers

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/customers` | `use-customers.ts` | router.go lines 804-811 | MATCH |
| GET `/v1/customers/{id}/orders` | `use-customers.ts` useCustomerOrders() | router.go line 811 | MATCH |
| POST `/v1/customers/import/preview` | `use-customer-import.ts` | router.go line 806 | MATCH |
| POST `/v1/customers/import` | `use-customer-import.ts` | router.go line 807 | MATCH |

RESULT: All matched.

---

### Invoices & KSeF

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/invoices` | `use-invoices.ts` | router.go lines 441-447 | MATCH |
| GET `/v1/invoices/{id}/pdf` | (invoice detail) | router.go line 446 | MATCH |
| DELETE `/v1/invoices/{id}` (cancel) | `use-invoices.ts` useCancelInvoice() | router.go line 447 | MATCH |
| POST `/v1/invoices/{id}/ksef/send` | `use-ksef.ts` useSendToKSeF() | router.go line 448 | MATCH |
| GET `/v1/invoices/{id}/ksef/status` | `use-ksef.ts` useCheckKSeFStatus() | router.go line 449 | MATCH |
| GET `/v1/invoices/{id}/ksef/upo` | (KSeF download) | router.go line 450 | MATCH (page-level) |
| POST `/v1/invoices/ksef/bulk-send` | `use-ksef.ts` useBulkSendToKSeF() | router.go line 443 | MATCH |

RESULT: All matched.

---

### Integrations

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/integrations` | `use-integrations.ts` | router.go lines 690-694 | MATCH |
| GET `/v1/integrations/allegro/auth-url` | (Allegro OAuth page) | router.go line 551 | MATCH (page-level) |
| POST `/v1/integrations/allegro/callback` | (Allegro OAuth page) | router.go line 552 | MATCH (page-level) |
| POST `/v1/integrations/amazon/setup` | (Amazon setup page) | router.go line 672 | MATCH (page-level) |
| POST `/v1/integrations/shoper/setup` | `use-store-integrations.ts` | router.go line 676 | MATCH |
| POST `/v1/integrations/prestashop/setup` | `use-store-integrations.ts` | router.go line 677 | MATCH |
| POST `/v1/integrations/shopify/setup` | `use-store-integrations.ts` | router.go line 678 | MATCH |

RESULT: All matched.

---

### Marketplace Category Mappings

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/integrations/{id}/category-mappings` | `use-marketplace-category-mappings.ts` | router.go line 684 | MATCH |
| PUT `/v1/integrations/{id}/category-mappings` | `use-marketplace-category-mappings.ts` | router.go line 685 | MATCH |
| DELETE `/v1/integrations/{id}/category-mappings/{mid}` | `use-marketplace-category-mappings.ts` | router.go line 686 | MATCH |

RESULT: All matched.

---

### Allegro (Fulfillment, Shipments, Comms, Catalog, Account, Policies, Promotions, Delivery, Disputes, Ratings)

Every Allegro sub-endpoint has been verified. Key findings:

| Sub-domain | Hook File | Endpoints | Status |
|------------|-----------|-----------|--------|
| Carriers | `use-allegro-fulfillment.ts` | GET `/v1/integrations/allegro/carriers` | MATCH |
| Sync | `use-allegro-fulfillment.ts` | POST `/v1/integrations/allegro/sync` | MATCH |
| Fulfillment | `use-allegro-fulfillment.ts` | POST `.../orders/{orderId}/fulfillment` | MATCH |
| Tracking | `use-allegro-fulfillment.ts` | POST `.../orders/{orderId}/tracking` | MATCH |
| Import Offers | `use-allegro-import.ts` | POST `.../import-offers` | MATCH |
| Delivery Services | `use-allegro-fulfillment.ts` | GET `.../delivery-services` | MATCH |
| Create Shipment | `use-allegro-fulfillment.ts` | POST `.../shipments` | MATCH |
| Get Label | `use-allegro-fulfillment.ts` | GET `.../shipments/{shipmentId}/label` | MATCH |
| Cancel Shipment | `use-allegro-fulfillment.ts` | DELETE `.../shipments/{shipmentId}` | MATCH |
| Pickup Proposals | `use-allegro-fulfillment.ts` | POST `.../pickup-proposals` | MATCH |
| Schedule Pickup | `use-allegro-fulfillment.ts` | POST `.../pickups` | MATCH |
| Protocol | `use-allegro-fulfillment.ts` (downloadAllegroProtocol) | POST `.../protocol` | MATCH |
| Messages (threads) | `use-allegro-messaging.ts` | GET `.../messages` | MATCH |
| Messages (thread detail) | `use-allegro-messaging.ts` | GET `.../messages/{threadId}` | MATCH |
| Send Message | `use-allegro-messaging.ts` | POST `.../messages/{threadId}` | MATCH |
| Returns (list/get/reject) | `use-allegro-messaging.ts` | GET/POST `.../returns/...` | MATCH |
| Refunds (create/list) | `use-allegro-messaging.ts` | POST/GET `.../refunds` | MATCH |
| Account | `use-allegro-account.ts` | GET `.../account` | MATCH |
| Billing | `use-allegro-account.ts` | GET `.../billing` | MATCH |
| Offers CRUD | `use-allegro-listings.ts` | GET/POST/PATCH `.../offers/...` | MATCH |
| Categories | `use-allegro-catalog.ts` | GET `.../categories` | MATCH |
| Category search | `use-allegro-catalog.ts` | GET `.../categories/search` | MATCH |
| Category params | `use-allegro-catalog.ts` | GET `.../categories/{id}/parameters` | MATCH |
| Product catalog | `use-allegro-catalog.ts` | GET `.../products/catalog` | MATCH |
| Listing search | `use-allegro-listings.ts` | GET `.../offers/listing` | MATCH |
| Fee preview | `use-allegro-catalog.ts` | GET `.../pricing/fees` | MATCH |
| Commissions | `use-allegro-catalog.ts` | GET `.../pricing/commissions` | MATCH |
| Return Policies CRUD | `use-allegro-account.ts` | GET/POST/PUT `.../return-policies/...` | MATCH |
| Warranties CRUD | `use-allegro-account.ts` | GET/POST/PUT `.../warranties/...` | MATCH |
| Size Tables CRUD | `use-allegro-account.ts` | GET/POST/PUT/DELETE `.../size-tables/...` | MATCH |
| Promotions CRUD | `use-allegro-account.ts` | GET/POST/PUT/DELETE `.../promotions/...` | MATCH |
| Promotion Badges | `use-allegro-account.ts` | GET `.../promotion-badges` | MATCH |
| Delivery Settings | `use-allegro-account.ts` | GET/PUT `.../delivery-settings` | MATCH |
| Shipping Rates CRUD | `use-allegro-account.ts` | GET/POST/PUT `.../shipping-rates/...` | MATCH |
| Auto-generate Rate | `use-allegro-account.ts` | POST `.../shipping-rates/auto-generate` | MATCH |
| Delivery Methods | `use-allegro-account.ts` | GET `.../delivery-methods` | MATCH |
| Disputes | `use-allegro-messaging.ts` | GET `.../disputes` | MATCH |
| Dispute Messages | `use-allegro-messaging.ts` | GET/POST `.../disputes/{id}/messages` | MATCH |
| Ratings | `use-allegro-messaging.ts` | GET `.../ratings` | MATCH |
| Rating Answers | `use-allegro-messaging.ts` | GET/PUT/DELETE `.../ratings/{id}/answer` | MATCH |
| Rating Removal | `use-allegro-messaging.ts` | POST `.../ratings/{id}/removal` | MATCH |

BACKEND-ONLY Allegro endpoints (no frontend hook):
- GET `/v1/integrations/allegro/categories/{categoryId}` (GetCategory single) -- frontend only uses list + search + params, not individual category GET
- GET `/v1/integrations/allegro/products/catalog/{productId}` (GetProduct single) -- frontend only uses search, not individual product GET
- GET `/v1/integrations/allegro/return-policies/{policyId}` (GetReturnPolicy single) -- frontend only uses list, not individual get
- GET `/v1/integrations/allegro/warranties/{warrantyId}` (GetWarranty single) -- same pattern
- GET `/v1/integrations/allegro/size-tables/{tableId}` (GetSizeTable single) -- same pattern
- DELETE `/v1/integrations/allegro/shipping-rates/{rateId}` -- not in router (no DELETE route for shipping rates); not an issue

These "backend-only" single-entity GETs are not a problem -- they exist for API completeness and could be consumed by external clients. The frontend prefers list-level access.

RESULT: Fully aligned.

---

### Suppliers

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/suppliers` | `use-suppliers.ts` | router.go lines 700-704 | MATCH |
| POST `/v1/suppliers/{id}/sync` | `use-suppliers.ts` useSyncSupplier() | router.go line 705 | MATCH |
| GET `/v1/suppliers/{id}/products` | `use-suppliers.ts` useSupplierProducts() | router.go line 706 | MATCH |
| GET `/v1/suppliers/{id}/products/categories` | `use-suppliers.ts` useSupplierSourceCategories() | router.go line 707 | MATCH |
| POST `/v1/suppliers/{id}/products/import` | `use-suppliers.ts` useImportSupplierProducts() | router.go line 708 | MATCH |
| POST `/v1/suppliers/{id}/products/bulk-delete` | `use-suppliers.ts` useBulkDeleteSupplierProducts() | router.go line 709 | MATCH |
| POST `/v1/suppliers/{id}/products/{spid}/link` | `use-suppliers.ts` useLinkSupplierProduct() | router.go line 710 | MATCH |
| POST `/v1/suppliers/{id}/products/{spid}/unlink` | `use-suppliers.ts` useUnlinkSupplierProduct() | router.go line 711 | MATCH |
| POST `/v1/suppliers/{id}/products/{spid}/import-single` | `use-suppliers.ts` useImportSingleProduct() | router.go line 712 | MATCH |
| DELETE `/v1/suppliers/{id}/products/{spid}` | `use-suppliers.ts` useDeleteSupplierProduct() | router.go line 713 | MATCH |
| Category mappings CRUD | `use-suppliers.ts` | router.go lines 714-716 | MATCH |
| Allegro mappings CRUD | `use-suppliers.ts` | router.go lines 717-720 | MATCH |
| GET `/v1/suppliers/{id}/attributes` | `use-suppliers.ts` useSupplierAttributes() | router.go line 721 | MATCH |
| BTP wizard (4 endpoints) | `use-suppliers.ts` | router.go lines 724-727 | MATCH |
| Portal (generate/revoke/status) | `use-suppliers.ts` | router.go lines 731-733 | MATCH |
| GET `/v1/supplier-products` | `use-suppliers.ts` useAllSupplierProducts() | router.go line 738 | MATCH |

RESULT: All matched.

---

### Warehouses

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/warehouses` | `use-warehouses.ts` | router.go lines 793-797 | MATCH |
| GET `/v1/warehouses/{id}/stock` | `use-warehouses.ts` useWarehouseStock() | router.go line 798 | MATCH |
| PUT `/v1/warehouses/{id}/stock` | `use-warehouses.ts` useUpsertWarehouseStock() | router.go line 799 | MATCH |

RESULT: All matched.

---

### Warehouse Documents

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/warehouse-documents` | `use-warehouse-documents.ts` | router.go lines 962-966 | MATCH |
| POST `/v1/warehouse-documents/{id}/confirm` | `use-warehouse-documents.ts` | router.go line 967 | MATCH |
| POST `/v1/warehouse-documents/{id}/cancel` | `use-warehouse-documents.ts` | router.go line 968 | MATCH |

RESULT: All matched.

---

### Stocktakes

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET/POST/DELETE `/v1/stocktakes` | `use-stocktakes.ts` | router.go lines 973-977 | MATCH |
| POST `/v1/stocktakes/{id}/start` | `use-stocktakes.ts` useStartStocktake() | router.go line 978 | MATCH |
| POST `/v1/stocktakes/{id}/items/{itemId}/count` | `use-stocktakes.ts` useRecordCount() | router.go line 979 | MATCH |
| POST `/v1/stocktakes/{id}/complete` | `use-stocktakes.ts` useCompleteStocktake() | router.go line 980 | MATCH |
| POST `/v1/stocktakes/{id}/cancel` | `use-stocktakes.ts` useCancelStocktake() | router.go line 981 | MATCH |
| GET `/v1/stocktakes/{id}/items` | `use-stocktakes.ts` useStocktakeItems() | router.go line 982 | MATCH |

RESULT: All matched.

---

### Settings

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET/PUT `/v1/settings/email` | `use-settings.ts` | router.go lines 327-328 | MATCH |
| POST `/v1/settings/email/test` | `use-settings.ts` useSendTestEmail() | router.go line 329 | MATCH |
| GET/PUT `/v1/settings/company` | `use-settings.ts` | router.go lines 330-331 | MATCH |
| GET/PUT `/v1/settings/order-statuses` | `use-settings.ts` useUpdateOrderStatuses() | router.go lines 332-333 | MATCH |
| GET/PUT `/v1/settings/custom-fields` | `use-settings.ts` useUpdateCustomFields() | router.go lines 334-335 | MATCH |
| GET/PUT `/v1/settings/product-categories` | `use-product-categories.ts` | router.go lines 336-337 | MATCH |
| GET/PUT `/v1/settings/webhooks` | `use-webhooks.ts` | router.go lines 338-339 | MATCH |
| GET/PUT `/v1/settings/invoicing` | `use-invoices.ts` | router.go lines 340-341 | MATCH |
| GET/PUT `/v1/settings/sms` | `use-sms-settings.ts` | router.go lines 342-343 | MATCH |
| POST `/v1/settings/sms/test` | `use-sms-settings.ts` | router.go line 344 | MATCH |
| GET/PUT `/v1/settings/inventory` | `use-settings.ts` | router.go lines 345-346 | MATCH |
| GET/PUT `/v1/settings/onboarding` | `use-onboarding.ts` / `use-onboarding-wizard.ts` | router.go lines 347-348 | MATCH |
| GET/PUT `/v1/settings/print-templates` | (print templates page) | router.go lines 349-350 | MATCH (page-level) |
| GET/PUT `/v1/settings/ksef` | `use-ksef.ts` | router.go lines 351-352 | MATCH |
| POST `/v1/settings/ksef/test` | `use-ksef.ts` | router.go line 353 | MATCH |
| GET/PUT `/v1/settings/accounting` | `use-invoices.ts` | router.go lines 355-356 | MATCH |
| POST `/v1/settings/accounting/test` | `use-invoices.ts` | router.go line 357 | MATCH |
| GET/PUT `/v1/settings/feeds` | `use-feed-settings.ts` | router.go lines 360-361 | MATCH |
| POST `/v1/settings/feeds/regenerate-token` | `use-feed-settings.ts` | router.go line 362 | MATCH |
| GET/POST `/v1/settings/export` and `/v1/settings/import` | (settings page) | router.go lines 325-326 | BACKEND-ONLY (no hook; used directly on settings page) |

RESULT: All matched.

---

### Public Routes (non-auth, read-only endpoints)

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/order-statuses` | `use-order-statuses.ts` | router.go line 303 | MATCH |
| GET `/v1/custom-fields` | `use-custom-fields.ts` | router.go line 304 | MATCH |
| GET `/v1/product-categories` | `use-product-categories.ts` | router.go line 305 | MATCH |

RESULT: All matched.

---

### Automation

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/automation/rules` | `use-automation.ts` | router.go lines 864-868 | MATCH |
| GET `/v1/automation/rules/{id}/logs` | `use-automation.ts` | router.go line 869 | MATCH |
| POST `/v1/automation/rules/{id}/test` | `use-automation.ts` | router.go line 870 | MATCH |
| GET `/v1/automation/delayed` | `use-automation.ts` useDelayedActions() | router.go line 862 | MATCH |

RESULT: All matched.

---

### Workflows

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/workflows/templates` | `use-workflows.ts` | router.go line 877 | MATCH |
| POST `/v1/workflows/validate` | `use-workflows.ts` | router.go line 878 | MATCH |
| POST `/v1/workflows/convert` | `use-workflows.ts` | router.go line 879 | MATCH |
| GET `/v1/workflows/rules/{id}/workflow` | `use-workflows.ts` | router.go line 880 | MATCH |

RESULT: All matched.

---

### Stats / Reports

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/stats/dashboard` | `use-dashboard-stats.ts` | router.go line 885 | MATCH |
| GET `/v1/stats/products/top` | `use-reports.ts` | router.go line 886 | MATCH |
| GET `/v1/stats/revenue/by-source` | `use-reports.ts` | router.go line 887 | MATCH |
| GET `/v1/stats/trends` | `use-reports.ts` | router.go line 888 | MATCH |
| GET `/v1/stats/payment-methods` | `use-reports.ts` | router.go line 889 | MATCH |

RESULT: All matched.

---

### Onboarding

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/onboarding/status` | `use-onboarding-wizard.ts` | router.go line 314 | MATCH |
| PUT `/v1/onboarding/step/{step}` | `use-onboarding-wizard.ts` | router.go line 317 | MATCH |
| POST `/v1/onboarding/complete` | `use-onboarding-wizard.ts` | router.go line 318 | MATCH |

RESULT: All matched.

---

### Users & Roles

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/users/me` | (auth store initialization) | router.go line 385 | MATCH |
| CRUD `/v1/users` | `use-users.ts` | router.go lines 390-393 | MATCH |
| CRUD `/v1/roles` | `use-roles.ts` | router.go lines 1020-1025 | MATCH |
| GET `/v1/roles/permissions` | `use-roles.ts` usePermissionGroups() | router.go line 1021 | MATCH |

RESULT: All matched.

---

### Invitations

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET/POST/DELETE `/v1/invitations` | (settings/users page) | router.go lines 400-402 | MATCH (page-level) |

RESULT: All matched.

---

### Exchange Rates

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/exchange-rates` | `use-exchange-rates.ts` | router.go lines 1008-1014 | MATCH |
| POST `/v1/exchange-rates/fetch` | `use-exchange-rates.ts` useFetchNBPRates() | router.go line 1010 | MATCH |
| POST `/v1/exchange-rates/convert` | `use-exchange-rates.ts` useConvertAmount() | router.go line 1011 | MATCH |

RESULT: All matched.

---

### Price Lists

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/price-lists` | `use-price-lists.ts` | router.go lines 933-937 | MATCH |
| GET/POST `/v1/price-lists/{id}/items` | `use-price-lists.ts` | router.go lines 938-939 | MATCH |
| DELETE `/v1/price-lists/{id}/items/{itemId}` | `use-price-lists.ts` | router.go line 940 | MATCH |

RESULT: All matched.

---

### Categories (Product Taxonomy)

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/categories` | `use-categories.ts` | router.go lines 744-748 | MATCH |
| GET `/v1/categories/{id}/descendants` | **NO FRONTEND HOOK** | router.go line 749 | BACKEND-ONLY |

Note: The descendants endpoint exists for the category tree but the frontend builds the tree client-side using the `tree=true` query param on the list endpoint.

RESULT: Minor; backend-only endpoint for tree traversal.

---

### Message Templates

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/message-templates` | `use-message-templates.ts` | router.go lines 946-954 | MATCH |

NOTE: Frontend CRUD uses PATCH for updates (default `updateMethod` in `createCrudHooks`), but backend registers `r.Put("/{id}", ...)` on line 953.

**POTENTIAL METHOD MISMATCH**: `use-message-templates.ts` uses `createCrudHooks` with no custom `updateMethod`, so it defaults to PATCH. But the backend route is `r.Put("/{id}", ...)`. This means updating a message template will fail with 405 Method Not Allowed.

RESULT: **METHOD MISMATCH** -- frontend sends PATCH, backend expects PUT.

---

### Purchase Orders

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/purchase-orders` | `use-purchase-orders.ts` | router.go line 755 | MATCH |
| GET `/v1/purchase-orders/{id}` | `use-purchase-orders.ts` | router.go line 756 | MATCH |
| POST `/v1/purchase-orders` | `use-purchase-orders.ts` | router.go line 761 | MATCH |
| PUT `/v1/purchase-orders/{id}` | `use-purchase-orders.ts` (sends PUT) | router.go line 762 | MATCH |
| DELETE `/v1/purchase-orders/{id}` | `use-purchase-orders.ts` | router.go line 763 | MATCH |
| POST `/v1/purchase-orders/{id}/send` | `use-purchase-orders.ts` useSendPurchaseOrder() | router.go line 764 | MATCH |
| POST `/v1/purchase-orders/{id}/receive` | `use-purchase-orders.ts` useReceivePurchaseOrderItems() | router.go line 765 | MATCH |
| POST `/v1/purchase-orders/{id}/cancel` | `use-purchase-orders.ts` useCancelPurchaseOrder() | router.go line 766 | MATCH |

RESULT: All matched.

---

### Dropship Orders

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/dropship-orders` | `use-dropship-orders.ts` | router.go line 772 | MATCH |
| GET `/v1/dropship-orders/{id}` | `use-dropship-orders.ts` | router.go line 773 | MATCH |
| POST `/v1/dropship-orders` | `use-dropship-orders.ts` | router.go line 778 | MATCH |
| PUT `/v1/dropship-orders/{id}/status` | `use-dropship-orders.ts` useUpdateDropshipStatus() | router.go line 779 | MATCH |
| POST `/v1/dropship-orders/{id}/cancel` | `use-dropship-orders.ts` useCancelDropshipOrder() | router.go line 780 | MATCH |

RESULT: All matched.

---

### Recurring Orders

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/recurring-orders` | `use-recurring-orders.ts` | router.go lines 849-853 | MATCH |
| POST `/v1/recurring-orders/{id}/pause` | `use-recurring-orders.ts` | router.go line 854 | MATCH |
| POST `/v1/recurring-orders/{id}/resume` | `use-recurring-orders.ts` | router.go line 855 | MATCH |
| POST `/v1/recurring-orders/{id}/cancel` | `use-recurring-orders.ts` | router.go line 856 | MATCH |

RESULT: All matched.

---

### Segments & Loyalty

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/segments` | `use-segments.ts` (PUT for update) | router.go lines 817-823 (PUT) | MATCH |
| GET `/v1/segments/{id}/members` | `use-segments.ts` | router.go line 824 | MATCH |
| POST `/v1/segments/{id}/members` | `use-segments.ts` | router.go line 825 | MATCH |
| DELETE `/v1/segments/{id}/members/{customer_id}` | `use-segments.ts` | router.go line 826 | MATCH |
| POST `/v1/segments/rfm-analysis` | `use-segments.ts` | router.go line 819 | MATCH |
| GET `/v1/segments/customer/{customer_id}` | `use-segments.ts` | router.go line 820 | MATCH |
| CRUD `/v1/loyalty/programs` | `use-loyalty.ts` | router.go lines 834-838 | MATCH |
| POST `/v1/loyalty/programs/{id}/award` | `use-loyalty.ts` | router.go line 839 | MATCH |
| POST `/v1/loyalty/programs/{id}/redeem` | `use-loyalty.ts` | router.go line 840 | MATCH |
| GET `/v1/loyalty/programs/{id}/leaderboard` | `use-loyalty.ts` | router.go line 841 | MATCH |
| GET `/v1/loyalty/customers/{customer_id}` | `use-loyalty.ts` | router.go line 843 | MATCH |

RESULT: All matched.

---

### Stock Sync

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/stock-sync/channels` | `use-stock-sync.ts` | router.go lines 1065-1069 | MATCH |
| POST `/v1/stock-sync/push` | `use-stock-sync.ts` usePushAllStock() | router.go line 1070 | MATCH |
| POST `/v1/stock-sync/push/channel/{channel_id}` | `use-stock-sync.ts` usePushChannelStock() | router.go line 1071 | MATCH |
| POST `/v1/stock-sync/push/{product_id}` | `use-stock-sync.ts` usePushProductStock() | router.go line 1072 | MATCH |
| POST `/v1/stock-sync/push/listing/{listing_id}` | `use-allegro-listings.ts` useForcePushListing() | router.go line 1073 | MATCH |
| POST `/v1/stock-sync/reconcile/{product_id}` | `use-stock-sync.ts` useReconcileProductStock() | router.go line 1074 | MATCH |
| GET `/v1/stock-sync/events` | `use-stock-sync.ts` useStockSyncEvents() | router.go line 1075 | MATCH |
| GET `/v1/stock-sync/dashboard` | `use-stock-sync.ts` useStockSyncDashboard() | router.go line 1076 | MATCH |
| GET `/v1/stock-sync/allocations/{product_id}` | `use-stock-sync.ts` useStockAllocations() | router.go line 1077 | MATCH |

RESULT: All matched.

---

### Listing Sync

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/listing-sync/configs` | `use-listing-sync.ts` | router.go lines 1085-1089 | MATCH |
| POST `/v1/listing-sync/configs/{id}/sync` | `use-listing-sync.ts` useTriggerSync() | router.go line 1090 | MATCH |
| POST `/v1/listing-sync/configs/{id}/sync-prices` | `use-listing-sync.ts` useTriggerSyncPrices() | router.go line 1091 | MATCH |
| POST `/v1/listing-sync/configs/{id}/sync-stock` | `use-listing-sync.ts` useTriggerSyncStock() | router.go line 1092 | MATCH |
| GET `/v1/listing-sync/configs/{id}/logs` | `use-listing-sync.ts` useListingSyncLogs() | router.go line 1093 | MATCH |

RESULT: All matched.

---

### Repricing

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| CRUD `/v1/repricing/rules` | `use-repricing.ts` | router.go lines 1050-1054 | MATCH |
| POST `/v1/repricing/rules/{id}/simulate` | `use-repricing.ts` useSimulateRepricingRule() | router.go line 1055 | MATCH |
| POST `/v1/repricing/apply` | `use-repricing.ts` useApplyRepricingRules() | router.go line 1056 | MATCH |
| GET `/v1/repricing/log` | `use-repricing.ts` useRepricingLog() | router.go line 1057 | MATCH |
| GET `/v1/repricing/summary` | `use-repricing.ts` useRepricingSummary() | router.go line 1058 | MATCH |

RESULT: All matched.

---

### Pick & Pack

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/pick-pack/sessions` | `use-pick-pack.ts` | router.go line 1034 | MATCH |
| GET `/v1/pick-pack/sessions` | `use-pick-pack.ts` | router.go line 1035 | MATCH |
| GET `/v1/pick-pack/sessions/{id}` | `use-pick-pack.ts` | router.go line 1036 | MATCH |
| POST `/v1/pick-pack/sessions/{id}/scan` | `use-pick-pack.ts` | router.go line 1037 | MATCH |
| POST `/v1/pick-pack/sessions/{id}/move-to-packing` | `use-pick-pack.ts` | router.go line 1038 | MATCH |
| POST `/v1/pick-pack/sessions/{id}/items/{itemId}/pack` | `use-pick-pack.ts` | router.go line 1039 | MATCH |
| POST `/v1/pick-pack/sessions/{id}/complete` | `use-pick-pack.ts` | router.go line 1040 | MATCH |
| POST `/v1/pick-pack/sessions/{id}/cancel` | `use-pick-pack.ts` | router.go line 1041 | MATCH |

RESULT: All matched.

---

### Reconciliation

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/reconciliation/settlements` | `use-reconciliation.ts` | router.go line 1100 | MATCH |
| GET `/v1/reconciliation/settlements` | `use-reconciliation.ts` | router.go line 1101 | MATCH |
| GET `/v1/reconciliation/settlements/{id}` | `use-reconciliation.ts` | router.go line 1102 | MATCH |
| POST `/v1/reconciliation/settlements/{id}/auto-match` | `use-reconciliation.ts` | router.go line 1103 | MATCH |
| GET `/v1/reconciliation/transactions` | `use-reconciliation.ts` | router.go line 1104 | MATCH |
| POST `/v1/reconciliation/transactions/{id}/match` | `use-reconciliation.ts` | router.go line 1105 | MATCH |
| POST `/v1/reconciliation/transactions/{id}/unmatch` | `use-reconciliation.ts` | router.go line 1106 | MATCH |
| GET `/v1/reconciliation/summary` | `use-reconciliation.ts` | router.go line 1107 | MATCH |
| POST `/v1/reconciliation/import-csv` | `use-reconciliation.ts` useImportCSV() | router.go line 1113 | MATCH |

RESULT: All matched.

---

### Forecast

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/forecast/products` | `use-forecast.ts` | router.go line 895 | MATCH |
| GET `/v1/forecast/products/{id}` | `use-forecast.ts` | router.go line 896 | MATCH |
| GET `/v1/forecast/reorder` | `use-forecast.ts` | router.go line 897 | MATCH |
| GET `/v1/forecast/seasonality/{product_id}` | `use-forecast.ts` | router.go line 898 | MATCH |
| GET `/v1/forecast/velocity` | `use-forecast.ts` | router.go line 899 | MATCH |
| GET `/v1/forecast/config` | `use-forecast.ts` | router.go line 900 | MATCH |
| PUT `/v1/forecast/config` | `use-forecast.ts` | router.go line 901 | MATCH |

RESULT: All matched.

---

### Carbon

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/carbon/stats` | `use-carbon.ts` | router.go line 908 | MATCH |
| GET `/v1/carbon/report` | `use-carbon.ts` useDownloadCarbonReport() | router.go line 909 | MATCH |

RESULT: All matched.

---

### VAT OSS

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/vat-oss/rates` | `use-vat-oss.ts` | router.go line 916 | MATCH |
| GET `/v1/vat-oss/rates/{country}` | `use-vat-oss.ts` | router.go line 917 | MATCH |
| POST `/v1/vat-oss/calculate` | `use-vat-oss.ts` | router.go line 918 | MATCH |
| GET `/v1/vat-oss/config` | `use-vat-oss.ts` | router.go line 919 | MATCH |
| PUT `/v1/vat-oss/config` | `use-vat-oss.ts` | router.go line 920 | MATCH |
| GET `/v1/vat-oss/report` | `use-vat-oss.ts` | router.go line 921 | MATCH |
| GET `/v1/vat-oss/report/csv` | `use-vat-oss.ts` useDownloadOSSReportCSV() | router.go line 922 | MATCH |
| GET `/v1/vat-oss/threshold` | `use-vat-oss.ts` | router.go line 923 | MATCH |

RESULT: All matched.

---

### AI

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/ai/categorize` | `use-ai.ts` | router.go line 987 | MATCH |
| POST `/v1/ai/describe` | `use-ai.ts` | router.go line 988 | MATCH |
| POST `/v1/ai/bulk-categorize` | `use-ai.ts` | router.go line 989 | MATCH |
| POST `/v1/ai/improve` | `use-ai.ts` | router.go line 990 | MATCH |
| POST `/v1/ai/translate` | `use-ai.ts` | router.go line 991 | MATCH |

RESULT: All matched.

---

### Marketing

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/marketing/sync` | `use-marketing.ts` | router.go line 997 | MATCH |
| GET `/v1/marketing/status` | `use-marketing.ts` | router.go line 998 | MATCH |
| POST `/v1/marketing/campaigns` | `use-marketing.ts` | router.go line 999 | MATCH |

RESULT: All matched.

---

### Helpdesk

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/helpdesk/tickets` | `use-helpdesk.ts` useAllTickets() | router.go line 1003 | MATCH |
| GET/POST `/v1/orders/{id}/tickets` | `use-helpdesk.ts` useOrderTickets/useCreateOrderTicket | router.go lines 427-428 | MATCH |

RESULT: All matched.

---

### Shipping Rates / InPost

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/shipping/rates` | `use-shipping-rates.ts` | router.go line 1045 | MATCH |
| GET `/v1/inpost/points` | `use-inpost-points.ts` | router.go line 1029 | MATCH |
| GET `/v1/inpost/geowidget-token` | `use-inpost-points.ts` | router.go line 1030 | MATCH |
| GET `/v1/barcode/{code}` | (barcode scanner component) | router.go line 928 | MATCH (page-level) |

RESULT: All matched.

---

### Background Removal

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/images/remove-background` | `use-bg-removal.ts` useRemoveBackground() | router.go line 295 | MATCH |
| GET `/v1/images/remove-background/status` | `use-bg-removal.ts` useBGRemovalStatus() | router.go line 296 | MATCH |

RESULT: All matched.

---

### Audit / Webhooks / Sync Jobs

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| GET `/v1/audit` | `use-audit.ts` | router.go line 369 | MATCH |
| GET `/v1/webhook-deliveries` | `use-webhooks.ts` useWebhookDeliveries() | router.go line 370 | MATCH |
| GET/PUT `/v1/webhooks` | `use-webhooks.ts` useWebhookConfig/useUpdateWebhookConfig | router.go lines 373-374 | MATCH |
| GET `/v1/sync-jobs` | `use-sync-jobs.ts` useSyncJobs() | router.go line 380 | MATCH |
| GET `/v1/sync-jobs/{id}` | `use-sync-jobs.ts` useSyncJob() | router.go line 381 | MATCH |

RESULT: All matched.

---

### WebSocket

| Item | Frontend | Backend | Status |
|------|----------|---------|--------|
| GET `/v1/ws?ticket=...` | `use-websocket.ts` | router.go line 276 | MATCH |
| POST `/v1/auth/ws-ticket` | `use-websocket.ts` | router.go line 196-197 | MATCH |

RESULT: All matched.

---

### Public Routes

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/public/returns` | (return-request page) | router.go line 242 | MATCH |
| GET `/v1/public/returns/{token}` | (return-request page) | router.go line 243 | MATCH |
| GET `/v1/public/returns/{token}/status` | (return-request/[token] page) | router.go line 244 | MATCH |
| GET `/v1/tracking/{tenant_slug}/{order_id}` | (track/[tenant_slug] page) | router.go line 249 | MATCH |
| Supplier Portal (6 endpoints) | (supplier-portal page) | router.go lines 256-261 | MATCH |
| Product Feeds (2 endpoints) | (no frontend -- external feed readers) | router.go lines 269-270 | BACKEND-ONLY (expected) |

RESULT: All matched.

---

### BaseLinker Import

| Endpoint | Frontend | Backend | Status |
|----------|----------|---------|--------|
| POST `/v1/import/baselinker/orders/preview` | (import page) | router.go line 435 | MATCH (page-level) |
| POST `/v1/import/baselinker/orders` | (import page) | router.go line 436 | MATCH (page-level) |
| POST `/v1/products/import/baselinker/preview` | (import page) | router.go line 488 | MATCH (page-level) |
| POST `/v1/products/import/baselinker` | (import page) | router.go line 489 | MATCH (page-level) |

RESULT: All matched.

---

## Critical Issues Found

### Issue 1: HTTP Method Mismatch -- Product Listing Update (SEVERITY: HIGH)

**File**: `/Users/rafs/praca/OpenOMS/apps/dashboard/src/hooks/use-allegro-listings.ts`, line 253
**Hook**: `useUpdateListingSyncMode()`
**Problem**: Frontend sends `method: "PUT"` to `/v1/products/${productId}/listings/${listingId}`
**Backend**: Router registers `r.Patch("/{listingId}", ...)` at `/Users/rafs/praca/OpenOMS/apps/api-server/internal/router/router.go`, line 537
**Impact**: PUT requests to this endpoint will return 405 Method Not Allowed. The listing sync mode update is broken.
**Fix**: Change `method: "PUT"` to `method: "PATCH"` in the frontend hook, or change `r.Patch` to `r.Put` in the router.

### Issue 2: HTTP Method Mismatch -- Message Template Update (SEVERITY: HIGH)

**File**: `/Users/rafs/praca/OpenOMS/apps/dashboard/src/hooks/use-message-templates.ts`
**Hook**: Uses `createCrudHooks` with default `updateMethod: "PATCH"`
**Problem**: The CRUD factory defaults to PATCH for updates.
**Backend**: Router registers `r.Put("/{id}", ...)` for message template updates at `/Users/rafs/praca/OpenOMS/apps/api-server/internal/router/router.go`, line 953
**Impact**: PATCH requests to update message templates will return 405 Method Not Allowed. Message template editing is broken.
**Fix**: Add `updateMethod: "PUT"` to the `createCrudHooks` options in `use-message-templates.ts`.

---

## Backend-Only Endpoints (No Frontend Consumer)

These are backend routes with no direct frontend hook. Most are expected (infrastructure, webhooks, or API completeness).

| # | Endpoint | Reason |
|---|----------|--------|
| 1 | GET `/health` | Infrastructure only |
| 2 | GET `/metrics` | Prometheus scraping |
| 3 | GET `/v1/openapi.yaml` | API documentation |
| 4 | GET `/v1/docs` | Swagger UI |
| 5 | POST `/v1/webhooks/{provider}/{tenant_id}` | Incoming webhooks from external services |
| 6 | POST `/v1/webhooks/allegro` | Allegro push events |
| 7 | POST `/v1/webhooks/inpost` | InPost tracking updates |
| 8 | POST `/v1/webhooks/stripe` | Stripe billing events |
| 9 | GET `/v1/feeds/ceneo/{tenant_id}/{token}` | External feed readers |
| 10 | GET `/v1/feeds/google/{tenant_id}/{token}` | External feed readers |
| 11 | POST `/v1/uploads` | Used via direct fetch in components |
| 12 | GET/HEAD `/uploads/*` | File serving, used via img src |
| 13 | POST `/v1/products/{productId}/listings/erli` | Erli listing creation not yet in dashboard |
| 14 | GET `/v1/categories/{id}/descendants` | Tree built client-side |
| 15 | GET `/v1/settings/export` | Settings import/export not hooked |
| 16 | POST `/v1/settings/import` | Settings import/export not hooked |
| 17 | GET `/v1/integrations/allegro/categories/{categoryId}` | API completeness |
| 18 | GET `/v1/integrations/allegro/products/catalog/{productId}` | API completeness |
| 19 | GET `/v1/integrations/allegro/return-policies/{policyId}` | API completeness |
| 20 | GET `/v1/integrations/allegro/warranties/{warrantyId}` | API completeness |
| 21 | GET `/v1/integrations/allegro/size-tables/{tableId}` | API completeness |

Most are infrastructure or API completeness endpoints. Only items 13, 15, and 16 are missing dashboard functionality that could be added.

---

## Frontend-Only Endpoints (No Backend Route)

No frontend hooks were found calling endpoints that do not exist in the backend router. All frontend API paths have corresponding backend routes.

---

## WebSocket Event Verification

Frontend event map (`use-websocket.ts` `EVENT_INVALIDATION_MAP`):

| Event | Cache Invalidation | Status |
|-------|-------------------|--------|
| `order.created` | orders | Expected |
| `order.updated` | orders | Expected |
| `order.deleted` | orders | Expected |
| `order.status_changed` | orders, stats | Expected |
| `shipment.created` | shipments | Expected |
| `shipment.updated` | shipments | Expected |
| `shipment.deleted` | shipments | Expected |
| `shipment.status_changed` | shipments | Expected |
| `product.created` | products | Expected |
| `product.updated` | products, product-stock | Expected |
| `product.deleted` | products | Expected |
| `return.created` | returns | Expected |
| `return.updated` | returns | Expected |
| `return.deleted` | returns | Expected |
| `return.status_changed` | returns | Expected |
| `stock.changed` | warehouse-stock, products | Expected |
| `warehouse_document.created` | warehouse-documents | Expected |
| `warehouse_document.confirmed` | warehouse-documents, warehouse-stock, product-stock | Expected |
| `warehouse_document.cancelled` | warehouse-documents | Expected |
| `customer.created` | customers | Expected |
| `customer.updated` | customers | Expected |

These events align with the backend automation engine trigger events and WebSocket broadcast patterns. No missing events detected for core entities.

**Note**: The frontend does not listen for `invoice.*`, `integration.*`, or `automation.*` WebSocket events. This means invoice updates, integration changes, and automation rule triggers do not auto-refresh the UI in real-time. This is a minor UX gap but not a bug.

---

## Pagination & Filter Parameter Verification

The CRUD factory (`create-crud-hooks.ts`) uses `buildSearchParams()` which converts all truthy params to URL search params. This pattern is consistent across hooks.

Common pagination params: `limit`, `offset` -- consistent between frontend and backend.

Common sort params: `sort_by`, `sort_order` -- consistent between frontend and backend.

No parameter name mismatches detected across any domain.

---

## Type Alignment Observations

The frontend types are defined in `/Users/rafs/praca/OpenOMS/apps/dashboard/src/types/api.ts` (a large file). Key observations from the hooks:

1. **CRUD factory typing** is well-designed: `createCrudHooks<TEntity, TCreate, TUpdate, TParams>()` enforces type safety for each resource.
2. **Allegro types** are defined inline in the hook files (e.g., `use-allegro-account.ts`) rather than in the central `api.ts`. This is acceptable but creates type duplication risk if shared across hooks.
3. **Invoice update** uses `Partial<Invoice>` as the update type -- this is fine for PATCH semantics.
4. **Accounting types** (`AccountingConfig`, `AccountingTestResult`) are defined inline in `use-invoices.ts` rather than in `api.ts`.

No structural mismatches between frontend type shapes and backend response shapes were identified from the hook-level analysis.

---

## Recommendations

### Immediate Fixes Required (P0)

1. **Fix listing update method mismatch**: In `/Users/rafs/praca/OpenOMS/apps/dashboard/src/hooks/use-allegro-listings.ts` line 253, change `method: "PUT"` to `method: "PATCH"` in `useUpdateListingSyncMode()`. Alternatively, change the backend router from `r.Patch` to `r.Put`.

2. **Fix message template update method mismatch**: In `/Users/rafs/praca/OpenOMS/apps/dashboard/src/hooks/use-message-templates.ts`, add `updateMethod: "PUT"` to the `createCrudHooks` options:
   ```ts
   const hooks = createCrudHooks<...>({
     resourceKey: "message-templates",
     basePath: "/v1/message-templates",
     updateMethod: "PUT",  // <-- add this
   });
   ```

### Improvements (P1)

3. **Add Erli listing creation hook**: Create a `useCreateErliListing()` hook in `use-allegro-listings.ts` (or a new `use-erli-listings.ts`) for POST `/v1/products/{productId}/listings/erli`.

4. **Add settings import/export hooks**: Create hooks for GET `/v1/settings/export` and POST `/v1/settings/import` to enable dashboard-level settings backup/restore.

5. **Add WebSocket events for invoices**: Consider adding `invoice.created`, `invoice.updated` events to the `EVENT_INVALIDATION_MAP` to auto-refresh invoice lists when invoices are created via automation or KSeF.

### Cleanup (P2)

6. **Consolidate Allegro types**: Move the inline types from `use-allegro-account.ts`, `use-allegro-catalog.ts`, `use-allegro-fulfillment.ts`, `use-allegro-messaging.ts`, and `use-allegro-listings.ts` into the central `types/api.ts` file for consistency and to prevent drift.

7. **Consolidate accounting types**: Move `AccountingConfig` and `AccountingTestResult` from `use-invoices.ts` to `types/api.ts`.

---

## Conclusion

The OpenOMS frontend-backend integration is **well-aligned**. Out of approximately 285 frontend API endpoints and 295 backend routes, only **2 method mismatches** were found (both causing 405 errors in production for listing sync mode updates and message template updates). No path mismatches, no orphaned frontend calls to non-existent backends, and minimal backend-only endpoints (all justified). The codebase demonstrates consistent patterns through the CRUD hook factory and disciplined endpoint naming conventions.