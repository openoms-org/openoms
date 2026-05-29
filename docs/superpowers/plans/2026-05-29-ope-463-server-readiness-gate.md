# OPE-463 — Server-side Feature Readiness Gating — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce the feature-readiness registry server-side so that, in `client-ready` surface mode, only `ready` features (routes + providers) are reachable over the API — closing the gap where staged features are callable directly.

**Architecture:** A canonical readiness JSON (feature-id → state + dashboard routes + `/v1` endpoint prefixes) is the single source of truth. Go embeds it and exposes `IsFeatureEnabled`/`IsProviderEnabled`; a `RequireFeature(featureID)` chi middleware gates non-ready route groups (mirroring `requirePermission`); integration/shipment create validate the provider value. The dashboard derives its existing maps from the same JSON. A bidirectional CI test guarantees no endpoint is silently left ungated.

**Tech Stack:** Go 1.25 (chi/v5, caarlos0/env/v11, testify), Next.js/TS, JSON.

**Worktree:** `/Users/rafs/praca/openoms-dev/public-ope-463-server-readiness` (branch `fix/OPE-463-server-readiness`). All paths below are relative to it unless absolute.

**Spec:** `docs/superpowers/specs/2026-05-29-ope-463-server-readiness-gate-design.md`

---

## File structure

- Create `apps/api-server/internal/readiness/readiness.json` — canonical registry (Go-embeddable; see Task 1 for the dashboard-sharing decision).
- Create `apps/api-server/internal/readiness/readiness.go` — embed + parse + `IsFeatureEnabled`/`IsProviderEnabled`.
- Create `apps/api-server/internal/readiness/readiness_test.go` — unit tests for the lookup logic.
- Create `apps/api-server/internal/middleware/feature_gate.go` — `RequireFeature` factory.
- Create `apps/api-server/internal/middleware/feature_gate_test.go`.
- Modify `apps/api-server/internal/config/config.go` — add `APISurfaceMode`.
- Modify `apps/api-server/internal/router/router.go` — add `requireFeature` alias + `r.Use(requireFeature("..."))` per non-ready group; provider validation is in handlers.
- Modify `apps/api-server/internal/handler/integration_handler.go` (Create) and `shipment_handler.go` (Create, CreateForOrder) — provider-value validation.
- Create `apps/api-server/internal/router/readiness_gate_test.go` — route-gate behavior.
- Create `apps/api-server/internal/router/readiness_coverage_test.go` — drift guard.
- Modify `apps/dashboard/src/lib/readiness.ts` — derive `NAV_ROUTE_READINESS` + `PROVIDER_READINESS` from the JSON.
- Modify/Create `apps/dashboard/src/lib/__tests__/readiness.test.ts` — derivation regression lock (existing file already pins lists).

---

## Task 1: Resolve canonical-file placement (R1) + create the registry JSON

**Files:**
- Read: `apps/api-server/Dockerfile`, `apps/dashboard/Dockerfile` (determine build contexts)
- Create: `apps/api-server/internal/readiness/readiness.json`

- [ ] **Step 1: Determine build contexts.** Run:
```bash
cd /Users/rafs/praca/openoms-dev/public-ope-463-server-readiness
grep -rnE 'context:|COPY ' apps/api-server/Dockerfile apps/dashboard/Dockerfile .github/workflows/*.yml 2>/dev/null | grep -iE 'context|COPY (\.|apps)' | head
```
Decision:
- If BOTH images build from the **repo root** context → canonical = `apps/api-server/internal/readiness/readiness.json`; the dashboard imports it via a relative path (Task 9, option A).
- If images build from **per-app** contexts (`apps/api-server`, `apps/dashboard`) → canonical still lives at `apps/api-server/internal/readiness/readiness.json`; Task 9 adds a committed copy `apps/dashboard/src/lib/readiness.generated.json` produced by a sync script + a CI freshness test (option B). Go always embeds the api-server copy directly.

Record the chosen option in a comment at the top of `readiness.json` ("canonical source; dashboard sync: option A/B").

