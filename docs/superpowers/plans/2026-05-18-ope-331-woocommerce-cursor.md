# OPE-331 WooCommerce Cursor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix WooCommerce order polling so the sync cursor is based on deterministic modified timestamps instead of local-time string comparison.

**Architecture:** WooCommerce exposes `date_modified_gmt` and supports `modified_after`, `dates_are_gmt`, and `orderby=modified` for order listing. The SDK will expose those fields/params, and the OpenOMS provider will store future cursors as UTC RFC3339 strings while keeping a one-poll compatibility path for legacy cursors without a timezone.

**Tech Stack:** Go 1.25, WooCommerce REST API v3 SDK, `httptest`, OpenOMS marketplace provider interface.

---

## Files And Responsibilities

- Modify `packages/woocommerce-go-sdk/orders.go`
  - Add `DatesAreGMT bool` to `OrderListParams`.
  - Emit `dates_are_gmt=true` when requested.
  - Add `DateModifiedGMT string` to `WooOrder`.
  - Update the `OrderBy` comment to include WooCommerce `modified`.
- Modify `apps/api-server/internal/integration/woocommerce/provider.go`
  - Request orders sorted by `modified`, not created date.
  - Use `date_modified_gmt` as the cursor source.
  - Compare parsed UTC times, not raw strings.
  - Preserve legacy no-timezone cursors for the first filtered request, then store new cursors as UTC RFC3339.
- Modify `apps/api-server/internal/integration/woocommerce/provider_test.go`
  - Add a failing regression test with `httptest` that verifies query params and cursor advancement through `date_modified_gmt`.
- Add `docs/superpowers/plans/2026-05-18-ope-331-woocommerce-cursor.md`
  - This implementation plan.

## Task 1: Provider Regression Test

**Files:**
- Modify: `apps/api-server/internal/integration/woocommerce/provider_test.go`

- [ ] **Step 1: Write the failing test**

Add a test that drives the real provider through an `httptest.Server` and expects:

```go
func TestPollOrdersUsesGMTModifiedCursorAndModifiedOrdering(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id": 101,
				"status": "processing",
				"currency": "PLN",
				"total": "10.00",
				"date_created": "2026-03-29T01:15:00",
				"date_modified": "2026-03-30T01:30:00",
				"date_modified_gmt": "2026-03-29T23:30:00"
			},
			{
				"id": 102,
				"status": "processing",
				"currency": "PLN",
				"total": "20.00",
				"date_created": "2026-03-29T01:20:00",
				"date_modified": "2026-03-30T03:15:00",
				"date_modified_gmt": "2026-03-30T01:15:00"
			}
		]`))
	}))
	defer srv.Close()

	provider := &Provider{
		client: woocommercesdk.NewClient("https://shop.example.com", "ck", "cs",
			woocommercesdk.WithBaseURL(srv.URL),
			woocommercesdk.WithHTTPClient(srv.Client()),
		),
	}

	orders, cursor, err := provider.PollOrders(context.Background(), "2026-03-29T00:30:00Z")
	if err != nil {
		t.Fatalf("PollOrders returned error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("len(orders) = %d, want 2", len(orders))
	}
	if got := gotQuery.Get("modified_after"); got != "2026-03-29T00:30:00Z" {
		t.Fatalf("modified_after = %q, want 2026-03-29T00:30:00Z", got)
	}
	if got := gotQuery.Get("dates_are_gmt"); got != "true" {
		t.Fatalf("dates_are_gmt = %q, want true", got)
	}
	if got := gotQuery.Get("orderby"); got != "modified" {
		t.Fatalf("orderby = %q, want modified", got)
	}
	if cursor != "2026-03-30T01:15:00Z" {
		t.Fatalf("cursor = %q, want 2026-03-30T01:15:00Z", cursor)
	}
}
```

- [ ] **Step 2: Run test to verify RED**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/integration/woocommerce -run TestPollOrdersUsesGMTModifiedCursorAndModifiedOrdering -count=1
```

Expected: fail because the current provider sends `orderby=date`, omits `dates_are_gmt`, ignores `date_modified_gmt`, and returns a local timestamp cursor.

## Task 2: SDK Params And Response Field

**Files:**
- Modify: `packages/woocommerce-go-sdk/orders.go`

- [ ] **Step 1: Add SDK fields**

In `OrderListParams`, add:

