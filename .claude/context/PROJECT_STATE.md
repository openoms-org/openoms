# Project State
Updated: 2026-02-20 by Rafal

## Target
Open production for paying customers: **May 2026** (~12 weeks remaining)

## Pricing Model
Subscription: Standard / Plus / Pro tiers based on order volume. Details TBD.

## Current Focus (P0 — must do next)
- [ ] Billing/Subscription integration (Stripe + Przelewy24?) — **0% done, #1 blocker**
- [ ] Monitoring/Alerting (Grafana/Sentry) — **10% done, SaaS requirement**
- [ ] Onboarding wizard (first-time user flow) — **25% done**
- [ ] Allegro edge cases polish — **70% done, needs hardening**

## In Progress
- [x] Supabase migration — DONE (simple_protocol JSONB fix deployed 2026-02-20)
- [ ] CSRF cross-subdomain fix — plan exists (plan: fancy-cuddling-snail.md), not implemented
- [ ] Security audit HIGH findings — 2 unfixed (WebSocket origin, SSRF in automation)

## Recently Completed
- 2026-02-20: JSONB type registration fix for pgx simple_protocol (AfterConnect + JSONBCodec)
- 2026-02-20: Tenant repository explicit jsonb cast
- 2026-02-19: Supplier product enrichment (PR #13)
- 2026-02-19: BTP.pro SDK and supplier integration
- 2026-02-17: Security audit completed (4 HIGH, 4 MEDIUM findings)
- 2026-02-17: Gap analysis vs competitors (BaseLinker, Sellasist, Apilo)

## Recent Deploys
- 2026-02-20 14:00: `6bd6a7d` — JSONBCodec Marshal/Unmarshal fix (fixed login panic)
- 2026-02-20 13:50: `8dc4a19` — Tenant repository jsonb cast
- 2026-02-20 13:40: `b9aab06` — Lint fix (unused ctx param)
- 2026-02-20 13:30: `ed3bbe2` — json.RawMessage JSONB type registration

## Active Blockers
- None currently blocking development

## MVP Critical Path
```
Billing → Monitoring → Onboarding → Allegro polish → Stock sync
→ Listing sync → KSeF → BaseLinker import → Landing page → 3 carriers
```

## Estimated Hours Remaining to MVP
~400h (tight fit in 600h capacity over 12 weeks)