- [ ] **Step 2: Author the canonical JSON.** Build the content from the current `apps/dashboard/src/lib/readiness.ts` (`NAV_ROUTE_READINESS`, `PROVIDER_READINESS`) plus `/v1` endpoint prefixes. Shape:
```jsonc
{
  "_comment": "OPE-463 canonical feature-readiness registry. Single source of truth for dashboard + API gating. Dashboard sync: option <A|B>.",
  "providers": {
    "allegro": "ready", "inpost": "ready",
    "olx": "controlled", "dhl": "controlled", "dpd": "controlled", "gls": "controlled",
    "fakturownia": "controlled", "btp": "controlled",
    "amazon": "beta", "ebay": "beta", "woocommerce": "beta", "erli": "beta",
    "ups": "beta", "fedex": "beta", "poczta_polska": "beta", "orlen_paczka": "beta",
    "wfirma": "beta", "infakt": "beta",
    "kaufland": "blocked", "empik": "blocked", "mirakl": "blocked",
    "shopify": "blocked", "prestashop": "blocked", "shoper": "blocked"
  },
  "features": {
    "invoicing":          { "state": "controlled", "routes": ["/invoices", "/invoicing"], "endpoints": ["/v1/invoices"] },
    "warehouses":         { "state": "controlled", "routes": ["/settings/warehouses"], "endpoints": ["/v1/warehouses"] },
    "warehouse_documents":{ "state": "verify", "routes": ["/settings/warehouse-documents"], "endpoints": ["/v1/warehouse-documents"] },
    "stocktakes":         { "state": "verify", "routes": ["/stocktakes"], "endpoints": ["/v1/stocktakes"] },
    "pick_pack":          { "state": "verify", "routes": ["/pick-pack", "/packing"], "endpoints": ["/v1/pick-pack/sessions"] },
    "purchase_orders":    { "state": "verify", "routes": ["/purchase-orders"], "endpoints": ["/v1/purchase-orders"] },
    "suppliers":          { "state": "controlled", "routes": ["/suppliers", "/settings/feeds"], "endpoints": ["/v1/suppliers"] },
    "dropship":           { "state": "beta", "routes": ["/dropship-orders"], "endpoints": ["/v1/dropship-orders"] },
    "loyalty":            { "state": "beta", "routes": ["/loyalty"], "endpoints": ["/v1/loyalty"] },
    "recurring_orders":   { "state": "beta", "routes": ["/recurring-orders"], "endpoints": ["/v1/recurring-orders"] },
    "repricing":          { "state": "beta", "routes": ["/repricing"], "endpoints": ["/v1/repricing"] },
    "reconciliation":     { "state": "beta", "routes": ["/reconciliation"], "endpoints": ["/v1/reconciliation"] },
    "stock_sync":         { "state": "beta", "routes": ["/stock-sync"], "endpoints": ["/v1/stock-sync"] },
    "listing_sync":       { "state": "beta", "routes": ["/listing-sync"], "endpoints": ["/v1/listing-sync"] },
    "segments":           { "state": "beta", "routes": ["/customers/segments"], "endpoints": ["/v1/segments"] },
    "forecast":           { "state": "beta", "routes": ["/forecast"], "endpoints": ["/v1/forecast"] },
    "carbon":             { "state": "beta", "routes": ["/carbon"], "endpoints": ["/v1/carbon"] },
    "vat_oss":            { "state": "beta", "routes": ["/vat-oss"], "endpoints": ["/v1/vat-oss"] },
    "workflows":          { "state": "beta", "routes": ["/workflows"], "endpoints": ["/v1/workflows"] },
    "ai":                 { "state": "beta", "routes": ["/tools"], "endpoints": ["/v1/ai", "/v1/images/remove-background"] },
    "price_lists":        { "state": "verify", "routes": ["/settings/price-lists"], "endpoints": ["/v1/price-lists"] },
    "exchange_rates":     { "state": "verify", "routes": ["/settings/currencies"], "endpoints": ["/v1/exchange-rates"] },
    "price_lists_b2b":    { "state": "verify", "routes": ["/settings/price-lists"], "endpoints": ["/v1/price-lists"] },
    "marketing":          { "state": "blocked", "routes": ["/settings/marketing"], "endpoints": ["/v1/marketing"] }
    // Complete this map in Task 6 using the cross-reference grep; every non-ready /v1 r.Route group MUST have an entry.
  }
}
```
> NOTE: `audit` stays `ready` (registry source of truth) → no entry needed (only non-`ready` features are listed/gated). Settings sub-routes that are `controlled` (custom-fields, order-statuses, accounting, message-templates, print-templates, inventory, webhooks, automation/rules, sync-jobs) get entries in Task 6 once their exact `/v1` prefixes are confirmed.

