# Project State
Updated: 2026-02-25

## Target
Open production for paying customers: **May 2026** (~11 weeks remaining)

## Pricing Model
Subscription: Standard / Plus / Pro tiers based on order volume. Details TBD.

## Current Focus (P0 — must do next)
- [ ] Billing/Subscription integration (Stripe + Przelewy24?) — **0% done, #1 blocker**
- [ ] Monitoring/Alerting (Grafana/Sentry) — **10% done, SaaS requirement**
- [ ] Onboarding wizard (first-time user flow) — **25% done**
- [x] Allegro competitive parity — **DONE** (PR #58: offer import, stock sync, messaging, KSeF auto-send)

## In Progress
- [x] Supabase migration — DONE (simple_protocol JSONB fix deployed 2026-02-20)
- [x] CSRF cross-subdomain fix — DONE (PR #36, double-submit cookie with Domain=.openoms.org)
- [x] Security audit HIGH findings — DONE (all 4 HIGH fixed in PR #36, additional fixes in PR #38)

## Recently Completed
- 2026-02-28: Erli SDK rebuild — fixed base URL (→ erli.pl/svc/shop-api), endpoints (/products/{externalId}), status mapping (3 statuses), polling (purchased), pagination (after), 202 async handling, provider integration, tests; security audit 1 HIGH (provider.go sandbox flag fix required), 4 MEDIUM (deferred), CONDITIONAL PASS
- 2026-02-25: Allegro competitive parity — offer import (SKU matching + auto-pagination), stock sync (per-channel push, error counts), message templates CRUD, send_marketplace_message action, KSeF auto-send + retry, activate_listing automation, full audit fixes (PR #58, 49 files, +5012/-139)
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
- 2026-02-25: PR #58 merged — Allegro competitive parity (offer import, stock sync, messaging, KSeF auto-send, audit fixes)
- 2026-02-25: PRs #40-56 merged — Allegro hardening, weight propagation, audit v3, test coverage expansion
- 2026-02-24: PR #38 merged — audit remediation v2
- 2026-02-24: PR #36 merged — security hardening + code quality
- 2026-02-20 14:00: `6bd6a7d` — JSONBCodec Marshal/Unmarshal fix (fixed login panic)
- 2026-02-20 13:50: `8dc4a19` — Tenant repository jsonb cast

## Active Blockers
- None currently blocking development

## MVP Critical Path
```
Billing → Monitoring → Onboarding → BaseLinker import → Landing page → 3 carriers
```
Note: Allegro polish, stock sync, listing sync, KSeF — all done in PR #58.

## Estimated Hours Remaining to MVP
~400h (tight fit in 600h capacity over 11 weeks)
