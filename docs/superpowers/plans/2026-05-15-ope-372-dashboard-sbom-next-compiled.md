# OPE-372 Dashboard SBOM Next Compiled Components Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop false HIGH/CRITICAL Dependency-Track alerts for the dashboard caused by Next.js vendored `dist/compiled` package metadata with `UNKNOWN` versions, while preserving runtime SBOM coverage for real installed packages.

**Architecture:** Keep release SBOM generation from the shipped Docker image, but exclude Next.js vendored compiled package metadata paths that Syft catalogs as standalone npm packages with `version: UNKNOWN`. Add a validation guard so dashboard SBOM artifacts fail release if any `pkg:npm/*` component has an unknown version from `/app/node_modules/next/dist/compiled/**/package.json`. Reimport the corrected SBOM after the release path is green.

**Tech Stack:** GitHub Actions release workflow, Syft 1.44.0, CycloneDX JSON, Next.js standalone image, Dependency-Track import workflow.

---

## Scope And Boundaries

Primary public repo change:

- `public/.github/workflows/release.yml`
- `public/scripts/check-sbom-artifacts.sh`
- `public/scripts/test-sbom-artifacts.sh`
- `public/docs/system-documentation.md`
- this plan file

Expected enterprise operational follow-up after public PR:

- reimport corrected SBOM artifact into Dependency-Track,
- cancel/close duplicate Linear tickets that came from the same false-positive pattern,
- keep `OPE-393` in mind because production deploys are currently blocked by pre-deploy backup upload failure.

This plan does not suppress vulnerabilities blindly in Dependency-Track. The source SBOM must first stop emitting misleading `UNKNOWN` npm components.

## Evidence

Release `1df9c34332459ea7550eec2a980626836cae8548` dashboard SBOM contains these components:

```text
jsonwebtoken@UNKNOWN pkg:npm/jsonwebtoken
tar@UNKNOWN pkg:npm/tar
http-proxy@UNKNOWN pkg:npm/http-proxy
picomatch@UNKNOWN pkg:npm/picomatch
ws@UNKNOWN pkg:npm/ws
fresh@UNKNOWN pkg:npm/fresh
zod@UNKNOWN pkg:npm/zod
debug@UNKNOWN pkg:npm/debug
semver@UNKNOWN pkg:npm/semver
```

Every `UNKNOWN` component came from:

```text
/app/node_modules/next/dist/compiled/**/package.json
```

The actual standalone runtime `node_modules` does not contain independent `jsonwebtoken`, `tar`, `http-proxy`, `ws`, `fresh`, `zod`, `picomatch`, `shadcn`, `vitest`, `eslint`, or `happy-dom` packages. The actual installed `debug@4.4.3` and `semver@7.7.4` packages remain in SBOM with concrete versions.

Local proof command:

```bash
syft registry:ghcr.io/openoms-org/openoms-dashboard@sha256:fb0603b265202123413a14600fca823182975f3ef50f6cb3a20d57c4f9e19497 \
  --source-name openoms-dashboard \
  --source-version 1df9c34332459ea7550eec2a980626836cae8548 \
  --exclude '/app/node_modules/next/dist/compiled/**/package.json' \
  -o cyclonedx-json=/tmp/openoms-dashboard-exclude-next-compiled.cdx.json
```

Result: unknown npm components drop from `46` to `0`; concrete runtime packages remain.

## Implementation Tasks

### Task 1: Create Work Branch

**Files:**
- No file edits.

- [ ] **Step 1: Check public repo status**

```bash
cd /Users/rafs/praca/openoms-dev/public
git status --short --branch
```

Expected:

```text
## main...origin/main
```

- [ ] **Step 2: Create branch**

```bash
git switch -c fix/OPE-372-dashboard-sbom-next-compiled
```

### Task 2: Exclude Misleading Next Compiled Package Metadata From Dashboard SBOM

**Files:**
- Modify: `/Users/rafs/praca/openoms-dev/public/.github/workflows/release.yml`

- [ ] **Step 1: Update dashboard SBOM generation**

In the `Generate Dashboard SBOM` step, change:

```bash
syft "docker:${DASHBOARD_IMAGE}:${GITHUB_SHA}" \
  --source-name openoms-dashboard \
  --source-version "${GITHUB_SHA}" \
  -o "cyclonedx-json=sbom/openoms-dashboard-${GITHUB_SHA}.cdx.json"
```

to:

```bash
syft "docker:${DASHBOARD_IMAGE}:${GITHUB_SHA}" \
  --source-name openoms-dashboard \
  --source-version "${GITHUB_SHA}" \
  --exclude '/app/node_modules/next/dist/compiled/**/package.json' \
  -o "cyclonedx-json=sbom/openoms-dashboard-${GITHUB_SHA}.cdx.json"
```

Only apply this to the dashboard image. Do not change API or migrate SBOM generation.

### Task 3: Add SBOM Guard For Unknown Dashboard NPM Components

**Files:**
- Modify: `/Users/rafs/praca/openoms-dev/public/scripts/check-sbom-artifacts.sh`
- Modify: `/Users/rafs/praca/openoms-dev/public/scripts/test-sbom-artifacts.sh`

- [ ] **Step 1: Extend `check-sbom-artifacts.sh` validation**

Inside `validate_sbom_json`, after the existing `components` list check, add Python validation:

```python
if expected_component == "openoms-dashboard":
    unknown_npm = []
    for item in components:
        purl = str(item.get("purl") or "")
        version = str(item.get("version") or "").strip()
        if purl.startswith("pkg:npm/") and (not version or version == "UNKNOWN"):
            locations = [
                str(prop.get("value") or "")
                for prop in item.get("properties", [])
                if str(prop.get("name") or "").startswith("syft:location")
            ]
            unknown_npm.append((item.get("name"), purl, locations))
    if unknown_npm:
        details = "; ".join(
            f"{name} ({purl}) at {', '.join(locations) or 'unknown location'}"
            for name, purl, locations in unknown_npm[:10]
        )
        raise SystemExit(f"{path}: dashboard SBOM contains npm components with unknown versions: {details}")
```

Expected: any future Syft change that reintroduces unknown npm components fails release before publishing/import.

- [ ] **Step 2: Add test coverage for the guard**

In `scripts/test-sbom-artifacts.sh`, add a test fixture that writes a valid dashboard SBOM with one component:

```json
{
  "type": "library",
  "name": "jsonwebtoken",
  "version": "UNKNOWN",
  "purl": "pkg:npm/jsonwebtoken",
  "properties": [
    {
      "name": "syft:location:0:path",
      "value": "/app/node_modules/next/dist/compiled/jsonwebtoken/package.json"
    }
  ]
}
```

Expected: `scripts/check-sbom-artifacts.sh` exits non-zero and stderr includes `dashboard SBOM contains npm components with unknown versions`.

### Task 4: Documentation

**Files:**
- Modify: `/Users/rafs/praca/openoms-dev/public/docs/system-documentation.md`

- [ ] **Step 1: Update SBOM docs**

In the release/SBOM documentation section, add a short factual note:

```text
Dashboard SBOM generation excludes Next.js vendored `node_modules/next/dist/compiled/**/package.json` metadata files because they describe bundled internal packages without versions. The release guard fails if the dashboard SBOM contains npm components with missing or `UNKNOWN` versions.
```

### Task 5: Local Validation

**Files:**
- Validate public repo changes.

- [ ] **Step 1: Shell syntax**

```bash
cd /Users/rafs/praca/openoms-dev/public
bash -n scripts/check-sbom-artifacts.sh scripts/test-sbom-artifacts.sh
```

- [ ] **Step 2: SBOM script tests**

```bash
scripts/test-sbom-artifacts.sh
```

- [ ] **Step 3: Current release artifact regression**

Use the downloaded dashboard SBOM artifact from release `1df9c34332459ea7550eec2a980626836cae8548` and verify the guard fails before the exclusion, then verify a generated SBOM with the Syft exclusion has no unknown npm components:

```bash
node - <<'NODE'
const fs = require("fs");
for (const p of [
  "/tmp/openoms-sbom-1df9c343/openoms-dashboard-1df9c34332459ea7550eec2a980626836cae8548.cdx.json",
  "/tmp/openoms-dashboard-exclude-next-compiled.cdx.json",
]) {
  const bom = JSON.parse(fs.readFileSync(p, "utf8"));
  const unknown = (bom.components || []).filter(
    (c) => String(c.purl || "").startsWith("pkg:npm/") && (!c.version || c.version === "UNKNOWN"),
  );
  console.log(`${p}: unknown npm components=${unknown.length}`);
}
NODE
```

Expected:

```text
current release SBOM: unknown npm components=46
excluded SBOM: unknown npm components=0
```

- [ ] **Step 4: Dashboard dependency audit**

```bash
cd apps/dashboard
npm audit --audit-level=high
```

Expected: no high or critical npm audit findings.

- [ ] **Step 5: Public local CI**

Before pushing:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: full local CI passes on a clean HEAD.

### Task 6: PR And Operational Follow-Up

**Files:**
- Commit planned public changes only.

- [ ] **Step 1: Commit**

```bash
git add .github/workflows/release.yml scripts/check-sbom-artifacts.sh scripts/test-sbom-artifacts.sh docs/system-documentation.md docs/superpowers/plans/2026-05-15-ope-372-dashboard-sbom-next-compiled.md
git commit -m "OPE-372: filter dashboard SBOM false positives"
```

- [ ] **Step 2: PR**

Open PR:

```text
OPE-372: filter dashboard SBOM false positives
```

PR body must mention:

- canonical issue `OPE-372`,
- related DTrack tickets `OPE-365` through `OPE-392`,
- validation evidence for current SBOM vs excluded SBOM,
- `Docs updated`.

- [ ] **Step 3: Review gate**

Before merge:

1. wait for required checks,
2. inspect PR comments and review threads, including CodeRabbit,
3. fix actionable comments,
4. merge only with clean review state.

- [ ] **Step 4: Reimport and ticket cleanup**

After release publishes corrected SBOM:

1. ensure `OPE-393` no longer blocks production deploys or consciously run SBOM import only if deploy is not required,
2. reimport corrected SBOM into Dependency-Track,
3. verify the new `openoms-dashboard` project version has zero `UNKNOWN` npm components,
4. cancel duplicate false-positive Linear tickets with a comment referencing the corrected SBOM,
5. keep any real OS/package findings as separate remediation work.

## Risks And Rollback

- Risk: excluding all `next/dist/compiled` package metadata hides a real vendored package. Mitigation: exclude only `package.json` metadata files with no usable versions; keep the `next@16.2.6` component itself and real installed runtime packages.
- Risk: Dependency-Track still shows historical findings for old project versions. Mitigation: clean/cancel Linear duplicates after corrected SBOM import and document that the historical version was noisy.
- Risk: production deployment of the fix is blocked by `OPE-393`. Mitigation: finish the deploy backup upload fix before relying on a public release deploy.

## Self-Review

- Spec coverage: covers root-cause validation, SBOM generation fix, guardrail, docs, PR, reimport, and ticket cleanup.
- Placeholder scan: no `TBD`, `TODO`, or vague implementation-only steps.
- Command consistency: all commands use absolute repo paths or local repo-relative paths after `cd`.
