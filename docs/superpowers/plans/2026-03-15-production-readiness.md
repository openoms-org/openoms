# Production Readiness Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make OpenOMS production infrastructure observable, alertable, secure, and auto-scalable.

**Architecture:** Phased bottom-up approach — fix metrics pipeline first, then build alerts on real data, harden security, verify backups, and finally configure autoscaling informed by actual usage patterns. Each phase produces independent PRs.

**Tech Stack:** Helm charts, Grafana Cloud API, Alloy (metrics/logs collector), k6 (load testing), Kubernetes HPA, nginx-ingress annotations

**Spec:** `docs/superpowers/specs/2026-03-15-production-readiness-design.md`

**GitHub Project:** "OpenOMS Production Readiness" (enterprise repo, project #1)

---

## Chunk 1: Phase 0 — Fix Metrics Pipeline

### Task 1: Diagnose and fix Alloy metrics pipeline

**GitHub Issue:** openoms-org/openoms-enterprise#38
**Branch:** `fix/alloy-metrics` (public repo, enterprise repo if needed)

**Problem:** Prometheus metrics not reaching Grafana Cloud (`up` returns 0 targets). Loki logs work fine.

**Important context:** The namespace-wide `openoms-egress` NetworkPolicy already allows DNS (53), HTTPS (443), and intra-namespace traffic for ALL pods in the openoms namespace. So the Alloy-specific NetworkPolicy is NOT the blocker — the real cause is likely missing credentials or misconfigured secrets.

**Files (potentially):**
- Modify: `public/deploy/helm/openoms/templates/monitoring/alloy-networkpolicy.yaml` (cleanup, not root cause)
- Enterprise: secrets provisioning if `openoms-monitoring` secret is missing

- [ ] **Step 1: Check if Alloy pods are running**

```bash
export KUBECONFIG=~/.kube/openoms-config
kubectl get pods -n openoms -l app.kubernetes.io/component=alloy
```

Expected: Alloy pods `Running`. If `CrashLoopBackOff` or missing, check if monitoring is enabled.

- [ ] **Step 2: Check if `openoms-monitoring` secret exists**

The Alloy DaemonSet requires a secret named `openoms-monitoring` with keys:
`GRAFANA_REMOTE_WRITE_URL`, `GRAFANA_USER`, `GRAFANA_TOKEN`, `LOKI_URL`, `LOKI_USER`

```bash
kubectl get secret -n openoms | grep monitoring
kubectl get secret openoms-monitoring -n openoms -o jsonpath='{.data}' | python3 -c "import json,sys,base64; d=json.load(sys.stdin); [print(f'{k}: {base64.b64decode(v).decode()[:30]}...') for k,v in d.items()]"
```

If the secret doesn't exist, this is the root cause — Alloy pods won't start without these env vars (they're not marked optional).

- [ ] **Step 3: Create the monitoring secret if missing**

```bash
kubectl create secret generic openoms-monitoring -n openoms \
  --from-literal=GRAFANA_REMOTE_WRITE_URL="https://prometheus-prod-24-prod-eu-west-2.grafana.net/api/prom/push" \
  --from-literal=GRAFANA_USER="<grafana-cloud-prometheus-user-id>" \
  --from-literal=GRAFANA_TOKEN="<grafana-cloud-api-key>" \
  --from-literal=LOKI_URL="https://logs-prod-eu-west-2.grafana.net/loki/api/v1/push" \
  --from-literal=LOKI_USER="<grafana-cloud-loki-user-id>"
```

Note: Get the correct URLs and user IDs from Grafana Cloud → your stack → Details.

- [ ] **Step 4: Verify METRICS_TOKEN matches**

```bash
# Get the token Alloy uses to scrape /metrics
kubectl get secret openoms-secrets -n openoms -o jsonpath='{.data.METRICS_TOKEN}' | base64 -d

# Test /metrics endpoint directly
kubectl port-forward -n openoms svc/openoms-api 8080:8080 &
curl -H "Authorization: Bearer <METRICS_TOKEN>" http://localhost:8080/metrics | head -20
kill %1
```

If `/metrics` returns 401, the token doesn't match.

- [ ] **Step 5: Check Alloy logs for errors**

```bash
kubectl logs -n openoms -l app.kubernetes.io/component=alloy --tail=100
```

Look for:
- `msg="remote_write request successful"` — working
- `"401"` or `"403"` — auth issue with Grafana Cloud credentials
- `connection refused` — network issue (unlikely given namespace egress policy)
- `context deadline exceeded` — DNS or connectivity timeout

- [ ] **Step 6: Clean up Alloy NetworkPolicy (optional, good hygiene)**

The Alloy-specific NetworkPolicy at `alloy-networkpolicy.yaml` is redundant (namespace egress already covers it) but having explicit rules is clearer. Update it to be self-documenting:

Create branch and edit `deploy/helm/openoms/templates/monitoring/alloy-networkpolicy.yaml`:

```yaml
{{- if and .Values.networkPolicy.enabled .Values.monitoring .Values.monitoring.enabled }}
# NOTE: These rules are covered by the namespace-wide openoms-egress policy.
# This policy exists for documentation — it explicitly lists what Alloy needs.
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "openoms.fullname" . }}-alloy-egress
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
    app.kubernetes.io/component: monitoring
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/component: alloy
  policyTypes:
    - Egress
  egress:
    # DNS resolution
    - ports:
        - protocol: UDP
          port: 53
        - protocol: TCP
          port: 53
    # Grafana Cloud remote_write + Loki push (HTTPS)
    - ports:
        - protocol: TCP
          port: 443
    # Kubernetes API server (pod/node discovery)
    - ports:
        - protocol: TCP
          port: 6443
    # Kubelet metrics (cAdvisor + kubelet scraping)
    - ports:
        - protocol: TCP
          port: 10250
    # OpenOMS API /metrics endpoint (intra-namespace)
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: {{ include "openoms.fullname" . }}-api
      ports:
        - protocol: TCP
          port: {{ .Values.apiServer.port }}
{{- end }}
```

- [ ] **Step 7: Verify metrics flow in Grafana**

```bash
curl -s -H "Authorization: Bearer <GRAFANA_TOKEN>" \
  "https://openoms.grafana.net/api/prometheus/grafanacloud-prom/api/v1/query?query=up" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print(f'Active targets: {len(d[\"data\"][\"result\"])}')"
```

Expected: `Active targets: > 0`

Then verify application metrics:
```bash
curl -s -H "Authorization: Bearer <GRAFANA_TOKEN>" \
  "https://openoms.grafana.net/api/prometheus/grafanacloud-prom/api/v1/query?query=sum(rate(openoms_http_requests_total[5m]))"
```

- [ ] **Step 8: Verify dashboards have data**

Check "OpenOMS API" dashboard (Request Rate, Error Rate, P95 Latency) and "OpenOMS Cluster" dashboard (pod CPU/memory) show data.

- [ ] **Step 9: Commit, push, create PR (if code changes needed)**

```bash
cd public/
git checkout -b fix/alloy-metrics
git add deploy/helm/openoms/templates/monitoring/alloy-networkpolicy.yaml
git commit -m "fix(helm): clean up Alloy NetworkPolicy with explicit egress rules"
git push -u origin fix/alloy-metrics
gh pr create --title "fix(helm): clean up Alloy NetworkPolicy" --body "$(cat <<'EOF'
## Summary
- Updated Alloy NetworkPolicy to explicitly document required egress rules
- Fixed pod selector label to match actual API pod labels
- Added DNS egress (was missing from Alloy-specific policy)

## Note
The namespace-wide openoms-egress policy already covers Alloy's needs.
This change is for documentation clarity and defense-in-depth.
EOF
)"
```

- [ ] **Step 10: Close GitHub issue**

```bash
cd enterprise/
gh issue close 38 --comment "Metrics flowing. Root cause: [describe actual root cause found in diagnosis]"
```

---

## Chunk 2: Phase 1 — Alert Rules

### Task 3: Provision Grafana alert rules via API

**GitHub Issue:** openoms-org/openoms-enterprise#26
**Branch:** `feat/grafana-alerts` (enterprise repo)

**Files:**
- Create: `enterprise/scripts/grafana-alerts.sh`

**Depends on:** Phase 0 complete (metrics flowing)

- [ ] **Step 1: Create feature branch**

```bash
cd enterprise/
git checkout -b feat/grafana-alerts
```

- [ ] **Step 2: Create alert provisioning script**

Create `scripts/grafana-alerts.sh` — a shell script that uses the Grafana Cloud API to provision alert rules and a contact point. This is infrastructure-as-code: alerts defined in a script that can be re-run idempotently.

The script must:
1. Accept `GRAFANA_URL` and `GRAFANA_TOKEN` as env vars
2. Create a notification contact point (email first, Discord later)
3. Create a notification policy
4. Create alert rules in a folder "OpenOMS Alerts":

**Prometheus-based alerts (datasource: `grafanacloud-prom`):**
- `APIHighErrorRate`: `sum(rate(openoms_http_requests_total{status=~"5.."}[5m])) / sum(rate(openoms_http_requests_total[5m])) > 0.05` for 5m → Critical
- `APIHighLatency`: `histogram_quantile(0.95, sum(rate(openoms_http_request_duration_seconds_bucket[5m])) by (le)) > 2` for 5m → Warning
- `PodHighMemory`: `container_memory_working_set_bytes{namespace="openoms",container!="",container!="POD"} / on(container,namespace,pod) kube_pod_container_resource_limits{resource="memory"} > 0.8` for 10m → Warning (Note: requires kube-state-metrics or use container limits from cAdvisor)
- `PodHighCPU`: `sum(rate(container_cpu_usage_seconds_total{namespace="openoms",container!="",container!="POD"}[5m])) by (pod) > 0.8` for 10m → Warning
- `DiskUsageHigh`: `kubelet_volume_stats_used_bytes{namespace="openoms"} / kubelet_volume_stats_capacity_bytes{namespace="openoms"} > 0.85` for 10m → Warning (uses kubelet metrics already scraped by Alloy, no kube-state-metrics needed)
- `PodRestarting`: `increase(kube_pod_container_status_restarts_total{namespace="openoms"}[15m]) > 3` → Critical (Note: requires kube-state-metrics — if not available, use Loki-based alternative)

**Loki-based alerts (datasource: `grafanacloud-logs`):**
- `HealthCheckFailing`: `count_over_time({namespace="openoms",container="openoms-api"} |~ "health check failed|Health check error" [3m]) > 5` → Critical
- `BackupFailed`: `count_over_time({namespace="openoms",container=~"backup.*"} |~ "error|failed|Error|FATAL" [1h]) > 0` → Critical
- `PanicDetected`: `count_over_time({namespace="openoms"} |~ "panic|PANIC" [5m]) > 0` → Critical

**Important:** Some alerts (PodRestarting, PodHighMemory with limits comparison) require `kube-state-metrics`. If not deployed, use simplified alternatives:
- PodHighMemory simplified: `container_memory_working_set_bytes{namespace="openoms"} > 400e6` (absolute threshold based on known 512Mi limit)
- PodRestarting: Use Loki log-based `|~ "Back-off restarting failed container"` instead

- [ ] **Step 3: Test the script against Grafana Cloud**

```bash
GRAFANA_URL="https://openoms.grafana.net" \
GRAFANA_TOKEN="glsa_..." \
bash scripts/grafana-alerts.sh
```

Verify alerts appear in Grafana: Administration → Alerting → Alert rules

- [ ] **Step 4: Test an alert fires**

Scale API to 0 replicas (briefly) to trigger health check alert:
```bash
kubectl scale deployment openoms-api -n openoms --replicas=0
# Wait 3-5 minutes for alert to fire
# Check Grafana Alerting → Alert Rules → should show "Firing"
kubectl scale deployment openoms-api -n openoms --replicas=2
```

- [ ] **Step 5: Commit**

```bash
git add scripts/grafana-alerts.sh
git commit -m "feat: add Grafana alert provisioning script

Provisions critical/warning/info alerts via Grafana Cloud API:
- API error rate, latency, pod memory/CPU, panics
- Loki-based: health check, backup failures, panics
- Contact point: email (Discord can be added later)"
```

- [ ] **Step 6: Push and create PR**

```bash
git push -u origin feat/grafana-alerts
gh pr create --title "feat: provision Grafana alerting rules" --body "$(cat <<'EOF'
## Summary
- Script to provision alert rules via Grafana Cloud API
- Critical: error rate >5%, pod restarts, health check failing, panics, backup failure
- Warning: P95 latency >2s, memory >80%, CPU >80%
- Contact point: email (add Discord later)

## Test plan
- [ ] Run script against Grafana Cloud
- [ ] Verify alerts appear in Grafana UI
- [ ] Test alert fires by scaling API to 0 replicas
EOF
)"
```

- [ ] **Step 7: Update GitHub project**

```bash
gh issue edit 26 --add-label "in-progress"
# After merge:
gh issue close 26 --comment "Alert rules provisioned via grafana-alerts.sh"
```

---

## Chunk 3: Phase 2 + 2b — Cloudflare Metrics & Backup Restore

### Task 4: Deploy Cloudflare Exporter (enterprise repo)

**GitHub Issue:** openoms-org/openoms-enterprise#24, #25
**Branch:** `feat/cloudflare-metrics` (enterprise repo)

**Files:**
- Create: `enterprise/deploy/manifests/cloudflare-exporter.yaml`
- Create or update: Grafana dashboard via API or JSON import

- [ ] **Step 1: Create feature branch**

```bash
cd enterprise/
git checkout -b feat/cloudflare-metrics
```

- [ ] **Step 2: Evaluate Cloudflare exporter images for PSS compliance**

The cluster enforces PSS `enforce: restricted`. Check if `lablabs/cloudflare-exporter` or `cyrus-and/cloudflare-exporter` runs as non-root. If not, use the Grafana Infinity datasource (Option B from spec) instead — it queries Cloudflare API directly from Grafana without deploying a pod.

Option B approach (recommended if no PSS-compliant exporter exists):
- Use the existing `grafanacloud-infinity` datasource
- Create a Grafana dashboard that queries Cloudflare GraphQL API directly
- No pod deployment needed, no PSS concerns

- [ ] **Step 3: Configure Cloudflare API token**

Create a Cloudflare API token with `Zone:Analytics:Read` permission.
Store it as a Grafana Cloud datasource configuration or k8s secret.

- [ ] **Step 4: Create Grafana dashboard**

Create a dashboard "OpenOMS Cloudflare" with panels:
- Request rate at edge (total, cached, uncached)
- Cache hit ratio
- WAF events (blocked, challenged)
- Bandwidth (in/out)
- Threat events by type
- Top countries by request count

Use Cloudflare GraphQL API: `https://api.cloudflare.com/client/v4/graphql`

Query example for requests:
```graphql
{
  viewer {
    zones(filter: {zoneTag: "<ZONE_ID>"}) {
      httpRequests1mGroups(limit: 1440, filter: {datetime_geq: "$__from", datetime_lt: "$__to"}) {
        dimensions { datetime }
        sum { requests cachedRequests bytes }
      }
    }
  }
}
```

- [ ] **Step 5: Add WAF alert**

Add alert rule to `scripts/grafana-alerts.sh`:
- `CloudflareWAFSpike`: WAF blocks > 100/min for 5m → Warning

- [ ] **Step 6: Commit, push, create PR**

```bash
git add deploy/manifests/cloudflare-exporter.yaml scripts/grafana-alerts.sh  # or just the dashboard JSON
git commit -m "feat: add Cloudflare metrics to Grafana

Cloudflare edge visibility via Infinity datasource + GraphQL API.
Dashboard: request rate, cache ratio, WAF blocks, bandwidth, threats."
git push -u origin feat/cloudflare-metrics
gh pr create --title "feat: add Cloudflare metrics dashboard" --body "..."
```

- [ ] **Step 7: Update GitHub project**

```bash
gh issue close 24 --comment "Cloudflare metrics available in Grafana dashboard"
gh issue close 25 --comment "Cloudflared tunnel metrics available via Cloudflare dashboard"
```

### Task 5: Test backup restore and write runbook (enterprise repo)

**GitHub Issues:** openoms-org/openoms-enterprise#35, #36, #37
**Branch:** `feat/backup-restore-runbook` (enterprise repo)

**Files:**
- Create: `enterprise/docs/runbooks/backup-restore.md`
- Create: `enterprise/docs/runbooks/redis-persistence.md`

**This task requires SSH/kubectl access to the cluster.**

- [ ] **Step 1: Create feature branch**

```bash
cd enterprise/
git checkout -b feat/backup-restore-runbook
```

- [ ] **Step 2: List available backups**

```bash
export KUBECONFIG=~/.kube/openoms-config

# Run ephemeral pod with S3 access to list backups
kubectl run backup-check --rm -it --restart=Never \
  --image=amazon/aws-cli:2.22.35 \
  --env="AWS_ACCESS_KEY_ID=$(kubectl get secret openoms-secrets -n openoms -o jsonpath='{.data.S3_ACCESS_KEY}' | base64 -d)" \
  --env="AWS_SECRET_ACCESS_KEY=$(kubectl get secret openoms-secrets -n openoms -o jsonpath='{.data.S3_SECRET_KEY}' | base64 -d)" \
  --command -- aws s3 ls s3://openoms-backups/daily/ --endpoint-url https://fsn1.your-objectstorage.com
```

- [ ] **Step 3: Restore backup to throwaway PostgreSQL pod**

Use a two-step approach: download with aws-cli image, restore with postgres image.

```bash
# 1. Download backup to a shared emptyDir via Job (or use aws-cli pod)
kubectl run backup-download --rm -it --restart=Never \
  --image=amazon/aws-cli:2.22.35 \
  --env="AWS_ACCESS_KEY_ID=$(kubectl get secret openoms-secrets -n openoms -o jsonpath='{.data.S3_ACCESS_KEY}' | base64 -d)" \
  --env="AWS_SECRET_ACCESS_KEY=$(kubectl get secret openoms-secrets -n openoms -o jsonpath='{.data.S3_SECRET_KEY}' | base64 -d)" \
  --command -- aws s3 cp s3://openoms-backups/daily/<latest>.sql.gz /tmp/backup.sql.gz \
  --endpoint-url https://fsn1.your-objectstorage.com

# 2. Start temporary PG pod (Alpine uses apk, NOT apt-get)
kubectl run pg-restore-test --rm -it --restart=Never \
  --image=postgres:16-alpine \
  --env="POSTGRES_PASSWORD=testpass" \
  --command -- sh

# Inside the pod (Alpine shell):
# Install aws-cli via apk
apk add --no-cache aws-cli

# Download backup
aws s3 cp s3://openoms-backups/daily/<latest>.sql.gz /tmp/backup.sql.gz \
  --endpoint-url https://fsn1.your-objectstorage.com

# Start postgres and restore
pg_ctl start -D /var/lib/postgresql/data
createdb -U postgres restore_test
gunzip -c /tmp/backup.sql.gz | psql -U postgres restore_test

# Verify
psql -U postgres restore_test -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"
psql -U postgres restore_test -c "SELECT count(*) FROM tenants;"
psql -U postgres restore_test -c "SELECT count(*) FROM orders;"
psql -U postgres restore_test -c "SELECT count(*) FROM users;"
psql -U postgres restore_test -c "SELECT proname FROM pg_proc WHERE prosecdef = true AND pronamespace = 'public'::regnamespace;"
psql -U postgres restore_test -c "SELECT tablename FROM pg_tables WHERE rowsecurity = true AND schemaname = 'public';"
```

- [ ] **Step 4: Write backup restore runbook**

Create `docs/runbooks/backup-restore.md` with:
1. How to find available backups (S3 listing)
2. How to restore (step-by-step, same as above but documented)
3. How to verify restore integrity
4. How to restore to production (emergency procedure)
5. How to trigger an immediate backup

- [ ] **Step 5: Document Redis persistence decision**

Create `docs/runbooks/redis-persistence.md`:
- Redis is intentionally non-persistent
- Data stored: WebSocket tickets, rate limit counters, token blacklist, distributed locks
- All data is ephemeral and auto-regenerated
- Do NOT enable persistence — it adds complexity without value

- [ ] **Step 6: Commit, push, create PR**

```bash
git add docs/runbooks/
git commit -m "docs: add backup restore runbook and Redis persistence rationale"
git push -u origin feat/backup-restore-runbook
gh pr create --title "docs: add DR runbooks (backup restore, Redis)" --body "..."
```

- [ ] **Step 7: Update GitHub project**

```bash
gh issue close 35 --comment "Backup restore tested and documented in docs/runbooks/backup-restore.md"
gh issue close 36 --comment "Redis non-persistence documented in docs/runbooks/redis-persistence.md"
gh issue close 37 --comment "DR runbook added to docs/runbooks/"
```

---

## Chunk 4: Phase 3 — Security Hardening

### Task 6: Add rate limiting on nginx-ingress (public repo)

**GitHub Issue:** openoms-org/openoms-enterprise#30
**Branch:** `feat/ingress-rate-limiting` (public repo)

**Files:**
- Modify: `public/deploy/helm/openoms/templates/ingress.yaml`
- Modify: `public/deploy/helm/openoms/values.yaml`

- [ ] **Step 1: Create feature branch**

```bash
cd public/
git checkout -b feat/ingress-rate-limiting
```

- [ ] **Step 2: Add rate limiting annotations to ingress template**

Edit `deploy/helm/openoms/templates/ingress.yaml` — the annotations are already templated from values, so we only need to add rate limiting values.

Edit `deploy/helm/openoms/values.yaml` to add default rate limiting annotations:

```yaml
ingress:
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    # Rate limiting (defense-in-depth, Cloudflare WAF handles edge)
    nginx.ingress.kubernetes.io/limit-rps: "50"
    nginx.ingress.kubernetes.io/limit-burst-multiplier: "3"
    nginx.ingress.kubernetes.io/limit-connections: "20"
```

**Important — Cloudflare Tunnel + rate limiting interaction:**
All external traffic arrives via cloudflared pods → nginx-ingress. Without `use-forwarded-headers: "true"` in the nginx-ingress ConfigMap, rate limiting sees cloudflared's pod IP for ALL requests (all clients share one bucket).

Options:
1. Set `use-forwarded-headers: "true"` in nginx-ingress ConfigMap (enterprise repo) — then rate limiting works per real client IP via `X-Forwarded-For`
2. Keep generous limits (50 RPS) as defense-in-depth only — Cloudflare WAF handles real per-client rate limiting at the edge
3. Skip ingress rate limiting entirely — the Cloudflare WAF rules already cover this

**Recommended:** Option 2 — add generous limits + document that Cloudflare WAF is the primary rate limiter. Ingress limits protect against internal cluster abuse only.

- [ ] **Step 3: Verify template renders**

```bash
helm template openoms deploy/helm/openoms/ \
  --set ingress.enabled=true \
  --show-only templates/ingress.yaml
```

- [ ] **Step 4: Commit, push, create PR**

```bash
git add deploy/helm/openoms/templates/ingress.yaml deploy/helm/openoms/values.yaml
git commit -m "feat(helm): add rate limiting annotations to ingress"
git push -u origin feat/ingress-rate-limiting
gh pr create --title "feat(helm): add nginx-ingress rate limiting" --body "..."
```

### Task 7: Consolidate network policies (public + enterprise repo)

**GitHub Issue:** openoms-org/openoms-enterprise#31
**Branch:** `feat/consolidate-network-policies` (enterprise repo)

**Files:**
- Modify: `enterprise/deploy/manifests/network-policies.yaml` — remove openoms namespace policies (now Helm-managed)
- Verify: `public/deploy/helm/openoms/templates/networkpolicy.yaml` already covers openoms namespace

- [ ] **Step 1: Compare the two policy sources**

Helm chart `networkpolicy.yaml` manages:
- `openoms-allow-ingress` (ingress from nginx/cloudflared)
- `openoms-egress` (DNS, HTTPS, PG, Redis, intra-namespace)

Enterprise `network-policies.yaml` manages:
- `default-deny-ingress` (openoms namespace) — redundant with Helm's ingress policy + egress policy combo
- `allow-ingress-from-nginx` (openoms namespace) — redundant with Helm
- `default-deny-ingress` (apps-core) — NOT in Helm, keep
- `allow-postgres-from-openoms` (apps-core) — NOT in Helm, keep
- `allow-redis-from-openoms` (apps-core) — NOT in Helm, keep
- `default-deny-ingress` (cloudflared) — NOT in Helm, keep
- `cloudflared-egress` (cloudflared) — NOT in Helm, keep

- [ ] **Step 2: Remove only the openoms-namespace policies from enterprise manifest**

Keep the apps-core and cloudflared policies in the enterprise manifest. Remove only the openoms-namespace `default-deny-ingress` and `allow-ingress-from-nginx` since they're now fully managed by the Helm chart.

- [ ] **Step 3: Add a default-deny ingress policy to the Helm chart**

The Helm chart has allow-ingress but no explicit default-deny. Add one:

Edit `public/deploy/helm/openoms/templates/networkpolicy.yaml` to prepend:

```yaml
{{- if .Values.networkPolicy.enabled }}
# Default deny all ingress to openoms namespace
# Egress is NOT denied here — the openoms-egress policy below handles egress allowlisting
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "openoms.fullname" . }}-default-deny
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
spec:
  podSelector: {}
  policyTypes:
    - Ingress
---
```

**Important:** Only deny Ingress, NOT Egress. The existing `openoms-egress` policy already functions as a default-deny-plus-allowlist for egress (it specifies `policyTypes: [Egress]` with explicit port rules). Adding `Egress` to the default-deny would be redundant and confusing.

This makes the Helm chart the single source of truth for openoms namespace policies.

- [ ] **Step 3b: Create feature branch in public repo for networkpolicy change**

```bash
cd public/
git checkout -b feat/consolidate-network-policies
```

The public repo change (default-deny + any networkpolicy.yaml updates) needs its own branch and PR — don't commit to main directly.

- [ ] **Step 4: Verify on live cluster**

```bash
kubectl get networkpolicy -n openoms
kubectl get networkpolicy -n apps-core
kubectl get networkpolicy -n cloudflared
```

- [ ] **Step 5: Commit both repos, create PRs**

### Task 8: Add scheduled image scanning workflow (public repo)

**GitHub Issue:** openoms-org/openoms-enterprise#33
**Branch:** `feat/scheduled-image-scan` (public repo)

**Files:**
- Create: `public/.github/workflows/scheduled-scan.yml`

- [ ] **Step 1: Create weekly Trivy scan workflow**

```yaml
name: Scheduled Security Scan
on:
  schedule:
    - cron: '0 6 * * 1'  # Every Monday at 6 AM UTC
  workflow_dispatch: {}

jobs:
  scan:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        image: [openoms-api, openoms-dashboard, openoms-migrate]
    steps:
      - uses: aquasecurity/trivy-action@0.30.0
        with:
          image-ref: ghcr.io/openoms-org/${{ matrix.image }}:latest
          format: sarif
          output: ${{ matrix.image }}-results.sarif
          severity: CRITICAL,HIGH
          ignore-unfixed: true
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: ${{ matrix.image }}-results.sarif
          category: ${{ matrix.image }}-weekly
```

- [ ] **Step 2: Commit, push, create PR**

### Task 9: Verify probes on live cluster

**GitHub Issue:** openoms-org/openoms-enterprise#32

**This is a verification task — no code changes unless issues are found.**

- [ ] **Step 1: Check current probe configuration on running pods**

```bash
kubectl get pod -n openoms -l app.kubernetes.io/component=api -o jsonpath='{.items[0].spec.containers[0].livenessProbe}'
kubectl get pod -n openoms -l app.kubernetes.io/component=api -o jsonpath='{.items[0].spec.containers[0].readinessProbe}'
kubectl get pod -n openoms -l app.kubernetes.io/component=api -o jsonpath='{.items[0].spec.containers[0].startupProbe}'
```

- [ ] **Step 2: Verify probes match Helm values**

Compare output with values in `values.yaml`:
- Liveness: `/health`, initialDelay 15s, period 10s, timeout 5s, failureThreshold 5
- Readiness: `/health`, initialDelay 5s, period 5s, timeout 3s, failureThreshold 3
- Startup: `/health`, failureThreshold 30, period 2s (60s max)

- [ ] **Step 3: Check pod event history for probe failures**

```bash
kubectl describe pod -n openoms -l app.kubernetes.io/component=api | grep -A5 "Events:"
```

Look for `Unhealthy` or `FailedScheduling` events.

- [ ] **Step 4: Close issue with findings**

```bash
cd enterprise/
gh issue close 32 --comment "Probes verified on live cluster. Configuration matches Helm values. No probe failures in event history."
```

---

## Chunk 5: Phase 4 — Autoscaling & Load Testing

**Prerequisite:** Phase 0 complete, 1+ week of metrics data collected.

### Task 10: Create k6 load test suite (enterprise repo)

**GitHub Issues:** openoms-org/openoms-enterprise#22
**Branch:** `feat/k6-load-tests` (enterprise repo)

**Files:**
- Create: `enterprise/tests/k6/auth-flow.js`
- Create: `enterprise/tests/k6/orders-crud.js`
- Create: `enterprise/tests/k6/config.js`

- [ ] **Step 1: Create feature branch**

```bash
cd enterprise/
git checkout -b feat/k6-load-tests
mkdir -p tests/k6
```

- [ ] **Step 2: Create shared config**

Create `tests/k6/config.js`:

```javascript
export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
export const TENANT_SLUG = __ENV.TENANT_SLUG || '';
export const TEST_EMAIL = __ENV.TEST_EMAIL || '';
export const TEST_PASSWORD = __ENV.TEST_PASSWORD || '';

// Fail fast if required env vars are not set
if (!TENANT_SLUG || !TEST_EMAIL || !TEST_PASSWORD) {
  throw new Error('Required env vars: BASE_URL, TENANT_SLUG, TEST_EMAIL, TEST_PASSWORD');
}
```

Note: No default passwords in committed code. All credentials must be passed via `--env` flags.
Usage: `k6 run tests/k6/auth-flow.js --env BASE_URL=... --env TENANT_SLUG=mercpart --env TEST_EMAIL=... --env TEST_PASSWORD=...`

- [ ] **Step 3: Create auth flow test**

Create `tests/k6/auth-flow.js`:

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, TENANT_SLUG, TEST_EMAIL, TEST_PASSWORD } from './config.js';

export const options = {
  stages: [
    { duration: '1m', target: 10 },   // ramp up
    { duration: '3m', target: 10 },   // steady
    { duration: '1m', target: 50 },   // spike
    { duration: '2m', target: 50 },   // sustained spike
    { duration: '1m', target: 0 },    // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  // Login
  const loginRes = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
    email: TEST_EMAIL,
    password: TEST_PASSWORD,
    tenant_slug: TENANT_SLUG,
  }), { headers: { 'Content-Type': 'application/json' } });

  check(loginRes, { 'login 200': (r) => r.status === 200 });

  if (loginRes.status !== 200) return;

  const token = loginRes.json('access_token');
  const authHeaders = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };

  // Authenticated request
  const ordersRes = http.get(`${BASE_URL}/v1/orders?limit=10`, authHeaders);
  check(ordersRes, { 'orders 200': (r) => r.status === 200 });

  sleep(1);
}
```

- [ ] **Step 4: Create orders CRUD test**

Create `tests/k6/orders-crud.js` with similar structure:
- Create order → Get order → Update status → List orders
- Thresholds: P95 < 2s, error rate < 5%

- [ ] **Step 5: Test locally against staging**

```bash
k6 run tests/k6/auth-flow.js --env BASE_URL=https://staging-api.openoms.org
```

Note: Run against staging only, never production.

- [ ] **Step 6: Commit, push, create PR**

```bash
git add tests/k6/
git commit -m "feat: add k6 load test suite (auth flow, orders CRUD)"
git push -u origin feat/k6-load-tests
gh pr create --title "feat: add k6 load test suite" --body "..."
```

### Task 11: Configure HPA for API and Dashboard (public repo)

**GitHub Issues:** openoms-org/openoms-enterprise#18, #19, #20, #21
**Branch:** `feat/hpa-autoscaling` (public repo)

**Files:**
- Create: `public/deploy/helm/openoms/templates/api-server/hpa.yaml`
- Create: `public/deploy/helm/openoms/templates/dashboard/hpa.yaml`
- Modify: `public/deploy/helm/openoms/values.yaml` — add autoscaling config
- Modify: `public/deploy/helm/openoms/templates/api-server/deployment.yaml` — add anti-affinity
- Modify: `public/deploy/helm/openoms/templates/api-server/rollout.yaml` — add anti-affinity (production uses Rollout, not Deployment!)
- Modify: `public/deploy/helm/openoms/templates/dashboard/deployment.yaml` — add anti-affinity
- Modify: `public/deploy/helm/openoms/templates/dashboard/rollout.yaml` — add anti-affinity
- Modify: `enterprise/deploy/helm/values-production.yaml` — enable autoscaling

**Important:** Production has `blueGreen.enabled: true`, which means:
- `deployment.yaml` templates are wrapped in `{{- if not .Values.blueGreen.enabled }}` — they DON'T render in production
- `rollout.yaml` templates render instead (Argo Rollouts CRD)
- Anti-affinity must be added to BOTH deployment.yaml AND rollout.yaml
- HPA must conditionally target Rollout (when blueGreen enabled) or Deployment (when not)

- [ ] **Step 1: Create feature branch**

```bash
cd public/
git checkout -b feat/hpa-autoscaling
```

- [ ] **Step 2: Add autoscaling values to values.yaml**

```yaml
autoscaling:
  api:
    enabled: false
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70
    scaleDownStabilizationSeconds: 300
  dashboard:
    enabled: false
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70
    scaleDownStabilizationSeconds: 300
