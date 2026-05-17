# OPE-319 Checkout Finalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make paid checkout registration resilient when the final Stripe session fetch fails, so tenant billing customer/subscription records are not silently lost.

**Architecture:** Persist Stripe customer/subscription references on checkout completion, then let post-registration finalization fall back to those persisted references if Stripe is temporarily unavailable. Keep old SECURITY DEFINER functions available for blue-green compatibility and add new versioned functions for the richer shape.

**Tech Stack:** Go 1.25, stripe-go/v82, pgx, PostgreSQL SECURITY DEFINER migrations, testify.

---

## Scope

- Public repo only.
- Backend billing/registration path only.
- No dashboard UI changes.
- No live Stripe or production config changes.

## Files

- Modify: `apps/api-server/internal/service/checkout_service.go`
  - Add injectable Stripe session fetcher for tests.
  - Fetch checkout sessions with `customer` and `subscription` expansions.
  - Persist Stripe refs during status checks.
  - Finalize checkout from live Stripe data or stored DB refs.
  - Return finalization errors to the caller instead of hiding them internally.
- Modify: `apps/api-server/internal/service/stripe_webhook_service.go`
  - Persist Stripe refs when `checkout.session.completed` webhook arrives.
- Modify: `apps/api-server/internal/handler/auth_handler.go`
  - Log returned finalization errors after successful registration without failing already-created tenant/user response.
- Modify: `apps/api-server/internal/repository/interfaces.go`
  - Extend `BillingRepo.CompleteCheckoutSession` to accept persisted Stripe refs.
- Modify: `apps/api-server/internal/repository/billing_repository.go`
  - Call a new versioned SECURITY DEFINER completion function and read refs from a new getter function.
- Modify: `apps/api-server/internal/model/billing.go`
  - Add nullable persisted Stripe refs to `BillingCheckoutSession`.
  - Add an internal `CheckoutSessionStripeRefs` helper struct.
- Create: `apps/api-server/internal/service/checkout_service_test.go`
  - Regression tests for Stripe outage fallback and finalization error surfacing.
- Modify: `apps/api-server/internal/service/stripe_webhook_service_test.go`
  - Update fake repo signature.
- Create: `apps/api-server/migrations/000026_billing_checkout_session_refs.up.sql`
  - Add nullable reference/status columns to `billing_checkout_sessions`.
  - Add indexes for stored Stripe IDs.
  - Add `billing_complete_checkout_session_with_refs(...)`.
  - Add `billing_get_checkout_session_with_refs(...)`.
  - Revoke PUBLIC and grant only app/auth roles.
- Create: `apps/api-server/migrations/000026_billing_checkout_session_refs.down.sql`
  - Drop new functions, indexes, constraint, and nullable columns for rollback.
- Modify: `.claude/context/DOMAIN_MODEL.md`
  - Document the persisted checkout refs.
- Modify: `docs/system-documentation.md`
  - Update billing checkout sessions table summary.

## Implementation Tasks

### Task 1: Branch and Linear state

- [ ] Fetch `origin/main`.
- [ ] Create branch `fix/OPE-319-checkout-finalization` from `origin/main`.
- [ ] Move Linear `OPE-319` to `In Progress`.

### Task 2: RED test for fallback finalization

- [ ] Add `TestCheckoutServiceFinalizeCheckoutClaimUsesStoredStripeRefsWhenStripeFetchFails`.
- [ ] Fake repo returns a registered checkout session with persisted `stripe_customer_id`, `stripe_subscription_id`, and `subscription_status`.
- [ ] Fake Stripe fetcher returns an error.
- [ ] Expected before fix: test fails because `FinalizeCheckoutClaim` returns no error and creates no billing records.
- [ ] Run:

```bash
cd public/apps/api-server
go test ./internal/service -run TestCheckoutServiceFinalizeCheckoutClaimUsesStoredStripeRefsWhenStripeFetchFails -count=1
```

### Task 3: GREEN service/repository implementation

- [ ] Add `CheckoutSessionStripeRefs` model and fields on `BillingCheckoutSession`.
- [ ] Change `BillingRepo.CompleteCheckoutSession` signature to accept refs.
- [ ] Add checkout-session Stripe fetcher injection to `CheckoutService`, defaulting to `session.Get`.
- [ ] Extract customer/subscription refs from Stripe checkout session safely.
- [ ] Make `FinalizeCheckoutClaim` return `error`.
- [ ] On live Stripe fetch failure, reload `BillingCheckoutSession` and use stored refs.
- [ ] Create `billing_customers` and `billing_subscriptions` from either live refs or stored refs.
- [ ] Use stored or inferred `active`/`trialing` status only when Stripe status is absent; later subscription webhooks remain source of truth.

### Task 4: Migration

- [ ] Add nullable columns and new versioned functions in `000026_billing_checkout_session_refs.up.sql`.
- [ ] Keep old functions untouched for blue-green compatibility.
- [ ] Add down migration.
- [ ] Run migration safety/timeouts checks through local CI later.

### Task 5: Webhook/status persistence

- [ ] Update `GetSessionStatus` to fetch with `customer` and `subscription` expansions and persist refs on completion.
- [ ] Update Stripe webhook handler to persist refs from `checkout.session.completed`.
- [ ] Update fake repos/tests for new method signature.

### Task 6: Docs

- [ ] Update billing checkout session fields in `.claude/context/DOMAIN_MODEL.md`.
- [ ] Update `docs/system-documentation.md` billing table row.

### Task 7: Validation and PR

- [ ] Run targeted service tests:

```bash
cd public/apps/api-server
go test ./internal/service -count=1
```

- [ ] Run diff checks:

```bash
cd public
git diff --check
```

- [ ] Run full local CI on clean HEAD before push:

```bash
cd public
./scripts/local-ci.sh
```

- [ ] Commit with `OPE-319:` prefix.
- [ ] Push branch and open PR.
- [ ] Check CI, CodeQL, and CodeRabbit comments/review threads before merge.
- [ ] After merge, verify public release and enterprise deploy.
- [ ] Comment evidence on Linear and move `OPE-319` to Done.

## Risks and Rollback

- Risk: Blue-green migration could break old pods if existing function signatures are replaced. Mitigation: create new versioned SECURITY DEFINER functions and leave old function signatures intact.
- Risk: Stored webhook refs may not include expanded subscription status. Mitigation: finalization fetches Stripe with expansions; fallback only infers `trialing` for trial plans or `active` otherwise when Stripe is unavailable and no stored status exists.
- Risk: New nullable columns are safe for old code, but down migration drops them. Rollback should use the down migration only if the new app version is not running.
- Risk: Registration is already successful before finalization. Handler should log finalization errors loudly but still return the registration response; persisted refs make a retry/reconciliation path possible.
