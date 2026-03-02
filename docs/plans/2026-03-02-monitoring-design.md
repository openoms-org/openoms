# Monitoring & Observability Design

> **Decision:** Sentry SaaS (error tracking) + Grafana Cloud SaaS (metrics/dashboards) — pragmatic hybrid, no self-hosted infra.

**Goal:** Production-grade observability for SaaS launch — catch errors before customers report them, monitor API health, alert on degradation.

**Architecture:**

```
Go API ─── sentry-go SDK ──────────────→ Sentry SaaS (errors, panics, context)
       │
       └── Prometheus /metrics ──→ Grafana Alloy (k3s) ──→ Grafana Cloud (dashboards, alerts)

Next.js ── @sentry/nextjs ─────────────→ Sentry SaaS (client + server errors, source maps)
```

---

## 1. Go API — Sentry Integration

**SDK:** `github.com/getsentry/sentry-go` + `sentry-go/http`

**Integration points:**

1. **Middleware** — insert in `router.go` after RealIP, before Recoverer. Captures panics, enriches events with request context (method, URL, status, tenant_id, user_id).

2. **Worker panic recovery** — `manager.go:safeRun()` and `asyncutil/safego.go` already have `recover()` blocks that log to slog. Add `sentry.CaptureException()` alongside.

3. **Error responses** — `handler/response.go:writeServerError()` logs 5xx errors via slog. Add Sentry capture with request context for server errors.

**Config (env vars, optional — empty DSN = disabled):**
- `SENTRY_DSN` — Sentry project DSN
- `SENTRY_ENVIRONMENT` — defaults to `ENV` value
- `SENTRY_TRACES_SAMPLE_RATE` — `0.1` prod, `1.0` dev

**Init:** Conditional in `main.go`, same pattern as license/billing (disabled if no DSN).

---

## 2. Next.js Dashboard — Sentry Integration

**SDK:** `@sentry/nextjs`

**Files:**
- `sentry.client.config.ts` — client-side error capture
- `sentry.server.config.ts` — server-side error capture
- `sentry.edge.config.ts` — middleware error capture
- `instrumentation.ts` — Next.js instrumentation hook

**Source maps:**
- Enable `productionBrowserSourceMaps` during CI build only
- Upload to Sentry via `@sentry/cli` in `release.yml` workflow
- Readable stack traces in Sentry instead of minified code

**CSP:** Add `https://*.sentry.io` to `connect-src` in headers config.

**Error boundary:** `@sentry/nextjs` wraps React error boundaries automatically. Wire `global-error.tsx` to `Sentry.captureException`.

---

## 3. Grafana Cloud — Metrics

**Agent:** Grafana Alloy (lightweight, ~50MB RAM, successor to Grafana Agent)

**Flow:**
```
Go API /metrics (Prometheus text format, Bearer token auth)
    → Grafana Alloy (scrape every 15s)
    → Grafana Cloud remote_write endpoint
    → Dashboards + Alerts
```

**Existing metrics (no code changes needed):**
- `openoms_http_requests_total` (counter) — by method, route, status
- `openoms_http_request_duration_seconds` (histogram) — by method, route
- `openoms_http_active_requests` (gauge) — in-flight requests
- `openoms_http_response_bytes_total` (counter) — response bytes

**Dashboards:**
- API Overview: request rate, error rate (4xx/5xx), latency p50/p95/p99
- Error breakdown: by route, by status code
- Active requests: concurrent load

**Alerts (Discord webhook):**
- Error rate > 5% for 5 min
- API latency p95 > 2s for 5 min
- No scrape data for 5 min (Alloy down)

---

## 4. Helm + Config + CI

**Environment variables:**

| Var | Where | Type |
|-----|-------|------|
| `SENTRY_DSN` | API server | Secret |
| `SENTRY_ENVIRONMENT` | API server | ConfigMap |
| `SENTRY_TRACES_SAMPLE_RATE` | API server | ConfigMap |
| `NEXT_PUBLIC_SENTRY_DSN` | Dashboard (build arg) | Secret |
| `SENTRY_AUTH_TOKEN` | CI only | GitHub Secret |
| `GRAFANA_REMOTE_WRITE_URL` | Alloy config | Secret |
| `GRAFANA_USER` | Alloy config | Secret |
| `GRAFANA_TOKEN` | Alloy config | Secret |

**Alloy deployment:** Helm chart `grafana/alloy` in `monitoring` namespace, scrapes `openoms-api` service.

**CI additions (`release.yml`):**
- After dashboard Docker build: upload source maps to Sentry
- Requires `SENTRY_AUTH_TOKEN` and `SENTRY_ORG`/`SENTRY_PROJECT` GitHub secrets

---

## 5. Implementation Order

1. Go API: config + SDK init + middleware + worker hooks + error capture
2. Next.js: SDK setup + instrumentation + error boundary + CSP
3. Helm: env vars in deployment + Alloy manifests
4. CI: source maps upload step in release workflow
5. Grafana Cloud: dashboards + alerts (manual in UI)

**Estimated effort:** ~35h total

---

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Error tracking | Sentry SaaS | Best-in-class UX, free tier sufficient, zero maintenance |
| Metrics | Grafana Cloud SaaS | Free tier 10K series, zero maintenance, existing /metrics endpoint |
| Metrics agent | Grafana Alloy | Lightweight, scrapes Prometheus, forwards to Cloud |
| OTel | Deferred to post-MVP | Existing Prometheus metrics work, Sentry beats OTel for errors |
| Source maps | Upload in CI | Readable stack traces without exposing maps publicly |
| Self-hosted | Rejected | Maintenance overhead, small team, MVP timeline |
