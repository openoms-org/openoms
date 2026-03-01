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
- 2026-03-01: Carrier SDK audit & remediation COMPLETE (DHL, DPD, GLS verified):
  - **Audit findings**: 4 CRITICAL test failures, 2 HIGH frontend bugs, 4 MEDIUM best practices identified
  - **DHL24 SOAP WebAPI2**: Fictional REST API → correct SOAP marshaling, service types AH/09/12/EK/PI, XML response parsing
  - **DPD REST API**: Fictional URLs → dpdservices.dpd.com.pl, session auth, two-phase labels, COD+Insurance form fields
  - **GLS ShipIT API**: Bearer → Basic Auth, tracking GET→POST, cancel DELETE→POST, model alignment, test assertion fixes
  - **Status**: All 3 carriers now VERIFIED with specification tests (specs cover official API contracts)
  - **Commits**: 9859edb (DHL), 92727d7 (DPD), 80a8663+f4b9419 (GLS), 2943d6b (spec tests)
- 2026-03-01: Carrier fields fix — FedEx, UPS, GLS, Poczta Polska service type corrections + GLS backend wiring (PR #77)
- 2026-02-28: Erli SDK rebuild — base URL, endpoints, statuses, polling, pagination, 202 handling, sandbox fail-open fix (PR #76)
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