- [ ] **Step 3: Commit.**
```bash
git add apps/api-server/internal/readiness/readiness.json
git commit -m "OPE-463: add canonical feature-readiness registry"
```

---

## Task 2: Go readiness package (embed + lookup) — TDD

**Files:**
- Create: `apps/api-server/internal/readiness/readiness.go`
- Test: `apps/api-server/internal/readiness/readiness_test.go`

- [ ] **Step 1: Write the failing test.**
```go
package readiness

import "testing"

func TestIsFeatureEnabled(t *testing.T) {
	cases := []struct {
		feature, mode string
		want          bool
	}{
		{"repricing", "client-ready", false}, // beta blocked in client-ready
		{"repricing", "full", true},          // beta allowed in full
		{"marketing", "full", false},         // blocked never allowed
		{"unknown_feature", "client-ready", false}, // unknown -> treat as not-ready
		{"unknown_feature", "full", true},    // unknown -> non-blocked, allowed in full
	}
	for _, c := range cases {
		if got := IsFeatureEnabled(c.feature, c.mode); got != c.want {
			t.Errorf("IsFeatureEnabled(%q,%q)=%v want %v", c.feature, c.mode, got, c.want)
		}
	}
}

func TestIsProviderEnabled(t *testing.T) {
	if !IsProviderEnabled("allegro", "client-ready") {
		t.Error("allegro should be enabled in client-ready")
	}
	if IsProviderEnabled("amazon", "client-ready") {
		t.Error("amazon (beta) must be blocked in client-ready")
	}
	if IsProviderEnabled("shopify", "full") {
		t.Error("shopify (blocked) must never be enabled")
	}
}
```

- [ ] **Step 2: Run — expect FAIL (undefined).**
```bash
cd apps/api-server && go test ./internal/readiness/ -run TestIs -count=1
```
Expected: build error `undefined: IsFeatureEnabled`.

- [ ] **Step 3: Implement.**
```go
// Package readiness is the embedded single source of truth for feature-readiness gating.
package readiness

import (
	_ "embed"
	"encoding/json"
)

//go:embed readiness.json
var registryJSON []byte

type State string

const (
	Ready      State = "ready"
	Controlled State = "controlled"
	Verify     State = "verify"
	Beta       State = "beta"
	Blocked    State = "blocked"
)

type Feature struct {
	State     State    `json:"state"`
	Routes    []string `json:"routes"`
	Endpoints []string `json:"endpoints"`
}

type registry struct {
	Providers map[string]State   `json:"providers"`
	Features  map[string]Feature `json:"features"`
}

var reg registry

func init() {
	if err := json.Unmarshal(registryJSON, &reg); err != nil {
		panic("readiness: invalid embedded readiness.json: " + err.Error())
	}
}

// isVisible mirrors dashboard readiness.ts isReadinessVisible: blocked is never
// visible; "full" allows any non-blocked state; "client-ready" allows only "ready".
func isVisible(s State, mode string) bool {
	if s == Blocked {
		return false
	}
	if mode == "full" {
		return true
	}
	return s == Ready
}

// IsFeatureEnabled reports whether a feature is reachable under the surface mode.
// Unknown feature ids are treated as non-ready (state "verify"), matching the
// frontend getRouteReadiness fallback.
func IsFeatureEnabled(featureID, mode string) bool {
	f, ok := reg.Features[featureID]
	if !ok {
		return isVisible(Verify, mode)
	}
	return isVisible(f.State, mode)
}

// IsProviderEnabled reports whether a provider key is selectable under the mode.
// Unknown providers are treated as non-ready.
func IsProviderEnabled(providerKey, mode string) bool {
	s, ok := reg.Providers[providerKey]
	if !ok {
		return isVisible(Verify, mode)
	}
	return isVisible(s, mode)
}

// FeatureEndpoints exposes the endpoint prefixes per non-ready feature for the
// coverage drift guard.
func NonReadyFeatures() map[string]Feature {
	out := map[string]Feature{}
	for id, f := range reg.Features {
		if f.State != Ready {
			out[id] = f
		}
	}
	return out
}
```

