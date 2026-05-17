# OPE-307 Shopify PushOffer Real ID Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Shopify `PushOffer` fake IDs with real Shopify product creation IDs so listing rows are not permanently broken.

**Architecture:** Extend the existing Shopify REST SDK with product creation and variant lookup. `PushOffer` will create a Shopify product, return the first real `variant_id` as `product_listings.external_id`, and stock updates will resolve that variant to its `inventory_item_id` before updating inventory levels.

**Tech Stack:** Go 1.25, OpenOMS marketplace provider interface, Shopify Admin REST SDK, `httptest`, targeted Go tests.

---

## Scope

- Fix only Shopify marketplace provider behavior for `OPE-307`.
- Do not change dashboard UI, public Helm, enterprise deploy, or unrelated Shoper/PrestaShop fake IDs in this PR.
- Preserve existing order import behavior.

## Files

- Modify: `packages/shopify-go-sdk/products.go`
  - Add `Create(ctx, data)` for `POST /products.json`.
  - Add `GetVariant(ctx, variantID)` for `GET /variants/{id}.json`.
- Modify: `packages/shopify-go-sdk/client_test.go`
  - Add SDK transport tests for product create and variant lookup.
- Modify: `apps/api-server/internal/integration/shopify/provider.go`
  - Replace fake `shopify-<uuid>` return with real product creation.
  - Return first created Shopify variant ID.
  - Resolve variant ID to inventory item ID in `UpdateStock`.
- Modify: `apps/api-server/internal/integration/shopify/provider_test.go`
  - Add regression tests proving `PushOffer` returns real numeric variant ID, not a fake local UUID.
  - Add stock update test proving variant ID is resolved through Shopify before inventory update.
- Modify: `docs/system-documentation.md`
  - Record Shopify listing behavior and external ID semantics.

## Root Cause

- Current `PushOffer` returns `shopify-<product_uuid>`, which is not a Shopify ID.
- `UpdatePrice` expects a numeric Shopify variant ID.
- `UpdateStock` currently expects a numeric Shopify inventory item ID.
- A single `external_id` must be usable by later listing operations; the stable choice is Shopify `variant_id`, with stock updates resolving variant to inventory item.

## Tasks

### Task 1: Add failing provider regression tests

- [x] Add `TestPushOfferCreatesShopifyProductAndReturnsVariantID`.
- [x] Add `TestPushOfferRejectsCreateResponseWithoutVariantID`.
- [x] Add `TestUpdateStockResolvesVariantIDToInventoryItem`.
- [x] Run:

```bash
(cd apps/api-server && go test ./internal/integration/shopify -run 'TestPushOffer|TestUpdateStock' -count=1)
```

Expected before implementation: tests fail because `PushOffer` returns `shopify-<uuid>` and `UpdateStock` treats the external ID as an inventory item ID.

### Task 2: Add failing SDK tests

- [x] Add `TestProductServiceCreate`.
- [x] Add `TestProductServiceGetVariant`.
- [x] Run:

```bash
(cd packages/shopify-go-sdk && go test ./... -run 'TestProductServiceCreate|TestProductServiceGetVariant' -count=1)
```

Expected before implementation: tests fail because SDK methods do not exist.

### Task 3: Implement SDK methods

- [x] Add `Create(ctx context.Context, data map[string]any) (*Product, error)`.
- [x] Add `GetVariant(ctx context.Context, variantID int64) (*Variant, error)`.
- [x] Run SDK targeted tests until green.

### Task 4: Implement provider behavior

- [x] Build Shopify product create payload from `listingData` with defaults from `model.Product`.
- [x] Ensure default variant data includes `price`, `sku`, `barcode`, and `inventory_quantity` when not provided by `listingData`.
- [x] Call `p.client.Products.Create`.
- [x] Fail closed if Shopify response has no first variant ID.
- [x] Return `strconv.FormatInt(created.Variants[0].ID, 10)`.
- [x] Change `UpdateStock` to parse external ID as variant ID, call `GetVariant`, then update inventory with `variant.InventoryItemID`.
- [x] Run provider targeted tests until green.

### Task 5: Documentation and validation

- [x] Update `docs/system-documentation.md` Shopify SDK row.
- [x] Run:

```bash
git diff --check
(cd apps/api-server && go test ./internal/integration/shopify -count=1)
(cd packages/shopify-go-sdk && go test ./... -count=1)
./scripts/local-ci.sh
```

## Risks And Rollback

- Risk: existing manually-created Shopify listings may store inventory item IDs instead of variant IDs. Rollback is reverting this PR; forward follow-up can add a one-time migration/metadata enrichment if production data exists.
- Risk: Shopify REST product creation requires scopes that some tenants may not grant. The provider will return a clear create error and will not create fake listing rows.
- Rollback: revert branch commit; no database migration is planned.
