# OPE-402 NetworkPolicy Egress Controls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add configurable destination-scoped egress controls for OpenOMS Helm NetworkPolicies while keeping safe self-hosted defaults.

**Architecture:** The public chart keeps backward-compatible egress behavior by default, but each egress class can now specify Kubernetes NetworkPolicy `to` destinations such as `namespaceSelector`, `podSelector`, or `ipBlock`. Enterprise staging/production values then constrain DNS, PostgreSQL, and Redis to known in-cluster services, while HTTPS remains allowed on port 443 for dynamic provider, S3, Sentry, and SaaS endpoints.

**Tech Stack:** Helm, Kubernetes `networking.k8s.io/v1` NetworkPolicy, public OpenOMS chart, enterprise Helm values overlays.

---

## Scope

- Public chart change in `public/deploy/helm/openoms`.
- Enterprise overlay change in `enterprise/deploy/helm/values-staging.yaml` and `enterprise/deploy/helm/values-production.yaml`.
- Documentation updates in `public/docs/system-documentation.md` and `public/.claude/context/SECURITY_POSTURE.md`.
- No live `kubectl apply`, no Helm release against production/staging from this task.

## Design

- Add `networkPolicy.egress.<class>.to` lists:
  - `dns.to`
  - `https.to`
  - `database.to`
  - `redis.to`
- Leave every default `to: []` in the public chart. In Kubernetes NetworkPolicy, an egress rule without `to` preserves current destination behavior, so self-hosted users are not broken.
- Add `networkPolicy.egress.<class>.ports` lists for explicit port/protocol customization. Defaults match existing behavior.
- Enterprise overlays set:
  - DNS to `kube-system`/`kube-dns` pods.
  - Database to `apps-core` PostgreSQL pods.
  - Redis to `apps-core` Redis pods.
  - HTTPS remains destination-unscoped on TCP 443 because OpenOMS must call multiple external provider APIs and S3 endpoints whose CIDRs are not stable.
- Keep intra-namespace egress unchanged.

## Files

- Modify: `public/deploy/helm/openoms/values.yaml`
- Modify: `public/deploy/helm/openoms/templates/networkpolicy.yaml`
- Modify: `public/docs/system-documentation.md`
- Modify: `public/.claude/context/SECURITY_POSTURE.md`
- Modify: `enterprise/deploy/helm/values-staging.yaml`
- Modify: `enterprise/deploy/helm/values-production.yaml`

## Implementation Steps

- [x] Update public chart values.
  - Add `networkPolicy.egress` defaults for DNS, HTTPS, database, Redis.
  - Keep defaults backward compatible.

- [x] Update public NetworkPolicy template.
  - Render ports from values instead of hardcoding.
  - Render `to:` only when the corresponding list is non-empty.
  - Keep existing ingress and intra-namespace behavior.

- [x] Update public documentation.
  - Document that chart defaults are broad for self-hosting.
  - Document that production/staging overlays should scope DB/Redis/DNS destinations.

- [x] Update enterprise values.
  - Add DNS destination selectors for CoreDNS.
  - Add PostgreSQL and Redis destination selectors for `apps-core`.
  - Keep HTTPS unscoped and document why.

- [x] Validate public chart.
  - Run `helm lint deploy/helm/openoms`.
  - Run `helm template openoms deploy/helm/openoms --set networkPolicy.enabled=true`.
  - Run `helm template openoms deploy/helm/openoms -f <enterprise>/deploy/helm/values-staging.yaml`.
  - Run `helm template openoms deploy/helm/openoms -f <enterprise>/deploy/helm/values-production.yaml`.
  - Run `git diff --check`.

- [x] Validate enterprise overlay.
  - From `enterprise`, run `git diff --check`.
  - Ensure enterprise CI can still template the public chart after the public PR is merged.

## Risk And Rollback

- Risk: a wrong selector can block DNS, DB, or Redis egress after deployment.
- Mitigation: the public chart defaults remain broad; only enterprise overlays tighten known in-cluster destinations.
- Rollback: remove the enterprise `networkPolicy.egress.*.to` lists or set them to `[]` and redeploy with the same chart version.
- Live rollout requires separate explicit approval naming the kube context and namespace.

## Out Of Scope

- Restricting external HTTPS by provider CIDR.
- Live cluster mutation.
- Reworking enterprise hand-written namespace NetworkPolicies.
- Provider-specific egress gateways or Cilium policies.

## Validation Evidence

- `helm lint deploy/helm/openoms`
- `helm lint deploy/helm/openoms -f <enterprise>/deploy/helm/values-staging.yaml`
- `helm lint deploy/helm/openoms -f <enterprise>/deploy/helm/values-production.yaml`
- `helm template openoms deploy/helm/openoms --set networkPolicy.enabled=true`
- `helm template openoms deploy/helm/openoms -f <enterprise>/deploy/helm/values-staging.yaml`
- `helm template openoms deploy/helm/openoms -f <enterprise>/deploy/helm/values-production.yaml`
- `git diff --check` in `public`
- `git diff --check` in `enterprise`

Read-only cluster label checks confirmed:

- CoreDNS: `k8s-app=kube-dns`
- PostgreSQL: `app.kubernetes.io/name=postgresql`
- Redis: `app.kubernetes.io/name=redis`