- [ ] **Step 4: Run — expect PASS.**
```bash
cd apps/api-server && go test ./internal/readiness/ -count=1
```

- [ ] **Step 5: Commit.**
```bash
git add apps/api-server/internal/readiness/
git commit -m "OPE-463: embed readiness registry with feature/provider lookup"
```

---

## Task 3: Config — `APISurfaceMode`

**Files:**
- Modify: `apps/api-server/internal/config/config.go:19-24` (add field near other surface/URL fields)

- [ ] **Step 1: Add the field.** After the `FrontendURL` field (line ~22), add:
```go
	// APISurfaceMode gates non-ready features over the API. "client-ready" (default)
	// exposes only "ready" features; "full" exposes all except "blocked". Keep this in
	// sync with the dashboard's NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE.
	APISurfaceMode string `env:"OPENOMS_API_SURFACE" envDefault:"client-ready"`
```

- [ ] **Step 2: Verify it parses.**
```bash
cd apps/api-server && go build ./internal/config/ && echo OK
```
Expected: `OK`.

- [ ] **Step 3: Commit.**
```bash
git add apps/api-server/internal/config/config.go
git commit -m "OPE-463: add OPENOMS_API_SURFACE config (default client-ready)"
```

---

## Task 4: `RequireFeature` middleware — TDD

**Files:**
- Create: `apps/api-server/internal/middleware/feature_gate.go`
- Test: `apps/api-server/internal/middleware/feature_gate_test.go`

- [ ] **Step 1: Failing test.**
```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireFeature_BlocksNonReadyInClientReady(t *testing.T) {
	h := RequireFeature("repricing", "client-ready")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/repricing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
}

func TestRequireFeature_AllowsInFullMode(t *testing.T) {
	called := false
	h := RequireFeature("repricing", "full")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/repricing", nil))
	if !called {
		t.Fatal("handler should be reached in full mode")
	}
}
```

- [ ] **Step 2: Run — expect FAIL (undefined RequireFeature).**
```bash
cd apps/api-server && go test ./internal/middleware/ -run TestRequireFeature -count=1
```

- [ ] **Step 3: Implement.**
```go
package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/openoms-org/openoms/apps/api-server/internal/readiness"
)

// RequireFeature gates a route group by feature readiness. When the feature is not
// enabled for the active surface mode it returns 404 with a JSON body, hiding the
// capability from clients (mirrors the dashboard readiness route guard).
func RequireFeature(featureID, surfaceMode string) func(http.Handler) http.Handler {
	enabled := readiness.IsFeatureEnabled(featureID, surfaceMode)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "feature_not_available"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```
> The enabled value is computed once at construction (wire-up time) — surface mode is process-global, so this is correct and avoids per-request work.

- [ ] **Step 4: Run — expect PASS.**
```bash
cd apps/api-server && go test ./internal/middleware/ -run TestRequireFeature -count=1
```

- [ ] **Step 5: Commit.**
```bash
git add apps/api-server/internal/middleware/feature_gate.go apps/api-server/internal/middleware/feature_gate_test.go
git commit -m "OPE-463: add RequireFeature readiness gate middleware"
```

---

## Task 5: Router alias for `requireFeature`

**Files:**
- Modify: `apps/api-server/internal/router/router.go:127` (next to `requirePermission := middleware.RequirePermission`)

