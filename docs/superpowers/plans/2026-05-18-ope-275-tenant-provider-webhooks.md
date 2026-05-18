# OPE-275 Tenant Provider Webhooks Implementation Plan

> Execution note: implement this plan task-by-task and keep the checkbox status current.

**Goal:** Make Allegro and InPost inbound webhooks tenant/integration scoped without requiring customers to change Kubernetes or deployment secrets.

**Architecture:** Add scoped callback URLs `/v1/webhooks/allegro/{integration_id}` and `/v1/webhooks/inpost/{integration_id}`. Store each provider webhook signing secret inside encrypted `integrations.credentials` as `webhook_secret`, resolve the integration through the privileged worker pool, and process events only for the verified tenant/integration scope. Keep the current unscoped `/v1/webhooks/{provider}` routes as legacy deployment-secret fallback until production provider callbacks are migrated.

**Tech Stack:** Go 1.25 API server, chi router, pgx, encrypted integration credentials, Next.js dashboard integration setup forms, Vitest/ESLint/local CI.

---

## Scope

- Public repo only.
- Backend and dashboard only; no enterprise deploy change in this PR.
- No database migration: `webhook_secret` lives in the already encrypted `integrations.credentials` JSON.
- No secret values in API responses, logs, PR text, or documentation examples.
- Legacy `ALLEGRO_WEBHOOK_SECRET` and `INPOST_WEBHOOK_SECRET` stay supported for existing unscoped callbacks.

## Files And Responsibilities

- `apps/api-server/internal/service/provider_webhook_secret_resolver.go`
  - New resolver for active integration lookup by provider + integration ID.
  - Decrypt credentials and return only scope metadata plus signing secret to the handler.
- `apps/api-server/internal/service/provider_webhook_secret_resolver_test.go`
  - Pure tests for extracting `webhook_secret` from decrypted credential JSON.
- `apps/api-server/internal/service/integration_service.go`
  - Merge partial credential updates with existing decrypted credentials so webhook secret rotation does not wipe provider credentials.
- `apps/api-server/internal/service/integration_service_test.go`
  - Add merge-helper tests for credential update behavior.
- `apps/api-server/internal/handler/allegro_webhook_handler.go`
  - Read optional `{integrationID}` from chi route.
  - Verify scoped signatures with the integration secret.
  - Dispatch to tenant-scoped syncer methods when scoped.
  - Preserve unscoped legacy behavior.
- `apps/api-server/internal/handler/inpost_webhook_handler.go`
  - Read optional `{integrationID}` from chi route.
  - Verify scoped signatures with the integration secret.
  - Update shipment status only inside verified tenant scope when scoped.
  - Preserve unscoped legacy behavior.
- `apps/api-server/internal/handler/allegro_webhook_handler_test.go`
  - Add scoped valid, wrong-secret and missing-secret tests.
- `apps/api-server/internal/handler/inpost_webhook_handler_test.go`
  - Add scoped valid, wrong-secret and missing-secret tests.
- `apps/api-server/internal/router/router.go`
  - Register `/v1/webhooks/allegro/{integrationID}` and `/v1/webhooks/inpost/{integrationID}` next to legacy routes.
- `apps/api-server/internal/worker/allegro_webhook_syncer.go`
  - Add tenant/integration-scoped import and status update methods.
- `apps/api-server/internal/service/shipment_service.go`
  - Add tenant-scoped `UpdateStatusByTrackingNumberForTenant`.
- `apps/api-server/cmd/server/main.go`
  - Wire resolver with workerPool + encryption key into webhook handlers.
- `apps/dashboard/src/lib/constants.ts`
  - Add optional `webhook_secret` credential fields for Allegro and InPost.
- `apps/dashboard/src/app/(dashboard)/marketplaces/allegro/page.tsx`
  - Add optional webhook secret input in create/update forms and submit it only when filled.
- `apps/dashboard/messages/pl/integrations.json`
- `apps/dashboard/messages/en/integrations.json`
  - Add clear labels/help copy without exposing stored values. The dedicated Allegro page still reads its `marketplaces` namespace keys from `integrations.json`, so no `marketplaces.json` edit was needed for this PR.
