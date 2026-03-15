# Production Readiness — Design Document

**Date:** 2026-03-15
**Status:** Approved
**Scope:** Enterprise infrastructure (openoms-enterprise repo) + Helm chart (public repo)

## Context

OpenOMS runs on k3s (1 control plane + 1-2 workers, CX32) on Hetzner Cloud with Cloudflare Tunnel ingress. The application is functional in production but lacks operational maturity:

- **Prometheus metrics not flowing** — Alloy DaemonSet configured but `up` query returns 0 targets. Dashboards ("OpenOMS API", "OpenOMS Cluster") exist but show no data.
- **Zero alert rules** — no notifications on pod failures, high error rates, or resource pressure.
- **Cloudflare metrics not in Grafana** — edge visibility limited to Cloudflare dashboard.
- **Backup restore untested** — daily CronJob runs, but restore process never verified.
- **No autoscaling** — manual replica management, no HPA, no load testing baseline.
- **Loki logs working** — log streams from openoms namespace confirmed flowing.

## Approach

Bottom-up: monitoring first (can't improve what you can't measure), then security, backup/DR, and autoscaling informed by real data.

## Phased Plan

### Phase 0: Fix Metrics Pipeline (Alloy → Grafana Cloud)

**Problem:** Alloy DaemonSet is configured in Helm chart (`monitoring/alloy-daemonset.yaml`) but metrics are not reaching Grafana Cloud Prometheus. `up` query returns 0 active targets.

**Likely root cause:** Alloy NetworkPolicy (`alloy-networkpolicy.yaml`) only allows egress to ports 6443 (k8s API) and 10250 (kubelet). Missing HTTPS egress (port 443) to Grafana Cloud remote_write endpoint, and missing egress to API pod port 8080 for `/metrics` scraping.

**Actions:**
1. Diagnose why Alloy is not sending metrics:
   - Check if Alloy pods are running (`kubectl get pods -n openoms -l app.kubernetes.io/component=alloy`)
   - Check Alloy logs for connection/auth errors
   - **Fix Alloy NetworkPolicy** — add egress rules for port 443 (Grafana Cloud) and port 8080 (API `/metrics`)
   - Verify `openoms-monitoring` secret exists (contains `GRAFANA_REMOTE_WRITE_URL`, `GRAFANA_USER`, `GRAFANA_TOKEN`)
   - Verify `METRICS_TOKEN` secret matches what API server expects on `/metrics`
2. Fix configuration and redeploy (PR in **public repo** for NetworkPolicy template, enterprise for values if needed)
3. Verify metrics flow:
   - `openoms_http_requests_total` (request count by status/route)
   - `openoms_http_request_duration_seconds_bucket` (latency histogram)
   - `openoms_http_active_requests` (concurrent requests gauge)
   - `container_cpu_usage_seconds_total` (pod CPU via cAdvisor)
   - `container_memory_working_set_bytes` (pod memory via cAdvisor)
4. Confirm existing dashboards populate with data

**Deliverable:** 1 PR (public repo — NetworkPolicy fix) + possibly 1 PR (enterprise — secret/values). Metrics visible in Grafana dashboards.

**Success criteria:** `sum(rate(openoms_http_requests_total[5m])) > 0` in Grafana.

### Phase 1: Critical Alert Rules

**Problem:** Zero alert rules configured. No notification on failures.

**Actions:**
1. Configure alerts via Grafana Cloud Alerting API (provisioned as code):

   **Critical (page immediately):**
   - Pod not ready > 5 minutes
   - API error rate (5xx) > 5% for 5 minutes
   - Pod restart count > 3 in 15 minutes
   - Health check (`/health`) failing for 3 minutes
   - Backup CronJob failed (via Loki log-based alert — backup emits no Prometheus metrics)

   **Warning (investigate within hours):**
   - P95 latency > 2s for 5 minutes
   - Memory usage > 80% of limit for 10 minutes
   - CPU usage > 80% of limit for 10 minutes
   - Disk usage > 85% on postgres volume

   **Info (review daily):**
   - Deployment rollout completed
   - Certificate expiry < 30 days (if applicable)

2. Set up notification channel: email on start, Discord webhook later
3. Create "Alerts Overview" dashboard panel

**Deliverable:** 1 PR (enterprise) with alert provisioning script/manifest.

**Success criteria:** Alerts fire on test conditions (e.g., scale API to 0 replicas, see "pod not ready" alert).

### Phase 2: Cloudflare Metrics in Grafana

**Problem:** Edge-level visibility only through Cloudflare dashboard. No correlation with application metrics.

**Actions:**
1. Evaluate approach:
   - **Option A:** Cloudflare Exporter (Prometheus exporter, deployed as pod) — scrapes Cloudflare GraphQL API, exposes as Prometheus metrics
   - **Option B:** Grafana Infinity datasource (already provisioned) — query Cloudflare API directly from Grafana
   - **Recommended:** Option A (exporter) — works with existing Prometheus pipeline, alertable
2. Deploy exporter with Cloudflare API token (read-only: Zone Analytics)
   - Pod spec must comply with PSS `enforce: restricted` (runAsNonRoot, readOnlyRootFilesystem, drop ALL, seccomp RuntimeDefault)
   - Verify chosen exporter image runs as non-root, or build custom