- [ ] **Step 1: Add the alias** right after line 127:
```go
	requirePermission := middleware.RequirePermission
	requireFeature := func(featureID string) func(http.Handler) http.Handler {
		return middleware.RequireFeature(featureID, deps.Config.APISurfaceMode)
	}
```

- [ ] **Step 2: Build (alias unused yet is fine only after Task 6; to keep this commit green, also do Task 6 before building/committing). Skip standalone build here.**

(No separate commit — folded into Task 6.)

---

## Task 6: Wire `requireFeature` on every non-ready route group + complete the JSON

**Files:**
- Modify: `apps/api-server/internal/router/router.go` (one line per group)
- Modify: `apps/api-server/internal/readiness/readiness.json` (complete `features` map)

- [ ] **Step 1: Enumerate the gap.** Cross-reference frontend non-ready routes with backend groups:
```bash
cd apps/api-server
grep -nE 'r\.Route\("/' internal/router/router.go   # all groups + line numbers
# compare against readiness.json features[].endpoints and the dashboard NAV_ROUTE_READINESS
```
For every `r.Route("/<group>", ...)` whose feature is non-`ready`, ensure a `features` entry exists in `readiness.json` and add the gate.

- [ ] **Step 2: Add the gate line** as the FIRST `r.Use` inside each non-ready group. Known groups (line numbers from current HEAD; re-confirm before editing):
```
invoices (466)            -> r.Use(requireFeature("invoicing"))
suppliers (763)           -> r.Use(requireFeature("suppliers"))
purchase-orders (819)     -> r.Use(requireFeature("purchase_orders"))
dropship-orders (831)     -> r.Use(requireFeature("dropship"))
warehouses (846)          -> r.Use(requireFeature("warehouses"))
loyalty (887)             -> r.Use(requireFeature("loyalty"))
recurring-orders (903)    -> r.Use(requireFeature("recurring_orders"))
price-lists (988)         -> r.Use(requireFeature("price_lists"))
stocktakes (1024)         -> r.Use(requireFeature("stocktakes"))
pick-pack/sessions (1087) -> r.Use(requireFeature("pick_pack"))  // BEFORE the existing requirePermission line
repricing (1103)          -> r.Use(requireFeature("repricing"))
reconciliation (1153)     -> r.Use(requireFeature("reconciliation"))
```
Plus the groups discovered in Step 1 (warehouse-documents, stock-sync, listing-sync, segments, forecast, carbon, vat-oss, workflows, marketing, ai/images, exchange-rates, products listings/variants sub-routes, the controlled settings sub-routes). Each gets a matching `features` entry.

Example (pick-pack, layered with the existing permission gate):
```go
			r.Route("/pick-pack/sessions", func(r chi.Router) {
				r.Use(requireFeature("pick_pack"))
				r.Use(requirePermission(model.PermWarehousesManage))
				// ...existing handlers unchanged...
			})
```

- [ ] **Step 3: Build.**
```bash
cd apps/api-server && go build ./... && echo OK
```
Expected: `OK` (confirms `requireFeature` is used and compiles).

- [ ] **Step 4: Commit.**
```bash
git add apps/api-server/internal/router/router.go apps/api-server/internal/readiness/readiness.json
git commit -m "OPE-463: gate non-ready route groups with RequireFeature"
```

---

## Task 7: Route-gate behavior test — TDD

**Files:**
- Create: `apps/api-server/internal/router/readiness_gate_test.go`

