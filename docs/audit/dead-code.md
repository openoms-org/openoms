# Dead Code Detection Report

**Audit Date:** 2026-03-03

---

## Summary

| Category | Count | Risk |
|----------|-------|------|
| DEAD CODE (Frontend types in `api.ts`) | 15 types | Safe to remove or inline |
| DEAD CODE (Frontend hook interface exports) | 6 interfaces | Safe to remove `export` keyword |
| DEAD CODE (Backend unexported-safe functions) | 3 functions | Safe to make unexported |
| POSSIBLY DEAD (Sub-types never directly imported) | 6 types | Low risk, could inline |
| POSSIBLY DEAD (Backend internal-only constants) | 4 items | Low risk, could unexport |
| STUB (Feature-gated handler) | 1 handler | Feature needs wiring |

**No entire hook files, component files, or model files are dead.** All hook files have at least one page importing from them. All component files are imported by at least one page or other component. All Go model files have corresponding handler/service/repository usage.

---

## DEAD CODE: Confirmed Unused (Safe to Remove)

### Frontend -- Unused Exported Types (`apps/dashboard/src/types/api.ts`)

| Type | Line | Notes |
|------|------|-------|
| `ReturnItem` | 175 | Never imported anywhere |
| `ImportPreviewRow` | 1330 | Used internally by `ImportPreviewResponse` but never imported directly |
| `ImportError` | 1350 | Never imported anywhere |
| `InPostPointAddress` | 848 | Never imported (used as sub-type of `InPostPoint`) |
| `InPostPointAddressDetails` | 853 | Never imported |
| `ProductImage` | 364 | Never imported directly (used as `images: ProductImage[]` inside `Product`) |
| `CompanySettingsRequest` | 2993 | Never imported anywhere |
| `BulkStatusResult` | 680 | Never imported (contained in `BulkStatusTransitionResponse`) |
| `ConditionResult` | 1267 | Never imported anywhere |
| `MatchResult` | 2264 | Never imported anywhere |
| `SupplierPortalOrdersResponse` | 956 | Never imported anywhere |
| `AIBulkCategorizeResult` | 1886 | Never imported (contained in `AIBulkCategorizeResponse`) |
| `CategoryDef` | 745 | Never imported (contained in `ProductCategoriesConfig`) |
| `SplitSpec` | 1584 | Never imported (contained in `SplitOrderRequest`) |
| `WarehouseDocItem` | 1732 | Never imported anywhere |

### Frontend -- Unused Exported Interfaces in Hook Files

| Interface | File | Line | Notes |
|-----------|------|------|-------|
| `AllegroOfferList` | `use-allegro-listings.ts` | 21 | Only used internally as generic type param |
| `AllegroListingSearchItem` | `use-allegro-listings.ts` | 29 | Only used within the hook file |
| `AllegroListingSearchResult` | `use-allegro-listings.ts` | 41 | Only used within the hook file |
| `CreateWooCommerceListingRequest` | `use-allegro-listings.ts` | 69 | Only used within the hook file |
| `AllegroShipmentAddress` | `use-allegro-fulfillment.ts` | 29 | Only used within the hook file |
| `AllegroShipmentPackage` | `use-allegro-fulfillment.ts` | 40 | Only used within the hook file |

### Backend -- Exported Functions Used Only Within Package

| Function | File | Line | Notes |
|----------|------|------|-------|
| `ValidateEmail` | `model/validation.go` | 37 | Only called from within `model` package. Could be unexported. |
| `ValidateSlug` | `model/validation.go` | 52 | Only called from within `model` package. Could be unexported. |
| `ValidatePassword` | `model/validation.go` | 60 | Only called from within `model` package. Could be unexported. |

---

## POSSIBLY DEAD: Used Only Indirectly

### Frontend -- Sub-types used as struct fields but never directly imported

| Type | Used By | Notes |
|------|---------|-------|
| `ProductImage` | `Product.images` field | Could be inlined or made non-exported |
| `BulkStatusResult` | `BulkStatusTransitionResponse.results` | Could be inlined |
| `AIBulkCategorizeResult` | `AIBulkCategorizeResponse.results` | Could be inlined |
| `ImportPreviewRow` | `ImportPreviewResponse.sample_rows` | Could be inlined |
| `CategoryDef` | `ProductCategoriesConfig.categories` | Could be inlined |
| `SplitSpec` | `SplitOrderRequest.items` | Could be inlined |

### Backend -- Exported constants only used internally

| Item | File | Notes |
|------|------|-------|
| `WorkflowNodeTrigger` | `model/workflow.go` | Only used in `ValidateWorkflowDefinition` and `WorkflowToAutomationRule` |
| `WorkflowNodeCondition` | `model/workflow.go` | Same |
| `WorkflowNodeAction` | `model/workflow.go` | Same |
| `WorkflowNodeType` | `model/workflow.go` | Exported type, only used within the file |

---

## STUB: Placeholder Implementations

**1. `RedownloadImages` handler** (`product_handler.go:448`)
- Returns `501 Not Implemented` when `imageDownloadService` is nil
- Legitimate feature gate, not a real stub -- works when service is injected

**2. Worker manager TODO** (`worker/manager.go:23`)
- Design note for future multi-instance Redis locking
- Distributed lock file already exists separately