```go
DatesAreGMT bool // Interpret after/before/modified date filters as GMT.
```

In `WooOrder`, add:

```go
DateModifiedGMT string `json:"date_modified_gmt"`
```

Update the `OrderBy` comment to include `modified`.

- [ ] **Step 2: Emit `dates_are_gmt`**

In `OrderService.List`, after `modified_after`, add:

```go
if params.DatesAreGMT {
	v.Set("dates_are_gmt", "true")
}
```

## Task 3: Provider Cursor Logic

**Files:**
- Modify: `apps/api-server/internal/integration/woocommerce/provider.go`

- [ ] **Step 1: Request modified ordering**

Change the list params to:

```go
params := woocommercesdk.OrderListParams{
	PerPage: 50,
	OrderBy: "modified",
	Order:   "asc",
}
```

When `cursor` includes an explicit timezone, set:

```go
params.ModifiedAfter = cursor
params.DatesAreGMT = true
```

When `cursor` has no explicit timezone, keep:

```go
params.ModifiedAfter = cursor
```

This avoids treating existing local-time cursors as GMT and accidentally skipping orders on the first rollout poll.

- [ ] **Step 2: Add cursor helpers**

Add helpers near `PollOrders`:

```go
func wooCursorHasExplicitZone(cursor string) bool {
	if cursor == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, cursor)
	return err == nil
}

func wooOrderModifiedCursor(o woocommercesdk.WooOrder) (time.Time, bool) {
	if o.DateModifiedGMT != "" {
		if t, ok := parseWooModifiedTime(o.DateModifiedGMT); ok {
			return t, true
		}
	}
	if o.DateModified != "" {
		return parseWooModifiedTime(o.DateModified)
	}
	return time.Time{}, false
}

func parseWooModifiedTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), true
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", value, time.UTC); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}
```

- [ ] **Step 3: Replace string comparison**

Inside the order loop, replace the raw string comparison with:

```go
var maxModified *time.Time
if wooCursorHasExplicitZone(cursor) {
	if t, ok := parseWooModifiedTime(cursor); ok {
		maxModified = &t
	}
}

// inside loop, after mapping succeeds
if modifiedAt, ok := wooOrderModifiedCursor(wco); ok {
	if maxModified == nil || modifiedAt.After(*maxModified) {
		t := modifiedAt
		maxModified = &t
		newCursor = modifiedAt.UTC().Format(time.RFC3339)
	}
}
```

## Task 4: Validation And PR

**Files:**
- Verify all touched files.

Assume every command below starts from the shared workspace root `/Users/rafs/praca/openoms-dev`.

- [ ] **Step 1: Targeted tests**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/apps/api-server
go test ./internal/integration/woocommerce -count=1
```

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public/packages/woocommerce-go-sdk
go test ./... -count=1
```

- [ ] **Step 2: Self-review**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
git diff --stat
git diff
```

Check that no unrelated untracked docs from 2026-05-17 are staged.

- [ ] **Step 3: Full local CI**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

- [ ] **Step 4: Commit, push, PR, review gate**

Commit:

```bash
git add apps/api-server/internal/integration/woocommerce/provider.go apps/api-server/internal/integration/woocommerce/provider_test.go packages/woocommerce-go-sdk/orders.go docs/superpowers/plans/2026-05-18-ope-331-woocommerce-cursor.md
git commit -m "OPE-331: fix WooCommerce modified cursor"
```

Push and create PR with title:

```text
OPE-331: fix WooCommerce modified cursor
```

Read CodeRabbit comments/review threads before merge. Merge only after PR checks, CodeQL, release, SBOM import and enterprise deploy are green.

## Risk And Rollback

- Risk: legacy WooCommerce sync cursors without timezone are ambiguous. Mitigation: do not set `dates_are_gmt=true` for no-zone cursors; after the first successful poll, store a UTC RFC3339 cursor.
- Risk: WooCommerce stores without `date_modified_gmt` would fall back to `date_modified`. Mitigation: parse fallback as UTC only to avoid string ordering bugs; this is no worse than the current local-time cursor and becomes deterministic.
- Rollback: revert the PR. Existing cursors are plain strings; UTC RFC3339 cursors remain acceptable ISO8601 values for WooCommerce `modified_after`.

## Docs

- No public API, DB schema, user-facing workflow, or deployment behavior changes.
- PR docs section should mark context docs as N/A and include this plan as the work log.