- [ ] **Step 1: Write the test** (mirrors `newPickPackPermissionRequest`; builds the router with a chosen `APISurfaceMode` and an owner token so only the feature gate decides):
```go
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/openoms-org/openoms/apps/api-server/internal/config"
	"github.com/openoms-org/openoms/apps/api-server/internal/model"
	"github.com/openoms-org/openoms/apps/api-server/internal/service"
)

func readinessRequest(t *testing.T, mode, method, path string) (http.Handler, *http.Request) {
	t.Helper()
	user := model.User{ID: uuid.New(), TenantID: uuid.New(), Email: "owner@example.com", Role: "owner"}
	tokenSvc, err := service.NewTokenService("test-jwt-secret-for-readiness-gate")
	require.NoError(t, err)
	token, err := tokenSvc.GenerateAccessToken(user)
	require.NoError(t, err)
	r := New(RouterDeps{
		Config:   &config.Config{Env: "development", FrontendURL: "http://localhost:3000", UploadDir: t.TempDir(), APISurfaceMode: mode},
		TokenSvc: tokenSvc,
	})
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return r, req
}

func TestReadinessGate_ClientReadyBlocksNonReady(t *testing.T) {
	for _, p := range []string{"/v1/repricing", "/v1/reconciliation", "/v1/loyalty"} {
		t.Run(p, func(t *testing.T) {
			r, req := readinessRequest(t, "client-ready", http.MethodGet, p)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			require.Equal(t, http.StatusNotFound, rr.Code)
		})
	}
}

func TestReadinessGate_ExcludedRoutesPassInClientReady(t *testing.T) {
	// /v1/stats is shared by the ready home dashboard -> must NOT be gated.
	r, req := readinessRequest(t, "client-ready", http.MethodGet, "/v1/stats/dashboard")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusNotFound, rr.Code) // reaches handler (may 4xx/5xx on nil deps, but not the 404 gate)
}

func TestReadinessGate_FullModeAllowsNonReady(t *testing.T) {
	r, req := readinessRequest(t, "full", http.MethodGet, "/v1/repricing")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.NotEqual(t, http.StatusNotFound, rr.Code)
}
```
> If a gated handler with `nil` deps panics before the gate in `full` mode, prefer asserting the gate via the 404 body in client-ready and, for full mode, assert the response is not the `feature_not_available` 404 (decode body). Adjust paths to groups whose handlers tolerate nil (GET list endpoints) to keep the test about the gate, not the handler.

- [ ] **Step 2: Run — expect the client-ready cases to PASS once Task 6 is wired; full-mode/excluded may need path tweaks.**
```bash
cd apps/api-server && go test ./internal/router/ -run TestReadinessGate -count=1 -v
```

- [ ] **Step 3: Commit.**
```bash
git add apps/api-server/internal/router/readiness_gate_test.go
git commit -m "OPE-463: test readiness route gate behavior"
```

---

## Task 8: Drift-guard coverage test — TDD

**Files:**
- Create: `apps/api-server/internal/router/readiness_coverage_test.go`

- [ ] **Step 1: Write the guard.** Parse `router.go` source for `requireFeature("X")` calls; cross-check against `readiness.json`:
```go
package router

import (
	"os"
	"regexp"
	"testing"

	"github.com/openoms-org/openoms/apps/api-server/internal/readiness"
)

// excludedEndpoints are non-ready-adjacent endpoints intentionally left ungated
// (public/token-auth or shared with ready surfaces). Reviewed allowlist.
var excludedEndpoints = map[string]bool{
	"/v1/stats": true, "/v1/barcode": true,
}

func TestReadinessCoverage_EveryGateHasFeature(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil { t.Fatal(err) }
	re := regexp.MustCompile(`requireFeature\("([a-z_]+)"\)`)
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		id := m[1]
		if readiness.IsFeatureEnabled(id, "full") == false && readiness.IsFeatureEnabled(id, "client-ready") == false {
			// blocked is the only state false in BOTH modes; that's allowed.
		}
		if _, ok := readiness.LookupFeature(id); !ok {
			t.Errorf("requireFeature(%q) has no entry in readiness.json", id)
		}
	}
}

func TestReadinessCoverage_EveryNonReadyFeatureIsGated(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil { t.Fatal(err) }
	body := string(src)
	for id := range readiness.NonReadyFeatures() {
		if !regexpQuote(body, id) {
			t.Errorf("non-ready feature %q in readiness.json is not gated by any requireFeature(...) in router.go", id)
		}
	}
}

func regexpQuote(body, id string) bool {
	return regexp.MustCompile(`requireFeature\("` + regexp.QuoteMeta(id) + `"\)`).MatchString(body)
}
```
- [ ] **Step 2: Add `LookupFeature` to the readiness package** (`readiness.go`):
```go
// LookupFeature returns the feature entry and whether it exists.
func LookupFeature(id string) (Feature, bool) { f, ok := reg.Features[id]; return f, ok }
```