3. Create dashboard: request rate at edge, cache hit ratio, WAF blocks, threat events, bandwidth
4. Alert: WAF block spike > 100/min, unusual traffic pattern

**Deliverable:** 1 PR (enterprise) + Grafana dashboard.

### Phase 2b: Backup Restore Test (parallel with Phase 2)

**Problem:** Daily backup CronJob runs, but restore never tested. Unknown if backups are valid.

**Actions:**
1. Download latest backup from S3 (`openoms-backups/daily/`) — via `kubectl run` ephemeral pod inside cluster (operator needs S3 credentials in-cluster, not locally)
2. Spin up temporary PostgreSQL pod (isolated, NOT connected to production Supabase — use a throwaway `postgres:16-alpine` pod)
3. Restore backup, verify:
   - Table count matches production
   - Row counts on key tables (tenants, orders, users)
   - RLS policies intact
   - SECURITY DEFINER functions present
4. Document restore procedure as runbook
5. Document that Redis is intentionally non-persistent (WS tickets, rate limits, token blacklist are all ephemeral — already confirmed, needs documentation so future operators don't re-enable persistence thinking it's misconfigured)

**Deliverable:** 1 PR (enterprise) with restore runbook + optional monthly restore test CronJob.

### Phase 3: Security Hardening

**Problem:** Security is solid at application level but ingress-level rate limiting and network policy verification are pending.

**Actions:**
1. **Rate limiting on nginx-ingress:**
   - Global: `limit-req-status-code: 429`, sensible defaults
   - Per-path annotations for `/v1/auth/login` (10/min), `/v1/public/returns` (30/min)
   - Note: Cloudflare WAF already does edge rate limiting — ingress rate limiting is defense-in-depth
2. **Network policies review:**
   - Reconcile dual-source policies: enterprise manifest (`network-policies.yaml`) vs Helm chart (`networkpolicy.yaml`) — decide on single source of truth (recommend Helm-managed)
   - Verify default-deny is active (`kubectl get networkpolicy -A`)
   - Test: pod in openoms namespace cannot reach internet directly
   - Test: pod outside openoms namespace cannot reach API pods
3. **Probes verification:**
   - Current probes look correct in Helm templates — verify on live cluster
   - Confirm startup probe prevents premature kill during slow starts
4. **Scheduled image scanning:**
   - Trivy runs in CI (release workflow) but only on new builds
   - Add weekly CronJob or GitHub scheduled workflow to scan deployed images for new CVEs

**Deliverable:** 1-2 PRs (public Helm chart + enterprise values).

### Phase 4: Autoscaling & Load Testing

**Problem:** Manual scaling, no baseline performance data, no HPA.

**Actions:**
1. **k6 load test suite:**
   - Auth flow: login → token refresh → authenticated requests
   - Orders CRUD: create, list, update status, bulk operations
   - Dashboard: page loads, WebSocket connection
   - Targets: establish baseline RPS, P95 latency, error rate at various loads
   - Run against staging (not production)

2. **HPA configuration:**
   - API server: scale on CPU (target 70%) — use built-in metrics-server (k3s ships with it)
   - Dashboard: scale on CPU (target 70%)
   - Worker: intentionally excluded from HPA (single instance with distributed locks, scale manually if needed)
   - Min: 2 replicas (HA), Max: 5 (cost control on CX32 workers)
   - Scale-down stabilization: 300s (prevent flapping)
   - Note: custom Prometheus metrics (e.g. `openoms_http_active_requests`) require KEDA or Prometheus Adapter — defer to future iteration, CPU-based HPA is sufficient for now

3. **Pod anti-affinity:**
   - Prefer scheduling API replicas on different nodes
   - `preferredDuringSchedulingIgnoredDuringExecution` (soft, not hard — we may have only 1-2 workers)

4. **Resource limits tuning:**
   - Analyze real usage data from Phase 0 (CPU/memory per pod over 1+ week)
   - Adjust requests/limits to match actual usage + 30% headroom
   - Current defaults may be over/under-provisioned

**Deliverable:** 2-3 PRs (k6 suite, HPA config, resource tuning).

**Prerequisite:** Phase 0 complete (need real metrics for HPA targets and resource tuning).

## What We're NOT Doing

- Multi-region / managed Kubernetes migration (future, when client count justifies)
- PostgreSQL read replicas / PgBouncer (current load doesn't warrant)
- Redis persistence (confirmed ephemeral data only)
- Log aggregation changes (Loki already works)
- Application-level changes (this is infrastructure only)

## Dependencies

```
Phase 0 (metrics) ← required by → Phase 1 (alerts), Phase 4 (autoscaling)
Phase 2 (cloudflare) ← independent
Phase 2b (backup) ← independent
Phase 3 (security) ← independent (but benefits from Phase 1 alerts)
Phase 4 (autoscaling) ← requires Phase 0 data (1+ week of metrics)
```

## Repo Ownership

| Phase | Public repo (Helm chart) | Enterprise repo |
|-------|--------------------------|-----------------|
| 0 | Alloy NetworkPolicy fix | Values/secrets if needed |
| 1 | — | Alert provisioning |
| 2 | — | Cloudflare exporter deploy |
| 2b | — | Restore runbook |
| 3 | Ingress annotations, NetworkPolicy | Values overlay |
| 4 | HPA templates, anti-affinity, k6 | Values overlay, k6 scripts |
