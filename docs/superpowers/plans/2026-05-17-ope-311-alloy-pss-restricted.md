# OPE-311 Alloy PSS Restricted Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the OpenOMS Helm chart truthful and enforceable for Pod Security Standards by running Alloy without root, hostPath, or added capabilities, while labeling the app namespace as `restricted`.

**Architecture:** Replace node-file log collection with Kubernetes API log tailing via Grafana Alloy `loki.source.kubernetes`, which Grafana documents as working without privileged containers, root users, node filesystem access, or a DaemonSet. Keep kubelet/cAdvisor metrics scraping, but run Alloy as a single restricted Deployment rather than a privileged/root DaemonSet. Add a Helm render guard so future chart changes cannot silently reintroduce an Alloy hostPath/root PSS violation.

**Tech Stack:** Helm, Kubernetes Pod Security Standards, Grafana Alloy, Python standard library render guard.

---

### References

- Kubernetes Pod Security Standards: `https://kubernetes.io/docs/concepts/security/pod-security-standards/`
- Grafana Alloy `loki.source.kubernetes`: `https://grafana.com/docs/alloy/latest/reference/components/loki/loki.source.kubernetes/`

### Task 1: Add A Failing Helm PSS Guard

**Files:**
- Create: `scripts/check-helm-pss.py`
- Modify: `.github/workflows/ci.yml`

- [x] **Step 1: Create the render guard**

Create `scripts/check-helm-pss.py`:

```python
#!/usr/bin/env python3
"""Render the OpenOMS Helm chart and guard Pod Security Standard assumptions."""

from __future__ import annotations

import subprocess
import sys


def render_chart() -> str:
    result = subprocess.run(
        [
            "helm",
            "template",
            "openoms",
            "deploy/helm/openoms",
            "--set",
            "monitoring.enabled=true",
            "--set",
            "networkPolicy.enabled=true",
        ],
        check=False,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if result.returncode != 0:
        sys.stderr.write(result.stderr)
        raise SystemExit(result.returncode)
    return result.stdout


def documents(rendered: str) -> list[str]:
    return [doc.strip() for doc in rendered.split("---") if doc.strip()]


def find_doc(rendered_docs: list[str], *, kind: str, name: str) -> str:
    for doc in rendered_docs:
        if f"kind: {kind}" in doc and f"name: {name}" in doc:
            return doc
    raise AssertionError(f"missing {kind}/{name}")


def assert_contains(doc: str, needle: str, context: str) -> None:
    if needle not in doc:
        raise AssertionError(f"{context}: missing {needle!r}")


def assert_not_contains(doc: str, needle: str, context: str) -> None:
    if needle in doc:
        raise AssertionError(f"{context}: forbidden {needle!r}")


def main() -> None:
    rendered_docs = documents(render_chart())

    namespace = find_doc(rendered_docs, kind="Namespace", name="openoms")
    for label in (
        "pod-security.kubernetes.io/enforce: restricted",
        "pod-security.kubernetes.io/audit: restricted",
        "pod-security.kubernetes.io/warn: restricted",
    ):
        assert_contains(namespace, label, "Namespace/openoms")

    alloy = find_doc(rendered_docs, kind="Deployment", name="openoms-alloy")
    for forbidden in (
        "kind: DaemonSet",
        "hostPath:",
        "runAsNonRoot: false",
        "runAsUser: 0",
        "DAC_READ_SEARCH",
        "/var/log",
        "/var/lib/docker/containers",
    ):
        assert_not_contains(alloy, forbidden, "Deployment/openoms-alloy")

    for required in (
        "runAsNonRoot: true",
        "runAsUser: 65534",
        "runAsGroup: 65534",
        "fsGroup: 65534",
        "allowPrivilegeEscalation: false",
        "readOnlyRootFilesystem: true",
        'drop: ["ALL"]',
        "seccompProfile:",
        "type: RuntimeDefault",
    ):
        assert_contains(alloy, required, "Deployment/openoms-alloy")


if __name__ == "__main__":
    try:
        main()
    except AssertionError as exc:
        print(f"Helm PSS check failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
```

- [x] **Step 2: Run RED**

Run:

```bash
python3 scripts/check-helm-pss.py
```

Expected: FAIL on the current chart because the namespace lacks `pod-security.kubernetes.io/enforce: restricted` and Alloy is still rendered as a root DaemonSet with hostPath mounts and `DAC_READ_SEARCH`.

Result: failed as expected with `Helm PSS check failed: Namespace/openoms: missing 'pod-security.kubernetes.io/enforce: restricted'`.

- [x] **Step 3: Add the guard to CI Helm Chart job**