- [ ] **Step 3: Run — expect PASS (fix any uncovered feature by gating it or moving its endpoint to excludedEndpoints with review).**
```bash
cd apps/api-server && go test ./internal/router/ -run TestReadinessCoverage -count=1 -v
```

- [ ] **Step 4: Commit.**
```bash
git add apps/api-server/internal/router/readiness_coverage_test.go apps/api-server/internal/readiness/readiness.go
git commit -m "OPE-463: add bidirectional readiness drift-guard test"
```

---

## Task 9: Dashboard derives maps from the canonical JSON

**Files:**
- Modify: `apps/dashboard/src/lib/readiness.ts`
- Test: `apps/dashboard/src/lib/__tests__/readiness.test.ts`

- [ ] **Step 1: Import the canonical JSON** per the Task-1 decision:
  - Option A (repo-root build context): `import registry from "../../../api-server/internal/readiness/readiness.json";`
  - Option B (per-app context): add a sync script `apps/dashboard/scripts/sync-readiness.mjs` copying the api-server JSON to `apps/dashboard/src/lib/readiness.generated.json`, a `prebuild`/`pretest` npm hook running it, a CI freshness check (`git diff --exit-code` after sync), then `import registry from "./readiness.generated.json";`.

- [ ] **Step 2: Derive the existing maps** (replace the hardcoded `NAV_ROUTE_READINESS` / `PROVIDER_READINESS` literals; keep all exported function signatures and `FeatureReadiness`/`DashboardSurfaceMode` types unchanged):
```ts
type Registry = {
  providers: Record<string, FeatureReadiness>;
  features: Record<string, { state: FeatureReadiness; routes: string[]; endpoints: string[] }>;
};
const reg = registry as Registry;

// Derive route->state. Routes not present default to 'ready' here ONLY for routes the
// app ships as ready; getRouteReadiness keeps its 'verify' fallback for unknown routes.
const NAV_ROUTE_READINESS: Record<string, FeatureReadiness> = (() => {
  const m: Record<string, FeatureReadiness> = {};
  for (const f of Object.values(reg.features)) for (const route of f.routes) m[route] = f.state;
  return m;
})();
const PROVIDER_READINESS: Record<string, FeatureReadiness> = reg.providers;
```
> The current `readiness.ts` also lists `ready` routes explicitly; preserve those by either (a) adding `ready` features to the JSON, or (b) keeping a small `READY_ROUTES` constant for routes that must resolve to `ready` (and are not gated server-side). Decide based on `readiness.test.ts` expectations — keep that test green.

- [ ] **Step 3: Run the regression lock.**
```bash
cd apps/dashboard && npx vitest run src/lib/__tests__/readiness.test.ts --reporter=dot
```
Expected: PASS (the test pins concrete provider/route lists; the derivation must reproduce them). If it fails, the JSON is missing entries — add them, do NOT change the test expectations.

- [ ] **Step 4: Commit.**
```bash
git add apps/dashboard/src/lib/readiness.ts apps/dashboard/src/lib/__tests__/readiness.test.ts apps/dashboard/scripts/ apps/dashboard/package.json 2>/dev/null
git commit -m "OPE-463: derive dashboard readiness maps from canonical registry"
```

---

## Task 10: Provider-value validation on create — TDD

**Files:**
- Modify: `apps/api-server/internal/handler/integration_handler.go:74` (Create)
- Modify: `apps/api-server/internal/handler/shipment_handler.go:155` (Create) and `:53` (CreateForOrder)
- Test: `apps/api-server/internal/handler/integration_provider_gate_test.go`

