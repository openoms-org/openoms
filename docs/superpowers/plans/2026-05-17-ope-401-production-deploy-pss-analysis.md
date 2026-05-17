# OPE-401 Production Deploy PSS Analysis Hotfix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the production deploy failure caused by Argo Rollouts smoke-test AnalysisRun pods violating `restricted` PodSecurity.

**Architecture:** The public Helm chart owns the Argo Rollouts `AnalysisTemplate`, so the smoke-test job must render a restricted-compatible pod and container security context. The enterprise deploy runner RBAC issue is tracked in the same Linear issue, but this PR should first unblock the failed deploy by making the generated smoke-test pod admissible.

**Tech Stack:** Helm chart templates, Argo Rollouts AnalysisTemplate, Kubernetes Pod Security Standards, GitHub Actions deploy pipeline.

---

## Scope

- Public repo only for this PR.
- Fix `deploy/helm/openoms/templates/api-server/analysis-template.yaml`.
- Add/adjust Helm chart tests if a chart test fixture exists.
- Update `docs/system-documentation.md` if the deploy behavior needs a durable note.
- Do not touch the DPD OPE-308 worktree or unrelated untracked docs.

## Files

- Modify: `deploy/helm/openoms/templates/api-server/analysis-template.yaml`
  - Add pod-level `seccompProfile.type: RuntimeDefault`.
  - Add explicit container-level `seccompProfile.type: RuntimeDefault` if needed for clear restricted compliance.
  - Keep `runAsNonRoot`, `runAsUser`, `allowPrivilegeEscalation: false`, read-only root filesystem, and dropped capabilities.
- Modify if present: chart/unit test fixture for AnalysisTemplate rendering.
- Modify if useful: `docs/system-documentation.md`
  - Document that pre-promotion smoke AnalysisRuns must remain PSS-restricted compatible.

## Implementation Tasks

### Task 1: Reproduce the Rendered Gap

- [ ] Run:

```bash
helm template openoms deploy/helm/openoms \
  --namespace openoms \
  --set blueGreen.enabled=true \
  --set blueGreen.prePromotionAnalysis=true \
  --show-only templates/api-server/analysis-template.yaml
```

- [ ] Confirm the rendered AnalysisTemplate contains `runAsNonRoot` and container hardening, but no `seccompProfile.type`.

### Task 2: Patch the AnalysisTemplate Security Context

- [ ] In `deploy/helm/openoms/templates/api-server/analysis-template.yaml`, change the job pod security context to:

```yaml
                securityContext:
                  runAsNonRoot: true
                  runAsUser: 65534
                  seccompProfile:
                    type: RuntimeDefault
```

- [ ] In the `smoke` container security context, add:

```yaml
                      seccompProfile:
                        type: RuntimeDefault
```

### Task 3: Validate Rendering

- [ ] Run the same `helm template` command.
- [ ] Confirm the rendered smoke-test job template includes both pod-level and container-level `seccompProfile.type: RuntimeDefault`.
- [ ] Run:

```bash
git diff --check
```

### Task 4: Chart/Repo Validation

- [ ] Run public quick local CI because this is a small chart-only hotfix:

```bash
./scripts/local-ci.sh --quick
```

- [ ] If quick CI passes, run any Helm-focused test command discovered in the repo. If there is no dedicated Helm test, record that Helm rendering plus local CI were used.

### Task 5: Commit and PR

- [ ] Commit:

```bash
git add deploy/helm/openoms/templates/api-server/analysis-template.yaml docs/superpowers/plans/2026-05-17-ope-401-production-deploy-pss-analysis.md
git commit -m "OPE-401: harden rollout smoke analysis pod"
```

- [ ] Push branch `fix/OPE-401-production-deploy-pss-analysis`.
- [ ] Open PR titled `OPE-401: harden rollout smoke analysis pod`.
- [ ] Read CI and CodeRabbit comments before merge.

## Test and Validation Plan

- `helm template ... --show-only templates/api-server/analysis-template.yaml`
- `git diff --check`
- `./scripts/local-ci.sh --quick`
- GitHub PR checks after push.
- After merge, observe public release and enterprise deploy run.

## Risks and Rollback

- Risk: `curlimages/curl` may need writable paths despite read-only root filesystem. It already ran under read-only root before, so this change only adds seccomp and should be low risk.
- Risk: existing degraded rollout may need `kubectl argo rollouts retry` after the fixed chart lands. The enterprise deploy workflow already retries degraded rollouts after Helm upgrade.
- Rollback: revert this commit. No database or persistent-state changes.

## Follow-Up

- Enterprise RBAC should add the `rollouts/status` subresource for `arc-deploy`, because `kubectl argo rollouts abort` failed with `cannot patch resource "rollouts/status"`.
