# OPE-268 PreStop Hook Warnings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove deploy-time `FailedPreStopHook` warnings from OpenOMS API and dashboard rollouts without changing blue/green safety semantics.

**Architecture:** Replace image-local lifecycle exec sleeps with kubelet-managed `preStop.sleep` hooks when the target Kubernetes cluster supports them. Kubernetes runs `sleep` lifecycle handlers in kubelet rather than inside the container, so distroless images do not need `/bin/sh`, `sleep`, or a reliable `node` binary on `PATH`. For older Kubernetes versions, omit the sleep hook rather than emitting an invalid field or an exec hook that cannot run in distroless images; graceful SIGTERM shutdown remains in the application and the existing termination grace period.

**Tech Stack:** Helm, Kubernetes lifecycle hooks, Argo Rollouts blue/green templates, GitHub Actions Helm chart validation.

---

## Context And Findings

Current chart state:

- `deploy/helm/openoms/templates/api-server/deployment.yaml` and `deploy/helm/openoms/templates/api-server/rollout.yaml` use:

  ```yaml
  lifecycle:
    preStop:
      exec:
        command: ["sh", "-c", "sleep 5"]
  ```

- `apps/api-server/Dockerfile` uses `gcr.io/distroless/static-debian12`, which has no shell.
- `deploy/helm/openoms/templates/dashboard/deployment.yaml` and `deploy/helm/openoms/templates/dashboard/rollout.yaml` use:

  ```yaml
  lifecycle:
    preStop:
      exec:
        command: ["node", "-e", "setTimeout(() => process.exit(0), 5000)"]
  ```

- `apps/dashboard/Dockerfile` uses `gcr.io/distroless/nodejs22-debian13:nonroot`; this avoids a shell but still depends on a runtime exec lookup during termination.
- API already handles SIGTERM with `http.Server.Shutdown(ctx)` and a 10 second timeout in `apps/api-server/cmd/server/main.go`.
- Kubernetes docs state that `exec` hooks run inside the container, while `sleep` hooks are executed by kubelet. This is the right model for distroless images.

## Files And Responsibilities

- Modify `deploy/helm/openoms/values.yaml`
  - Add lifecycle values for API and dashboard termination drain seconds.
  - Document the Kubernetes version behavior in comments.

- Modify `deploy/helm/openoms/templates/_helpers.tpl`
  - Add a helper that renders a `lifecycle.preStop.sleep.seconds` block only when:
    - configured seconds are greater than zero,
    - `.Capabilities.KubeVersion.Version` is `>= 1.30.0-0`.
  - Keep all version branching in one helper so API and dashboard templates cannot drift.

- Modify `deploy/helm/openoms/templates/api-server/deployment.yaml`
- Modify `deploy/helm/openoms/templates/api-server/rollout.yaml`
- Modify `deploy/helm/openoms/templates/dashboard/deployment.yaml`
- Modify `deploy/helm/openoms/templates/dashboard/rollout.yaml`
  - Replace inline `exec` lifecycle hooks with the shared helper.

- Modify `.github/workflows/ci.yml`
  - Add Helm validation that renders the chart with Kubernetes `1.34.0` and asserts:
    - API/dashboard lifecycle hooks use `preStop.sleep`,
    - rendered chart has no `preStop.exec` hooks.

- Optional docs update:
  - Update `docs/scaling-roadmap.md` only if the existing preStop checklist needs factual correction after implementation.
  - `docs/system-documentation.md` is not required unless the rollout/shutdown behavior is described there today.

## Task 1: Add Configurable Kubelet-Managed PreStop Sleep

**Files:**

- Modify: `deploy/helm/openoms/values.yaml`
- Modify: `deploy/helm/openoms/templates/_helpers.tpl`

- [ ] **Step 1: Add values for termination drain seconds**

  In `deploy/helm/openoms/values.yaml`, add `terminationDrainSeconds: 5` under both `apiServer:` and `dashboard:`:

  ```yaml
  apiServer:
    terminationDrainSeconds: 5
  ```

  ```yaml
  dashboard:
    terminationDrainSeconds: 5
  ```

  Add a short comment near the first value:

  ```yaml
  # Rendered as Kubernetes preStop.sleep on clusters >= 1.30.
  # Older clusters omit the lifecycle hook because distroless images have no shell/sleep binary.
  terminationDrainSeconds: 5
  ```

