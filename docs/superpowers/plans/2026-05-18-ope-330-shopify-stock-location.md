# OPE-330 Shopify Stock Location Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Shopify stock updates choose an inventory location deterministically when Shopify returns multiple inventory levels.

**Architecture:** Keep Shopify provider behavior local to `apps/api-server/internal/integration/shopify`. Add a small helper that selects the lowest positive `LocationID`, avoiding dependency on response ordering and keeping the provider compatible with shops that do not yet expose per-tenant preferred location settings.

**Tech Stack:** Go 1.25, Shopify REST SDK, `httptest`, standard `testing`.

---

## Files

- Modify: `apps/api-server/internal/integration/shopify/provider.go`
  - Replace `levels[0].LocationID` with deterministic location selection.
  - Add a focused helper for inventory level selection.
- Modify: `apps/api-server/internal/integration/shopify/provider_test.go`
  - Add a regression test with multiple inventory levels returned in non-sorted order.

## Task 1: Add Regression Test

- [ ] **Step 1: Add a multi-location stock update test**

In `apps/api-server/internal/integration/shopify/provider_test.go`, add:

```go
func TestUpdateStockSelectsInventoryLocationDeterministically(t *testing.T) {
	var sawSetLevel bool

	provider, closeServer := newShopifyProviderForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/variants/222.json":
			_, _ = w.Write([]byte(`{"variant":{"id":222,"product_id":111,"inventory_item_id":333}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/inventory_levels.json":
			if got := r.URL.Query().Get("inventory_item_ids"); got != "333" {
				t.Fatalf("inventory_item_ids = %q, want 333", got)
			}
			_, _ = w.Write([]byte(`{
				"inventory_levels": [
					{"inventory_item_id":333,"location_id":555,"available":3},
					{"inventory_item_id":333,"location_id":222,"available":4},
					{"inventory_item_id":333,"location_id":444,"available":5}
				]
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/inventory_levels/set.json":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode set level body: %v", err)
			}
			if payload["location_id"] != float64(222) {
				t.Fatalf("location_id = %#v, want deterministic lowest location 222", payload["location_id"])
			}
			if payload["available"] != float64(7) {
				t.Fatalf("available = %#v, want 7", payload["available"])
			}
			sawSetLevel = true
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request = %s %s", r.Method, r.URL.String())
		}
	})
	defer closeServer()

	if err := provider.UpdateStock(context.Background(), "222", 7); err != nil {
		t.Fatalf("UpdateStock returned error: %v", err)
	}
	if !sawSetLevel {
		t.Fatal("expected inventory level update request")
	}
}
```

- [ ] **Step 2: Verify RED**

Run:

```bash
cd apps/api-server
go test ./internal/integration/shopify -run TestUpdateStockSelectsInventoryLocationDeterministically -count=1
```

Expected: FAIL because current code uses the first response level (`555`) instead of deterministic lowest location (`222`).

## Task 2: Deterministic Location Selection

- [ ] **Step 1: Add a selection helper**

In `apps/api-server/internal/integration/shopify/provider.go`, add:

```go
func selectInventoryLevelLocationID(levels []shopifysdk.InventoryLevel) (int64, error) {
	if len(levels) == 0 {
		return 0, fmt.Errorf("no inventory levels found")
	}

	locationID := levels[0].LocationID
	for _, level := range levels[1:] {
		if level.LocationID < locationID {
			locationID = level.LocationID
		}
	}
	if locationID == 0 {
		return 0, fmt.Errorf("inventory levels have no location ID")
	}
	return locationID, nil
}
```

- [ ] **Step 2: Use helper in `UpdateStock`**

Replace:

```go
return p.client.Inventory.SetLevel(ctx, variant.InventoryItemID, levels[0].LocationID, quantity)
```

with:

```go
locationID, err := selectInventoryLevelLocationID(levels)
if err != nil {
	return fmt.Errorf("shopify: select inventory location for item %d: %w", variant.InventoryItemID, err)
}

return p.client.Inventory.SetLevel(ctx, variant.InventoryItemID, locationID, quantity)
```

- [ ] **Step 3: Verify GREEN**

Run:

```bash
cd apps/api-server
go test ./internal/integration/shopify -run 'TestUpdateStock' -count=1
```

Expected: PASS.

## Task 3: Validate And Ship

- [ ] **Step 1: Run focused backend checks**

Run:

```bash
cd apps/api-server
go test ./internal/integration/shopify -count=1
```

Expected: PASS.

- [ ] **Step 2: Run repository checks**

Run from repo root:

```bash
gofmt -w -s apps/api-server/internal/integration/shopify/provider.go apps/api-server/internal/integration/shopify/provider_test.go
git diff --check
./scripts/local-ci.sh
```

Expected: all PASS before push.

- [ ] **Step 3: Commit and push**

Use a Linear-prefixed commit:

```bash
git add apps/api-server/internal/integration/shopify/provider.go \
  apps/api-server/internal/integration/shopify/provider_test.go \
  docs/superpowers/plans/2026-05-18-ope-330-shopify-stock-location.md
git commit -m "OPE-330: select Shopify stock location deterministically"
git push -u origin fix/OPE-330-shopify-stock-location
```

## Risk And Rollback

- Risk: lowest `LocationID` is deterministic, but not necessarily the merchant's preferred fulfillment location. This is still safer than response-order randomness and can later be replaced by explicit per-tenant Shopify location settings.
- Risk: shops with zero `LocationID` values now receive a clearer error before `SetLevel`.
- Rollback: revert the PR to restore first-response-level behavior. No database migration or deploy rollback is required.

## Self-Review

- Spec coverage: OPE-330 asks to stop using the first inventory level nondeterministically; Task 2 selects a stable location.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: helper uses `shopifysdk.InventoryLevel`, already imported in provider.go.