- [ ] **Step 1: Failing test** (integration create rejects a non-ready provider in client-ready):
```go
// integration_provider_gate_test.go — table test: POST /v1/integrations with
// provider "amazon" under client-ready -> 422 {"error":"provider_not_available"};
// provider "allegro" -> passes provider gate (reaches service/validation).
```
Write it against the handler with a stub service, asserting the 422 short-circuit happens before the service is called for a non-ready provider. Follow the handler test pattern in `internal/handler/*_test.go`.

- [ ] **Step 2: Run — expect FAIL.**
```bash
cd apps/api-server && go test ./internal/handler/ -run ProviderGate -count=1
```

- [ ] **Step 3: Implement.** In `IntegrationHandler.Create`, right after decoding `req` and before the limit injection:
```go
	mode := h.surfaceMode // injected at construction from config.APISurfaceMode
	if !readiness.IsProviderEnabled(req.Provider, mode) {
		writeError(w, http.StatusUnprocessableEntity, "provider_not_available")
		return
	}
```
Add a `surfaceMode string` field to `IntegrationHandler` + constructor param (and pass `deps.Config.APISurfaceMode` at wire-up in main.go/router). In `ShipmentHandler.Create` and `CreateForOrder`, after decoding, validate `req.Provider` the same way, allowing `"manual"` unconditionally:
```go
	if req.Provider != "manual" && !readiness.IsProviderEnabled(req.Provider, h.surfaceMode) {
		writeError(w, http.StatusUnprocessableEntity, "provider_not_available")
		return
	}
```

- [ ] **Step 4: Run — expect PASS.**
```bash
cd apps/api-server && go test ./internal/handler/ -run ProviderGate -count=1
```

- [ ] **Step 5: Commit.**
```bash
git add apps/api-server/internal/handler/ apps/api-server/cmd/server/main.go
git commit -m "OPE-463: validate provider readiness on integration/shipment create"
```

---

## Task 11: Docs + Helm dependency + full validation

**Files:**
- Modify: `apps/api-server/docs/openapi.yaml` (note the 404 feature_not_available / 422 provider_not_available responses where relevant) — minimal.
- Modify: `.claude/context/API_CONTRACTS.md` + `.claude/context/SECURITY_POSTURE.md` (local) — record the readiness gate.
- Modify: `docs/system-documentation.md` — one line under security/architecture (tracked → part of PR).

- [ ] **Step 1: Document** the gate (env `OPENOMS_API_SURFACE`, 404/422 contract) in the above.
- [ ] **Step 2: Note the deploy dependency** in the PR description: enterprise `values-production.yaml`/`values-staging.yaml` must set `OPENOMS_API_SURFACE` to match `NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE` (a matching enterprise PR/checklist item).
- [ ] **Step 3: Full local CI.**
```bash
cd /Users/rafs/praca/openoms-dev/public-ope-463-server-readiness && ./scripts/local-ci.sh
```
Expected: all checks pass (gofmt, vet, golangci-lint, eslint, next build, go test incl. the new readiness + gate + coverage tests).
- [ ] **Step 4: Commit docs.**
```bash
git add docs/ .claude/ apps/api-server/docs/openapi.yaml
git commit -m "OPE-463: document server-side readiness gating"
```

---

## After implementation (Lane L pipeline)
Self-review → **adversarial review workflow** (correctness/security/integrity, esp. that no ready surface depends on a gated endpoint) → live verification (`OPENOMS_API_SURFACE=client-ready` a gated endpoint returns 404; `full` reaches it) → push → PR (Docs updated) → **GATE 2 (your approval)** → review-bot gate.

## Notes / decisions baked in
- 404 for route gate, 422 for provider value. Binary surface mode. Global (not per-tenant). `audit` stays `ready` (ungated). `ai` = `beta` (gated). Exclusions: `/v1/stats`, `/v1/barcode`, all public/token-auth (`/v1/webhooks`, `/v1/public`, `/v1/billing`, `/v1/config/public`, `/v1/feeds`, `/v1/supplier-portal`) — never gated.
- Escape hatch: `OPENOMS_API_SURFACE=full` disables gating with no code change.
