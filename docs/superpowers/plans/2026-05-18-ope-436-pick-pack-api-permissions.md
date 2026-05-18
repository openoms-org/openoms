# OPE-436 Pick Pack API Permissions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Require an explicit warehouse permission before any Pick & Pack API session route can be used.

**Architecture:** Keep the existing router-level RBAC pattern and guard `/v1/pick-pack/sessions` with `middleware.RequirePermission(model.PermWarehousesManage)`. This is intentionally conservative because Pick & Pack is still hidden from `client-ready` and changes operational fulfillment state. A later task can split this into dedicated Pick & Pack read/write permissions if the product needs non-admin warehouse operators.

**Tech Stack:** Go 1.25, chi router, existing JWT/RBAC middleware, httptest router regression tests.

---

## Scope

In scope:

- Add route-level RBAC to existing Pick & Pack session API routes.
- Add router regression tests proving users without `warehouses.manage` receive `403`.
- Add tests proving users with `warehouses.manage` pass the permission middleware and reach handler validation.
- Update security posture context.

Out of scope:

- No new RBAC permission constants.
- No database migrations.
- No dashboard UI exposure changes.
- No OPE-403 child work.

## Permission Decision

Use `model.PermWarehousesManage` for the entire Pick & Pack session route group.

Reason:

- Pick & Pack is operationally closer to warehouse/stocktake flows than generic order viewing.
- Existing warehouse documents and stocktakes already use `warehouses.manage`.
- `orders.edit` is included in the default member role, so it would not close the “any authenticated operational user” gap enough before module certification.
- A future dedicated permission can be added once Pick & Pack is ready for broader operator roles.

## Files

- Modify: `apps/api-server/internal/router/router.go`
  - Add `r.Use(requirePermission(model.PermWarehousesManage))` inside `/pick-pack/sessions`.
  - Update the comment from “any authenticated user” to the chosen warehouse permission.
- Create: `apps/api-server/internal/router/pick_pack_permissions_test.go`
  - Build the router with a real test `service.TokenService` and `handler.NewPickPackHandler(nil)`.
  - Use invalid request bodies or invalid UUIDs to prove allowed requests reach handler validation without needing a real service.
  - Use denied requests to prove the permission middleware blocks before handler validation.
- Modify: `.claude/context/SECURITY_POSTURE.md`
  - Add OPE-436 to recent security updates.

## Task 1: Add Router Permission Regression Tests

- [x] Create `apps/api-server/internal/router/pick_pack_permissions_test.go`.
- [x] Add test helpers:

```go
func newPickPackPermissionRequest(t *testing.T, method, path, body string, permissions []string) (http.Handler, *http.Request) {
	t.Helper()
	userID := uuid.New()
	tenantID := uuid.New()
	tokenSvc, err := service.NewTokenService("test-jwt-secret-for-pick-pack-rbac")
	require.NoError(t, err)
	token, err := tokenSvc.GenerateAccessToken(model.User{
		ID:          userID,
		TenantID:    tenantID,
		Email:       "operator@example.com",
		Role:        "member",
		Permissions: permissions,
	})
	require.NoError(t, err)

	router := New(RouterDeps{
		Config: &config.Config{
			Env:         "development",
			FrontendURL: "http://localhost:3000",
			UploadDir:   t.TempDir(),
		},
		TokenSvc: tokenSvc,
		PickPack: handler.NewPickPackHandler(nil),
	})

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	return router, req
}
```

- [x] Add denied table coverage for every Pick & Pack route:

```go
routes := []struct {
	method string
	path   string
	body   string
}{
	{http.MethodPost, "/v1/pick-pack/sessions", "bad"},
	{http.MethodGet, "/v1/pick-pack/sessions", ""},
	{http.MethodGet, "/v1/pick-pack/sessions/bad", ""},
	{http.MethodPost, "/v1/pick-pack/sessions/bad/scan", `{}`},
	{http.MethodPost, "/v1/pick-pack/sessions/bad/move-to-packing", ""},
	{http.MethodPost, "/v1/pick-pack/sessions/bad/items/bad/pack", `{}`},
	{http.MethodPost, "/v1/pick-pack/sessions/bad/complete", ""},
	{http.MethodPost, "/v1/pick-pack/sessions/bad/cancel", ""},
}
```