- `docs/system-documentation.md`
  - Document scoped callback URLs and legacy fallback.
- `docs/superpowers/plans/2026-05-18-ope-275-tenant-provider-webhooks.md`
  - Track implementation and validation evidence.

## Implementation Tasks

### Task 1: Backend Secret Resolver

- [x] Add resolver type with scope:
  - `ProviderWebhookScope{TenantID, IntegrationID, Provider, Legacy bool}`
  - `Resolve(ctx, provider, integrationID)` returns decrypted `webhook_secret` for active integration.
- [x] Extract helper:
  - `ExtractProviderWebhookSecret(credentials []byte) (string, bool)`
  - Accept only `webhook_secret` string; trim whitespace; return false when empty/missing.
- [x] Add tests:
  - returns secret when present,
  - returns false for missing/empty/non-string,
  - never logs or returns the whole credentials payload.
- [x] Add credential merge helper tests in `integration_service_test.go`:
  - partial update preserves existing client/token fields,
  - blank `webhook_secret` does not overwrite existing secret,
  - non-empty `webhook_secret` rotates only that field.

### Task 2: Scoped Routes And Handler Verification

- [x] Add chi routes:
  - `POST /v1/webhooks/allegro/{integrationID}`
  - `POST /v1/webhooks/inpost/{integrationID}`
- [x] In handlers, if `integrationID` exists:
  - parse UUID,
  - resolve tenant-scoped secret,
  - reject missing resolver/secret with `422`,
  - verify HMAC with resolved secret,
  - process only inside resolved scope.
- [x] If `integrationID` is absent:
  - preserve legacy env-secret behavior exactly.

### Task 3: Tenant-Scoped Processing

- [x] Extend `AllegroOrderSyncer` with:
  - `ImportOrderForIntegration(ctx, tenantID, integrationID, allegroOrderID)`
  - `UpdateOrderStatusForIntegration(ctx, tenantID, integrationID, allegroOrderID)`
- [x] Implement exact-integration lookup in `AllegroWebhookSyncer`.
- [x] Add `ShipmentService.UpdateStatusByTrackingNumberForTenant`.
- [x] Update InPost handler so scoped webhook cannot update another tenant's shipment.

### Task 4: UI Setup Fields

- [x] Add optional `webhook_secret` password field for generic Allegro/InPost credential forms.
- [x] Add optional webhook secret field to the dedicated Allegro setup/update cards.
- [x] Submit `webhook_secret` only when filled.
- [x] Preserve edit-mode behavior: blank secret means keep existing encrypted value.

### Task 5: Docs And Validation

- [x] Update system documentation for:
  - scoped provider callback URLs,
  - encrypted `webhook_secret`,
  - legacy fallback routes.
- [x] Run targeted tests:
  - `cd apps/api-server && go test ./internal/service ./internal/handler ./internal/worker`
  - `cd apps/dashboard && npm run lint:quiet`
- [x] Run mandatory full public validation:
  - `./scripts/local-ci.sh`
  - `git diff --check`
- [ ] Open PR with `Docs updated`.

## Validation Evidence

- `cd apps/api-server && go test ./internal/service ./internal/handler ./internal/worker -count=1` — passed.
- `cd apps/dashboard && npx vitest run src/app/'(dashboard)'/marketplaces/allegro/page.test.tsx --reporter=dot` — passed, 3 tests.
- `cd apps/dashboard && npm run lint:quiet` — passed.
- `git diff --check` — passed.
- `./scripts/local-ci.sh` — passed, all 10 checks, 102s total.

## Test Plan

- Backend unit tests prove wrong scoped secret does not dispatch work.
- Backend unit tests prove scoped InPost processing uses tenant-scoped update path.
- Existing legacy webhook tests still pass.
- Full local CI must pass before push.

## Risk And Rollback

- Risk: provider callbacks currently configured to unscoped URLs could still depend on env secrets.
- Mitigation: keep legacy routes and env-secret fallback unchanged.
- Risk: credential update could wipe existing credentials if only webhook secret is rotated.
- Mitigation: merge partial credential updates with decrypted existing credentials in `IntegrationService.Update`.
- Rollback: revert PR; no migration or data shape change is required because `webhook_secret` is optional JSON inside encrypted credentials.
