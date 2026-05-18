# OPE-435 Release Trigger Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure a production release/deploy is still triggered after a PR merge when GitHub does not emit or process the expected `push` workflow event for `main`.

**Architecture:** Keep the existing `Release` workflow as the primary path. Add a small, auditable fallback workflow on `pull_request.closed` that waits for the normal release run, checks by merge commit SHA, and dispatches `release.yml` on `main` only when no release run exists. Put the decision logic in a shell script with deterministic local tests.

**Tech Stack:** GitHub Actions, Bash, GitHub CLI, existing OpenOMS public release workflow.

---

## Evidence

- PR #415 merged into `main` as `a9dc012e68f286c9b4843a0639d6e851514b41e5`.
- `main` pointed at that merge commit.
- `gh run list --repo openoms-org/openoms --commit a9dc012e68f286c9b4843a0639d6e851514b41e5` returned no runs before manual intervention.
- GitHub API check suites for that SHA returned `[]` before manual intervention.
- Repository events showed `PullRequestEvent` `merged` and branch `DeleteEvent`, but no `PushEvent` for `refs/heads/main`.
- Previous PR #414 did emit a `PushEvent` for `refs/heads/main` and triggered `Release`.
- Manual `workflow_dispatch` of `release.yml` on `main` produced run `26006443776`, succeeded, and triggered enterprise deploy and SBOM import.

## Scope

In scope:

- Add a public-repo fallback for release dispatch after merged PRs.
- Make the fallback idempotent by checking for an existing run for the merge SHA before dispatching.
- Keep the fallback on the same release-relevant path set as `release.yml`, so documentation-only PRs do not deploy.
- Add local shell tests for the fallback decision logic.
- Keep all production deployment logic in the existing `Release` and enterprise workflows.

Out of scope:

- Changing enterprise deploy behavior.
- Changing image build, SBOM generation, Trivy scanning, or `repository_dispatch` payload format.
- Trying to make GitHub emit a missing `PushEvent`; the fallback only mitigates that platform/event gap.

## Files

- Create: `.github/workflows/release-fallback.yml`
  - Runs after a PR is closed and merged into `main`.
  - Waits for the normal `push` workflow path.
  - Calls the fallback script with `actions: write` permission.
- Create: `scripts/ensure-release-run-for-sha.sh`
  - Checks whether a release workflow run already exists for a SHA.
  - Dispatches `release.yml` on `main` only when missing.
  - Supports `DRY_RUN=1` for tests.
- Create: `scripts/test-ensure-release-run-for-sha.sh`
  - Uses fake `gh` commands to test: existing run, missing run dispatch, missing required env, and dispatch failure.
- Modify: `scripts/local-ci.sh`
  - Add the new shell test to local CI so future workflow changes keep this guard working.
- Modify: `README.md`
  - Document that release fallback is a guard, not the primary deploy path.
- Create: `docs/superpowers/plans/2026-05-17-ope-435-release-trigger-fallback.md`
  - This plan.

## Task 1: Write Failing Fallback Script Tests

**Files:**

- Create: `scripts/test-ensure-release-run-for-sha.sh`
- Test target not yet present: `scripts/ensure-release-run-for-sha.sh`

- [x] **Step 1: Add a test script that expects the fallback script to exist**