Expected result for permissions like `[]string{model.PermOrdersView, model.PermOrdersEdit}`: `http.StatusForbidden`.

- [x] Add allowed validation coverage:
  - `POST /v1/pick-pack/sessions` with invalid JSON returns `400`.
  - `GET /v1/pick-pack/sessions/bad` returns `400`.
  - `POST /v1/pick-pack/sessions/bad/scan` returns `400`.
  - `POST /v1/pick-pack/sessions/bad/move-to-packing` returns `400`.
  - `POST /v1/pick-pack/sessions/bad/items/bad/pack` returns `400`.
  - `POST /v1/pick-pack/sessions/bad/complete` returns `400`.
  - `POST /v1/pick-pack/sessions/bad/cancel` returns `400`.

- [x] Run the new tests before implementation and confirm they fail on missing route guard:

```bash
cd apps/api-server
go test ./internal/router -run TestPickPackRoutesRequireWarehousePermission -count=1
```

Expected before implementation: at least the denied cases fail because the request reaches handler validation instead of returning `403`.

## Task 2: Add The Route Guard

- [x] Modify `apps/api-server/internal/router/router.go`:

```go
// Pick & Pack workflow - requires warehouses.manage while the module is validation-gated.
r.Route("/pick-pack/sessions", func(r chi.Router) {
	r.Use(requirePermission(model.PermWarehousesManage))
	r.Post("/", deps.PickPack.CreateSession)
	r.Get("/", deps.PickPack.ListSessions)
	r.Get("/{id}", deps.PickPack.GetSession)
	r.Post("/{id}/scan", deps.PickPack.ScanItem)
	r.Post("/{id}/move-to-packing", deps.PickPack.MoveToPacking)
	r.Post("/{id}/items/{itemId}/pack", deps.PickPack.MarkItemPacked)
	r.Post("/{id}/complete", deps.PickPack.CompleteSession)
	r.Post("/{id}/cancel", deps.PickPack.CancelSession)
})
```

- [x] Run:

```bash
cd apps/api-server
go test ./internal/router -run TestPickPackRoutesRequireWarehousePermission -count=1
```

Expected after implementation: pass.

## Task 3: Update Security Context

- [x] Update the recent security section heading in `.claude/context/SECURITY_POSTURE.md` to include 2026-05-18.
- [x] Add a bullet to that recent security section.

Suggested text:

```markdown
- OPE-436: Pick & Pack API session routes now require `warehouses.manage` via router-level RBAC. Users with only order permissions receive `403`, while warehouse-authorized users reach handler validation; dashboard `client-ready` exposure remains blocked until Pick & Pack is certified.
```

## Task 4: Validation

- [x] Run targeted router tests:

```bash
cd apps/api-server
go test ./internal/router -run TestPickPackRoutesRequireWarehousePermission -count=1
```

- [x] Run router package tests:

```bash
cd apps/api-server
go test ./internal/router -count=1
```

- [x] Run API tests if the targeted package is clean:

```bash
cd apps/api-server
go test ./...
```

- [x] Run repository checks before push:

```bash
git diff --check
./scripts/local-ci.sh
```

## Risk And Rollback

- Risk: default member users with `orders.edit` will no longer be able to use Pick & Pack API directly.
- Mitigation: Pick & Pack is already hidden from `client-ready`; this is the desired conservative state until explicit operator-role design exists.
- Rollback: revert the router guard and test commit. No schema or data rollback is required.

## Completion Criteria

- Pick & Pack session routes reject authenticated users without `warehouses.manage`.
- Warehouse-authorized tokens reach existing handler validation.
- Security posture docs record the RBAC boundary.
- OPE-403 gated files remain untouched.