- [ ] **Step 2: Add a shared lifecycle helper**

  Append this helper to `deploy/helm/openoms/templates/_helpers.tpl`:

  ```gotemplate
  {{/*
  Render a kubelet-managed preStop sleep hook for distroless images.
  Kubernetes runs lifecycle sleep hooks in kubelet, unlike exec hooks which
  require binaries inside the container image. The sleep action is enabled by
  default from Kubernetes 1.30, so older clusters omit the hook.
  */}}
  {{- define "openoms.lifecycle.preStopSleep" -}}
  {{- $root := .root -}}
  {{- $seconds := int (default 0 .seconds) -}}
  {{- if and (gt $seconds 0) (semverCompare ">=1.30.0-0" $root.Capabilities.KubeVersion.Version) }}
  lifecycle:
    preStop:
      sleep:
        seconds: {{ $seconds }}
  {{- end }}
  {{- end }}
  ```

- [ ] **Step 3: Run a targeted template check**

  Run:

  ```bash
  helm template openoms deploy/helm/openoms --kube-version 1.34.0 >/tmp/openoms-helm-134.yaml
  grep -n "preStop:" /tmp/openoms-helm-134.yaml
  grep -n "sleep:" /tmp/openoms-helm-134.yaml
  ```

  Expected:

  - rendered chart contains `preStop:` entries for API and dashboard,
  - rendered chart contains `sleep:` entries,
  - no `preStop.exec` command is present after templates are updated in Task 2.

## Task 2: Replace Inline Exec Hooks In API And Dashboard Templates

**Files:**

- Modify: `deploy/helm/openoms/templates/api-server/deployment.yaml`
- Modify: `deploy/helm/openoms/templates/api-server/rollout.yaml`
- Modify: `deploy/helm/openoms/templates/dashboard/deployment.yaml`
- Modify: `deploy/helm/openoms/templates/dashboard/rollout.yaml`

- [ ] **Step 1: Replace API deployment lifecycle block**

  Replace this block:

  ```yaml
          lifecycle:
            preStop:
              exec:
                command: ["sh", "-c", "sleep 5"]
  ```

  with:

  ```gotemplate
          {{- include "openoms.lifecycle.preStopSleep" (dict "root" . "seconds" .Values.apiServer.terminationDrainSeconds) | nindent 10 }}
  ```

- [ ] **Step 2: Replace API rollout lifecycle block**

  Make the same replacement in `deploy/helm/openoms/templates/api-server/rollout.yaml`.

- [ ] **Step 3: Replace dashboard deployment lifecycle block**

  Replace this block:

  ```yaml
          lifecycle:
            preStop:
              exec:
                command: ["node", "-e", "setTimeout(() => process.exit(0), 5000)"]
  ```

  with:

  ```gotemplate
          {{- include "openoms.lifecycle.preStopSleep" (dict "root" . "seconds" .Values.dashboard.terminationDrainSeconds) | nindent 10 }}
  ```

- [ ] **Step 4: Replace dashboard rollout lifecycle block**

  Make the same replacement in `deploy/helm/openoms/templates/dashboard/rollout.yaml`.

- [ ] **Step 5: Render both supported behavior branches**

  Run:

  ```bash
  helm template openoms deploy/helm/openoms --kube-version 1.34.0 >/tmp/openoms-helm-134.yaml
  helm template openoms deploy/helm/openoms --kube-version 1.29.0 >/tmp/openoms-helm-129.yaml
  ```

  Expected:

  - `1.34.0` render contains `preStop.sleep.seconds: 5`,
  - `1.29.0` render does not contain a lifecycle `preStop.exec` fallback,
  - both renders exit 0.

## Task 3: Add CI Guard For Distroless Lifecycle Hooks

**Files:**

- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Add Helm lifecycle validation step**

  In the `helm-chart` job, after `Render chart`, add:

  ```yaml
      - name: Check distroless lifecycle hooks
        if: steps.filter.outputs.chart == 'true'
        run: |
          set -euo pipefail
          rendered="$(mktemp)"
          helm template openoms deploy/helm/openoms --kube-version 1.34.0 > "$rendered"

          if ! grep -q 'preStop:' "$rendered"; then
            echo "Expected preStop hooks in Kubernetes 1.34 render" >&2
            exit 1
          fi

          if ! grep -q 'sleep:' "$rendered"; then
            echo "Expected kubelet-managed preStop.sleep hooks in Kubernetes 1.34 render" >&2
            exit 1
          fi

          if grep -q 'command:.*sleep 5\|command:.*sh\|preStop:.*exec:' "$rendered"; then
            echo "Distroless preStop hooks must not depend on shell/node exec sleeps" >&2
            exit 1
          fi
  ```

- [ ] **Step 2: Confirm existing Helm validation still runs**

  Run locally if `helm` is available:

  ```bash
  helm lint deploy/helm/openoms
  helm template openoms deploy/helm/openoms >/dev/null
  helm template openoms deploy/helm/openoms --kube-version 1.34.0 >/tmp/openoms-helm-134.yaml
  ```

  If `helm` is not available locally, rely on GitHub CI for Helm validation and record that limitation in the PR test plan.

## Task 4: Documentation And Verification

**Files:**

- Modify: `docs/scaling-roadmap.md` only if needed.

- [ ] **Step 1: Update docs only if current wording is stale**

  If `docs/scaling-roadmap.md` still lists preStop as missing or shell-based, update that line to state:

  ```markdown
  - [x] preStop hook + graceful shutdown: API handles SIGTERM with `http.Server.Shutdown`; Helm uses kubelet-managed `preStop.sleep` on Kubernetes >= 1.30 for distroless-safe termination drain.
  ```

- [ ] **Step 2: Run repository validation**

  Run:

  ```bash
  git diff --check
  ./scripts/local-ci.sh --quick
  ```

  If Helm is available locally, also run:

  ```bash
  helm lint deploy/helm/openoms
  helm template openoms deploy/helm/openoms >/dev/null
  helm template openoms deploy/helm/openoms --kube-version 1.34.0 >/tmp/openoms-helm-134.yaml
  helm template openoms deploy/helm/openoms --kube-version 1.29.0 >/tmp/openoms-helm-129.yaml
  ```

- [ ] **Step 3: Commit**

  Commit with:

  ```bash
  git add .github/workflows/ci.yml deploy/helm/openoms docs/scaling-roadmap.md
  git commit -m "OPE-268: use distroless-safe prestop hooks"
  ```

## Risk And Rollback

- Risk: Kubernetes clusters older than 1.30 will omit the drain sleep. Mitigation: they still receive SIGTERM graceful shutdown and `terminationGracePeriodSeconds`; this is safer than rendering a hook that fails on distroless images.
- Risk: Helm semver comparison could render differently under local `helm template` default capabilities. Mitigation: CI explicitly renders with `--kube-version 1.34.0` and the standard render path.
- Risk: Production rollout still emits old warning events from old ReplicaSets. Mitigation: validation must check event timestamps after the deploy, not historical events.
- Rollback: revert the PR; the previous lifecycle hooks return. If a production issue appears, `helm rollback openoms <previous-revision> -n openoms` restores the prior chart release.

## Post-Merge Validation

- Watch the production deploy triggered by public `main`.
- After deploy, check:

  ```bash
  kubectl --kubeconfig ~/.kube/openoms-config --context openoms-vpn -n openoms get pods
  kubectl --kubeconfig ~/.kube/openoms-config --context openoms-vpn -n openoms get events --sort-by=.lastTimestamp | grep -E 'FailedPreStopHook|Startup probe failed|openoms-(api|dashboard)' | tail -40
  ```

- Expected:
  - new API/dashboard pods are Ready,
  - restart counts stay 0,
  - no fresh `FailedPreStopHook` events appear for the rollout timestamp.

## Self-Review

- Spec coverage: verifies current distroless mismatch, replaces failing hooks, keeps blue/green rollout templates covered, and includes staging/production validation expectations.
- Placeholder scan: no `TBD`, no open-ended TODOs, no undefined test commands.
- Type consistency: value key is consistently `terminationDrainSeconds`; helper name is consistently `openoms.lifecycle.preStopSleep`.