```

- [ ] **Step 3: Create API HPA template**

Create `deploy/helm/openoms/templates/api-server/hpa.yaml`:

```yaml
{{- if .Values.autoscaling.api.enabled }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "openoms.fullname" . }}-api
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
    app.kubernetes.io/component: api
spec:
  scaleTargetRef:
    {{- if .Values.blueGreen.enabled }}
    apiVersion: argoproj.io/v1alpha1
    kind: Rollout
    {{- else }}
    apiVersion: apps/v1
    kind: Deployment
    {{- end }}
    name: {{ include "openoms.fullname" . }}-api
  minReplicas: {{ .Values.autoscaling.api.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.api.maxReplicas }}
  behavior:
    scaleDown:
      stabilizationWindowSeconds: {{ .Values.autoscaling.api.scaleDownStabilizationSeconds }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .Values.autoscaling.api.targetCPUUtilizationPercentage }}
{{- end }}
```

Note: When `blueGreen.enabled=true`, HPA targets the Argo Rollout object (which manages its own ReplicaSets). When false, it targets the standard Deployment. Argo Rollouts natively supports being an HPA target.

- [ ] **Step 4: Create Dashboard HPA template**

Same pattern as API, in `deploy/helm/openoms/templates/dashboard/hpa.yaml`. Same conditional for Rollout vs Deployment.

- [ ] **Step 5: Add pod anti-affinity to BOTH deployment AND rollout templates**

Edit `deploy/helm/openoms/templates/api-server/deployment.yaml` AND `deploy/helm/openoms/templates/api-server/rollout.yaml` — add to pod spec:

```yaml
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app.kubernetes.io/name: {{ include "openoms.fullname" . }}-api
                topologyKey: kubernetes.io/hostname