In `.github/workflows/ci.yml`, add after `helm template openoms deploy/helm/openoms > /dev/null`:

```yaml
      - name: Check Helm Pod Security Standards
        run: python3 scripts/check-helm-pss.py
```

### Task 2: Make Namespace PSS Explicit

**Files:**
- Modify: `deploy/helm/openoms/values.yaml`
- Modify: `deploy/helm/openoms/templates/namespace.yaml`

- [x] **Step 1: Add pod security values**

Add to `deploy/helm/openoms/values.yaml` near `namespace`:

```yaml
podSecurity:
  enforce: restricted
  audit: restricted
  warn: restricted
  version: latest
```

- [x] **Step 2: Render namespace labels from values**

Update `deploy/helm/openoms/templates/namespace.yaml` labels:

```yaml
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
    pod-security.kubernetes.io/enforce: {{ .Values.podSecurity.enforce | default "restricted" | quote }}
    pod-security.kubernetes.io/audit: {{ .Values.podSecurity.audit | default "restricted" | quote }}
    pod-security.kubernetes.io/warn: {{ .Values.podSecurity.warn | default "restricted" | quote }}
    pod-security.kubernetes.io/enforce-version: {{ .Values.podSecurity.version | default "latest" | quote }}
    pod-security.kubernetes.io/audit-version: {{ .Values.podSecurity.version | default "latest" | quote }}
    pod-security.kubernetes.io/warn-version: {{ .Values.podSecurity.version | default "latest" | quote }}
```

### Task 3: Convert Alloy To Restricted Deployment

**Files:**
- Delete: `deploy/helm/openoms/templates/monitoring/alloy-daemonset.yaml`
- Create: `deploy/helm/openoms/templates/monitoring/alloy-deployment.yaml`
- Modify: `deploy/helm/openoms/templates/monitoring/alloy-configmap.yaml`

- [x] **Step 1: Replace the DaemonSet with a Deployment**

Create `deploy/helm/openoms/templates/monitoring/alloy-deployment.yaml`:

```yaml
{{- if and .Values.monitoring .Values.monitoring.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "openoms.fullname" . }}-alloy
  namespace: {{ .Values.namespace }}
  labels:
    {{- include "openoms.labels" . | nindent 4 }}
    app.kubernetes.io/component: monitoring
spec:
  replicas: {{ .Values.monitoring.alloy.replicaCount | default 1 }}
  selector:
    matchLabels:
      app.kubernetes.io/component: alloy
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/monitoring/alloy-configmap.yaml") . | sha256sum }}
      labels:
        app.kubernetes.io/component: alloy
    spec:
      serviceAccountName: {{ include "openoms.fullname" . }}-alloy
      automountServiceAccountToken: true
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        runAsGroup: 65534
        fsGroup: 65534
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: alloy
          image: {{ .Values.monitoring.alloy.image.repository | default "grafana/alloy" }}:{{ .Values.monitoring.alloy.image.tag | default "v1.8.2" }}
          imagePullPolicy: {{ .Values.monitoring.alloy.image.pullPolicy | default "IfNotPresent" }}
          args:
            - run
            - /etc/alloy/config.alloy
            - --storage.path=/tmp/alloy
          ports:
            - name: http
              containerPort: 12345
              protocol: TCP
          env:
            - name: HOSTNAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: METRICS_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.secretName" . }}
                  key: METRICS_TOKEN
                  optional: true
            - name: GRAFANA_REMOTE_WRITE_URL
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: GRAFANA_REMOTE_WRITE_URL
            - name: GRAFANA_USER
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: GRAFANA_USER
            - name: GRAFANA_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: GRAFANA_TOKEN
            - name: LOKI_URL
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: LOKI_URL
                  optional: true
            - name: LOKI_USER
              valueFrom:
                secretKeyRef:
                  name: {{ include "openoms.fullname" . }}-monitoring
                  key: LOKI_USER
                  optional: true
          resources:
            {{- toYaml .Values.monitoring.alloy.resources | nindent 12 }}
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            capabilities:
              drop: ["ALL"]
          volumeMounts:
            - name: config
              mountPath: /etc/alloy
              readOnly: true
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: config
          configMap:
            name: {{ include "openoms.fullname" . }}-alloy-config
        - name: tmp
          emptyDir:
            sizeLimit: 100Mi
{{- end }}
```

Delete `deploy/helm/openoms/templates/monitoring/alloy-daemonset.yaml`.

- [x] **Step 2: Add Alloy defaults**

Add under `monitoring:` in `deploy/helm/openoms/values.yaml`:

```yaml
monitoring:
  enabled: false
  alloy:
    replicaCount: 1
    image:
      repository: grafana/alloy
      tag: v1.8.2
      pullPolicy: IfNotPresent
    resources:
      requests:
        cpu: 50m
        memory: 256Mi
      limits:
        cpu: 200m
        memory: 384Mi
```

- [x] **Step 3: Switch log collection to Kubernetes API tailing**

In `deploy/helm/openoms/templates/monitoring/alloy-configmap.yaml`:

1. Remove the `__path__` relabel rule that writes `/var/log/pods/...`.
2. Replace:

```alloy
    local.file_match "pod_logs" {
      path_targets = discovery.relabel.pod_logs.output
    }

    loki.source.file "pod_logs" {
      targets    = local.file_match.pod_logs.targets
      forward_to = [loki.process.pod_logs.receiver]
    }
```

with:

```alloy
    loki.source.kubernetes "pod_logs" {
      targets    = discovery.relabel.pod_logs.output
      forward_to = [loki.process.pod_logs.receiver]
    }
```

### Task 4: Validate, Commit, PR

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `deploy/helm/openoms/values.yaml`
- Modify: `deploy/helm/openoms/templates/namespace.yaml`
- Modify: `deploy/helm/openoms/templates/monitoring/alloy-configmap.yaml`
- Create: `deploy/helm/openoms/templates/monitoring/alloy-deployment.yaml`
- Delete: `deploy/helm/openoms/templates/monitoring/alloy-daemonset.yaml`
- Create: `scripts/check-helm-pss.py`
- Create: `docs/superpowers/plans/2026-05-17-ope-311-alloy-pss-restricted.md`

- [x] **Step 1: Run GREEN validation**

Run:

```bash
python3 scripts/check-helm-pss.py
helm lint deploy/helm/openoms
helm template openoms deploy/helm/openoms --set monitoring.enabled=true --set networkPolicy.enabled=true >/tmp/openoms-helm-ope311.yaml
rg -n "kind: (DaemonSet|Deployment)|openoms-alloy|hostPath|runAsUser: 0|DAC_READ_SEARCH|pod-security.kubernetes.io" /tmp/openoms-helm-ope311.yaml
git diff --check
```

Expected:
- PSS guard passes.
- Helm lint/template pass.
- Rendered chart has `Namespace/openoms` with PSS restricted labels.
- Rendered Alloy is `Deployment/openoms-alloy`.
- Rendered Alloy has no `hostPath`, `runAsUser: 0`, or `DAC_READ_SEARCH`.

Result: `python3 scripts/check-helm-pss.py`, `helm lint deploy/helm/openoms`, and rendered chart grep passed. Rendered manifest shows PSS restricted namespace labels and `Deployment/openoms-alloy`; no `hostPath`, `runAsUser: 0`, or `DAC_READ_SEARCH` appeared.

- [ ] **Step 2: Run public local CI before push**

Run:

```bash
./scripts/local-ci.sh
```

Expected: full local CI passes for the clean current HEAD after commit.

- [ ] **Step 3: Commit**

Run:

```bash
git add .github/workflows/ci.yml deploy/helm/openoms scripts/check-helm-pss.py docs/superpowers/plans/2026-05-17-ope-311-alloy-pss-restricted.md
git commit -m "OPE-311: run Alloy within restricted pod security"
```

- [ ] **Step 4: Push and open PR**

Run:

```bash
git push -u origin fix/OPE-311-alloy-pss-restricted
```

Open PR title:

```text
OPE-311: run Alloy within restricted pod security
```

PR body must include:

```markdown
## Docs updated
- [x] docs/superpowers/plans/2026-05-17-ope-311-alloy-pss-restricted.md — implementation and validation plan
- [x] docs/system-documentation.md — Pod Security Standards/monitoring note
```

### Risk And Rollback

- Risk: Kubernetes API log tailing uses more API/kubelet network and CPU than node file tailing. Mitigation: keep Alloy at one replica by default and monitor Loki/API errors after deploy.
- Risk: switching from DaemonSet to Deployment may briefly recreate the Alloy workload during Helm upgrade. App serving path is unaffected.
- Rollback: revert this PR to restore DaemonSet file tailing, or temporarily disable `monitoring.enabled` in enterprise values if Alloy blocks deploy. Do not remove namespace PSS labels without a follow-up issue documenting the exception.

### Self-Review

- Spec coverage: OPE-311 is covered by explicit namespace PSS labels, removal of root/hostPath/capability Alloy DaemonSet behavior, and a CI guard.
- Placeholder scan: no placeholders or TODOs remain.
- Type/config consistency: `.Values.monitoring.alloy.*` defaults are defined before template usage.
