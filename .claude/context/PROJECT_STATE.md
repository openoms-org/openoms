# Project State
Updated: 2026-02-26

## Target
Open production for paying customers: **May 2026** (~11 weeks remaining)

## Pricing Model
Subscription: Standard / Plus / Pro tiers based on order volume. Details TBD.

## Current Focus (P0 — must do next)
- [ ] Billing/Subscription integration (Stripe + Przelewy24?) — **0% done, #1 blocker**
- [ ] Monitoring/Alerting (Grafana/Sentry) — **10% done, SaaS requirement**
- [ ] Onboarding wizard (first-time user flow) — **25% done**
- [ ] Allegro edge cases polish — **70% done, needs hardening**

## In Progress
- [x] Supabase migration — DONE (simple_protocol JSONB fix deployed 2026-02-20)
- [x] CSRF cross-subdomain fix — DONE (PR #36, double-submit cookie with Domain=.openoms.org)
- [x] Security audit HIGH findings — DONE (all 4 HIGH fixed in PR #36, additional fixes in PR #38)

## Recently Completed
- 2026-02-26: Erli SDK integration tests — 10 tests against real sandbox API (build tag `integration`); requires ERLI_SANDBOX_TOKEN env var; covers Orders list/get/pagination/404, Offers create/update-stock/update-price/invalid, status mapping; graceful skip when sandbox data unavailable (branch: pipeline/pipeline--mc-live-test)
- 2026-02-26: MessageTemplate handler body limits — MaxBytesReader 1MB added to Create and Update endpoints (previously unbounded)
- 2026-02-26: Warehouse document stock sync bug fix — `currentStockTotals` running map replaces `oldStockTotals` read in loop; multiple line items for same product now accumulate correctly before OnStockChange event
- 2026-02-26: Erli order poller marshal error logging — slog.Warn on JSON marshal failure for ShippingAddress/Items (was silently ignoring errors)
- 2026-02-25: Erli marketplace SDK — full implementation of MarketplaceProvider interface (PollOrders, GetOrder, PushOffer, UpdateStock, UpdatePrice), order poller worker, listings handler, provider wired to factory/handler/router; response body capped with io.LimitReader (50 MB success / 1 MB error); product-not-found 404 vs 500 discrimination (feat/erli-sdk-complete)
- 2026-02-25: Test coverage expansion — SDK 27/27, middleware 18/18, worker 19/19, handlers all covered, 54 model validators, service-layer tests (PRs #43-56)
- 2026-02-25: SafeGo panic recovery helper for goroutines (PR #52)
- 2026-02-25: Quiet CI/dev tooling — summary-only test output, eslint --quiet (PR #54)
- 2026-02-25: Audit remediation v3 — additional security, error handling, quality (PR #42)
- 2026-02-25: Weight propagation — supplier→product→shipment auto-calculation (PR #41)
- 2026-02-25: Allegro hardening — retry with backoff, bulk sync, order dedup (PR #40)
- 2026-02-24: Documentation update post-audit (PR #39)
- 2026-02-24: Audit remediation v2 — SSRF IPv6, atomic rate limiter, WS ticket-only auth, XSS fix, settings validation, dead code cleanup, 12 hook migrations (PR #38)
- 2026-02-24: Security hardening — CSRF middleware, composite token blacklist, WebSocket Origin validation, HSTS, automation SSRF fix, webhook body limits, input sanitization, response helpers (PR #36)
- 2026-02-20: JSONB type registration fix for pgx simple_protocol (AfterConnect + JSONBCodec)
- 2026-02-20: Tenant repository explicit jsonb cast
- 2026-02-19: Supplier product enrichment (PR #13)
- 2026-02-19: BTP.pro SDK and supplier integration
- 2026-02-17: Security audit completed (4 HIGH, 4 MEDIUM findings)
- 2026-02-17: Gap analysis vs competitors (BaseLinker, Sellasist, Apilo)

## Recent Deploys
- 2026-02-25: PRs #40-56 merged — Allegro hardening, weight propagation, audit v3, test coverage expansion
- 2026-02-24: PR #38 merged — audit remediation v2
- 2026-02-24: PR #36 merged — security hardening + code quality
- 2026-02-20 14:00: `6bd6a7d` — JSONBCodec Marshal/Unmarshal fix (fixed login panic)
- 2026-02-20 13:50: `8dc4a19` — Tenant repository jsonb cast

## Active Blockers
- None currently blocking development

## MVP Critical Path
```
Billing → Monitoring → Onboarding → Allegro polish → Stock sync
→ Listing sync → KSeF → BaseLinker import → Landing page → 3 carriers
```

## Estimated Hours Remaining to MVP
~400h (tight fit in 600h capacity over 11 weeks)