```

Same for dashboard deployment.yaml AND dashboard rollout.yaml.

**Important:** Use label `app.kubernetes.io/name` (which exists on pods), not `app.kubernetes.io/component` (which is only on monitoring resources).

- [ ] **Step 6: Verify Helm template renders**

```bash
helm template openoms deploy/helm/openoms/ \
  --set autoscaling.api.enabled=true \
  --set autoscaling.dashboard.enabled=true \
  --show-only templates/api-server/hpa.yaml

helm template openoms deploy/helm/openoms/ \
  --set autoscaling.api.enabled=true \
  --show-only templates/api-server/deployment.yaml | grep -A10 affinity
```

- [ ] **Step 7: Commit public repo**

```bash
git add deploy/helm/openoms/
git commit -m "feat(helm): add HPA and pod anti-affinity for API and Dashboard"
git push -u origin feat/hpa-autoscaling
gh pr create --title "feat(helm): add HPA autoscaling and pod anti-affinity" --body "..."
```

- [ ] **Step 8: Enable in production values (enterprise repo)**

After public PR is merged, update `enterprise/deploy/helm/values-production.yaml`:

```yaml
autoscaling:
  api:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 70
    scaleDownStabilizationSeconds: 300
  dashboard:
    enabled: true
    minReplicas: 2
    maxReplicas: 3
    targetCPUUtilizationPercentage: 70
    scaleDownStabilizationSeconds: 300