Create `scripts/test-ensure-release-run-for-sha.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="${repo_root}/scripts/ensure-release-run-for-sha.sh"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

if [[ ! -x "${target}" ]]; then
  echo "fallback script is missing or not executable: ${target}" >&2
  exit 1
fi

make_fake_gh() {
  local mode="$1"
  local fake_bin="${tmp_dir}/${mode}/bin"
  mkdir -p "${fake_bin}"

  cat > "${fake_bin}/gh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

mode="${OPENOMS_FAKE_GH_MODE:?}"
log_file="${OPENOMS_FAKE_GH_LOG:?}"

printf '%s\n' "$*" >> "${log_file}"

case "${mode}:${1:-}:${2:-}" in
  existing:run:list)
    cat <<'JSON'
[{"databaseId":26006443776,"workflowName":"Release","event":"push","status":"completed","conclusion":"success","headSha":"0123456789abcdef0123456789abcdef01234567","url":"https://github.com/openoms-org/openoms/actions/runs/26006443776"}]
JSON
    ;;
  missing:run:list)
    printf '[]\n'
    ;;
  missing:workflow:run)
    printf 'dispatch ok\n'
    ;;
  dispatch-fails:run:list)
    printf '[]\n'
    ;;
  dispatch-fails:workflow:run)
    echo "dispatch failed" >&2
    exit 42
    ;;
  *)
    echo "unexpected gh call for mode ${mode}: $*" >&2
    exit 99
    ;;
esac
SH

  chmod +x "${fake_bin}/gh"
  printf '%s\n' "${fake_bin}"
}

run_with_fake_gh() {
  local mode="$1"
  local log_file="${tmp_dir}/${mode}.log"
  local fake_bin
  fake_bin="$(make_fake_gh "${mode}")"

  OPENOMS_FAKE_GH_MODE="${mode}" \
  OPENOMS_FAKE_GH_LOG="${log_file}" \
  PATH="${fake_bin}:${PATH}" \
  GITHUB_REPOSITORY="openoms-org/openoms" \
  RELEASE_SHA="0123456789abcdef0123456789abcdef01234567" \
  RELEASE_WORKFLOW="release.yml" \
  RELEASE_REF="main" \
    "${target}"

  cat "${log_file}"
}

existing_calls="$(run_with_fake_gh existing)"
if grep -Fq "workflow run" <<<"${existing_calls}"; then
  echo "existing release run should not dispatch" >&2
  exit 1
fi

missing_calls="$(run_with_fake_gh missing)"
if ! grep -Fq "workflow run release.yml --repo openoms-org/openoms --ref main" <<<"${missing_calls}"; then
  echo "missing release run should dispatch release.yml on main" >&2
  echo "${missing_calls}" >&2
  exit 1
fi

if GITHUB_REPOSITORY="openoms-org/openoms" RELEASE_SHA="" RELEASE_WORKFLOW="release.yml" RELEASE_REF="main" "${target}" >/tmp/openoms-ope435.out 2>/tmp/openoms-ope435.err; then
  echo "empty RELEASE_SHA should fail" >&2
  exit 1
fi

fake_bin="$(make_fake_gh dispatch-fails)"
if OPENOMS_FAKE_GH_MODE="dispatch-fails" \
  OPENOMS_FAKE_GH_LOG="${tmp_dir}/dispatch-fails.log" \
  PATH="${fake_bin}:${PATH}" \
  GITHUB_REPOSITORY="openoms-org/openoms" \
  RELEASE_SHA="0123456789abcdef0123456789abcdef01234567" \
  RELEASE_WORKFLOW="release.yml" \
  RELEASE_REF="main" \
    "${target}" >/tmp/openoms-ope435-dispatch.out 2>/tmp/openoms-ope435-dispatch.err; then
  echo "dispatch failure should fail the fallback job" >&2
  exit 1
fi
```

- [x] **Step 2: Run the test and confirm RED**

Run:

```bash
./scripts/test-ensure-release-run-for-sha.sh
```

Expected: failure with `fallback script is missing or not executable`.

## Task 2: Implement Idempotent Release Fallback Script

**Files:**

- Create: `scripts/ensure-release-run-for-sha.sh`

- [x] **Step 1: Add the script**

Create `scripts/ensure-release-run-for-sha.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

repo="${GITHUB_REPOSITORY:-}"
sha="${RELEASE_SHA:-}"
workflow="${RELEASE_WORKFLOW:-release.yml}"
ref="${RELEASE_REF:-main}"
dry_run="${DRY_RUN:-0}"

require_env() {
  local name="$1"
  local value="$2"

  if [[ -z "${value}" ]]; then
    echo "${name} is required" >&2
    exit 2
  fi
}

require_env "GITHUB_REPOSITORY" "${repo}"
require_env "RELEASE_SHA" "${sha}"
require_env "RELEASE_WORKFLOW" "${workflow}"
require_env "RELEASE_REF" "${ref}"

existing_runs="$(
  gh run list \
    --repo "${repo}" \
    --workflow "${workflow}" \
    --commit "${sha}" \
    --limit 20 \
    --json databaseId,workflowName,event,status,conclusion,headSha,url
)"

if OPENOMS_RELEASE_RUNS_JSON="${existing_runs}" python3 - "${sha}" <<'PY'; then
import json
import os
import sys

sha = sys.argv[1]
runs = json.loads(os.environ["OPENOMS_RELEASE_RUNS_JSON"])

for run in runs:
    if run.get("headSha") == sha and run.get("status") in {"queued", "in_progress", "completed", "waiting", "requested", "pending"}:
        print(
            f"Release run already exists for {sha}: "
            f"{run.get('databaseId')} {run.get('status')} {run.get('conclusion') or ''} {run.get('url') or ''}"
        )
        raise SystemExit(0)

raise SystemExit(1)
PY
then
  exit 0
fi

echo "No release run found for ${sha}; dispatching ${workflow} on ${ref}"

if [[ "${dry_run}" == "1" ]]; then
  echo "DRY_RUN=1, not dispatching"
  exit 0
fi

gh workflow run "${workflow}" --repo "${repo}" --ref "${ref}"
```

- [x] **Step 2: Run tests and expect GREEN**

Run:

```bash
./scripts/test-ensure-release-run-for-sha.sh
```

Expected: exit 0.

## Task 3: Add Pull Request Merge Fallback Workflow

**Files:**

- Create: `.github/workflows/release-fallback.yml`

- [x] **Step 1: Add the fallback workflow**

Create `.github/workflows/release-fallback.yml`:

```yaml
name: Release Fallback

on:
  pull_request:
    branches: [main]
    types: [closed]
    paths:
      - "apps/**"
      - "packages/**"
      - "deploy/**"
      - ".github/workflows/release.yml"
      - ".github/workflows/release-fallback.yml"
      - "scripts/check-dashboard-bundle-placeholders.sh"
      - "scripts/check-dashboard-image.sh"
      - "scripts/check-dashboard-release-config.sh"
      - "scripts/check-sbom-artifacts.sh"
      - "scripts/ensure-release-run-for-sha.sh"
      - "scripts/install-syft.sh"
      - "go.work"
      - "go.work.sum"

permissions:
  actions: write
  contents: read

concurrency:
  group: release-fallback-${{ github.event.pull_request.merge_commit_sha || github.event.pull_request.number }}
  cancel-in-progress: false

jobs:
  ensure-release:
    name: Ensure release after merge
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6

      - name: Wait for normal push release
        run: sleep 90

      - name: Ensure Release workflow exists for merge SHA
        env:
          GH_TOKEN: ${{ github.token }}
          GITHUB_REPOSITORY: ${{ github.repository }}
          RELEASE_SHA: ${{ github.event.pull_request.merge_commit_sha }}
          RELEASE_WORKFLOW: release.yml
          RELEASE_REF: main
        run: ./scripts/ensure-release-run-for-sha.sh
```

- [x] **Step 2: Validate action pinning**

Run:

```bash
./scripts/validate-github-actions-pinning.sh
```

Expected: `GitHub Actions pinning validation passed`.

## Task 4: Wire the Script Test into Local CI

**Files:**

- Modify: `scripts/local-ci.sh`

- [x] **Step 1: Add a local CI check**

In `scripts/local-ci.sh`, add this check near the existing script validation checks:

```bash
run_check "release-fallback" ./scripts/test-ensure-release-run-for-sha.sh
```

- [x] **Step 2: Run quick local CI**

Run:

```bash
./scripts/local-ci.sh --quick
```

Expected: all quick checks pass, including `release-fallback=pass` in `/tmp/openoms-local-ci-quick-results.txt`.

## Task 5: Full Validation, PR, and Deployment Watch

**Files:**

- No additional file edits.

- [x] **Step 1: Run focused validation**

Run:

```bash
./scripts/test-ensure-release-run-for-sha.sh
./scripts/validate-github-actions-pinning.sh
git diff --check
```

Expected: all commands exit 0.

- [x] **Step 2: Run full local CI before push**

Run:

```bash
./scripts/local-ci.sh
```

Expected: full local CI exits 0.

- [ ] **Step 3: Commit**

Run:

```bash
git add .github/workflows/release-fallback.yml scripts/ensure-release-run-for-sha.sh scripts/test-ensure-release-run-for-sha.sh scripts/local-ci.sh docs/superpowers/plans/2026-05-17-ope-435-release-trigger-fallback.md
git commit -m "OPE-435: add release trigger fallback"
```

- [ ] **Step 4: Push and open PR**

Run:

```bash
git push -u origin fix/OPE-435-release-trigger-fallback
gh pr create --repo openoms-org/openoms --title "OPE-435: add release trigger fallback" --body-file /tmp/openoms-ope-435-pr.md
```

The PR body must include a Docs updated section and must not contain AI attribution.

- [ ] **Step 5: Before merge, verify checks and CodeRabbit**

Run:

```bash
gh pr view --repo openoms-org/openoms --json statusCheckRollup,mergeStateStatus,url
```

Also inspect CodeRabbit comments/review threads and fix actionable feedback before merge.

- [ ] **Step 6: After merge, validate the fallback itself**

After the PR is merged, watch:

```bash
gh run list --repo openoms-org/openoms --branch main --limit 20 --json databaseId,workflowName,event,status,conclusion,headSha,url
gh run list --repo openoms-org/openoms --workflow "Release Fallback" --limit 10 --json databaseId,event,status,conclusion,headSha,url
```

Expected:

- Normal `Release` should usually trigger on `push`.
- `Release Fallback` should trigger on `pull_request.closed`.
- If normal `Release` exists for the merge SHA, fallback should exit without dispatching.
- If normal `Release` is missing, fallback should dispatch `release.yml` on `main`.

## Risks and Rollback

- Risk: duplicate release dispatch if GitHub creates the normal run after the fallback check but before dispatch.
  - Mitigation: 90-second wait plus SHA-based check. Existing release workflow has `concurrency` by ref, so duplicate release runs on `main` should not deploy two versions concurrently.
- Risk: fallback needs `actions: write`.
  - Mitigation: permission is scoped to this fallback workflow; no `contents: write`.
- Risk: the fallback dispatches when GitHub Actions API is temporarily stale.
  - Mitigation: the release workflow is idempotent by SHA image tags and enterprise deploy concurrency handles one production deploy at a time.
- Rollback: revert the PR that adds `.github/workflows/release-fallback.yml` and the helper scripts. Existing `Release` behavior remains unchanged.

## Self-Review

- Spec coverage: the plan addresses missing post-merge release/deploy run, preserves the existing release workflow, adds testable fallback logic, and documents validation.
- Placeholder scan: no TBD/TODO placeholders are present.
- Type/command consistency: `RELEASE_SHA`, `RELEASE_WORKFLOW`, `RELEASE_REF`, and `GITHUB_REPOSITORY` are used consistently in script, tests, and workflow.