```

### Task 12: Resource limits tuning (after 1+ week of metrics)

**GitHub Issue:** openoms-org/openoms-enterprise#18

**Depends on:** 1+ week of metrics data from Phase 0

- [ ] **Step 1: Analyze actual resource usage**

Query Grafana for last 7 days:

```promql
# API CPU usage (P95)
quantile_over_time(0.95, sum(rate(container_cpu_usage_seconds_total{namespace="openoms",container="openoms-api"}[5m]))[7d:1h])

# API memory usage (P95)
quantile_over_time(0.95, container_memory_working_set_bytes{namespace="openoms",container="openoms-api"}[7d:1h])

# Same for dashboard and worker
```

- [ ] **Step 2: Calculate new limits**

Formula: `request = P95 actual usage`, `limit = request * 1.5` (50% headroom)

Compare with current values:
| Component | Current Request | Current Limit |
|-----------|----------------|---------------|
| API | 200m CPU, 256Mi | 1000m CPU, 512Mi |
| Dashboard | 100m CPU, 128Mi | 500m CPU, 256Mi |
| Worker | 100m CPU, 128Mi | 500m CPU, 256Mi |

- [ ] **Step 3: Update values-production.yaml with tuned limits**

Only adjust if actual usage is significantly different (>2x) from current requests.

- [ ] **Step 4: Update GitHub project**

```bash
gh issue close 18 --comment "Resource limits tuned based on 7 days of production metrics data"
gh issue close 19 --comment "HPA configured for API (CPU 70%, min 2, max 5)"
gh issue close 20 --comment "HPA configured for Dashboard (CPU 70%, min 2, max 3)"
gh issue close 21 --comment "Pod anti-affinity added (preferredDuringScheduling, topology: hostname)"
gh issue close 22 --comment "k6 load test suite added (auth flow, orders CRUD)"
```

---

## Chunk 6: Remaining Project Cleanup

### Task 13: Close remaining GitHub project issues

- [ ] **Step 1: External uptime monitoring (#27)**

Evaluate options:
- Grafana Cloud Synthetic Monitoring (built-in)
- UptimeRobot (free tier: 50 monitors)
- Better Uptime

Recommended: Grafana Cloud Synthetic Monitoring — already integrated.
Configure checks: `https://api.openoms.org/health` (1m interval), `https://app.openoms.org` (1m interval).

```bash
gh issue close 27 --comment "External uptime monitoring configured via Grafana Cloud Synthetic Monitoring"
```

- [ ] **Step 2: Log retention policy (#28)**

Document current retention:
- Grafana Cloud Loki: 30 days (default on free/pro tier)
- Prometheus metrics: 13 months (Grafana Cloud default)
- Application audit log: stored in PostgreSQL (no TTL — manual cleanup)

```bash
gh issue close 28 --comment "Log retention: Loki 30d, Prometheus 13mo, audit log in PG (manual). No changes needed."
```

- [ ] **Step 3: Update Epic issues**

```bash
gh issue close 23 --comment "All monitoring & alerting tasks completed"
gh issue close 29 --comment "All security hardening tasks completed"
gh issue close 34 --comment "All backup & DR tasks completed"
gh issue close 17 --comment "All autoscaling & resource management tasks completed"
```
