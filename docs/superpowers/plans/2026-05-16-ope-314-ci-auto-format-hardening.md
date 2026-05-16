# OPE-314 CI Auto-Format Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove write-capable auto-format commits from the public CI workflow while preserving the existing `Auto-format` required check context.

**Architecture:** The public CI `auto-format` job stays in `.github/workflows/ci.yml` with the same job name, but becomes a read-only verification job. It checks formatting/lint-fix drift and fails with clear instructions instead of pushing commits back to PR branches.

**Tech Stack:** GitHub Actions, Go 1.25, Node 22, ESLint, GitHub branch protection checks.

---

## Scope

This plan implements `OPE-314` in the public repository only.

In scope:

- Harden `.github/workflows/ci.yml` `auto-format` job.
- Remove `contents: write` from that job.
- Remove PR-branch checkout token/ref mutation and remove `stefanzweifel/git-auto-commit-action`.
- Preserve the check display name `Auto-format`.
- Update public docs that describe CI as auto-formatting.

Out of scope:

- `auto-merge-gate` and `dependabot-auto-merge.yml` write permissions; those are separate automation decisions.
- Changing branch protection rules manually.
- Reworking all CI duplication.

## Files

- Modify: `.github/workflows/ci.yml`
  - Replace write-capable auto-format commit job with read-only check.
  - Keep `jobs.auto-format.name: Auto-format`.
- Modify: `README.md`
  - Replace CI wording that says auto-format with wording that says format verification.
- Create or keep: `docs/superpowers/plans/2026-05-16-ope-314-ci-auto-format-hardening.md`
  - Implementation record and review checklist.

## Proposed Workflow Behavior

The hardened `auto-format` job should:

- Run only on pull requests, as today.
- Use `permissions.contents: read`.
- Checkout the PR merge ref normally, without `ref: ${{ github.head_ref }}` and without explicitly passing `secrets.GITHUB_TOKEN`.
- Run Go formatting check:

```bash
UNFORMATTED=$(gofmt -l apps/api-server packages)
if [ -n "$UNFORMATTED" ]; then
  echo "::error::Go files need formatting. Run: gofmt -w -s apps/ packages/"
  echo "$UNFORMATTED"
  exit 1
fi
```

- Run frontend install and ESLint autofix dry run as a drift check without committing:

```bash
cd apps/dashboard
npm ci
npx eslint --fix-dry-run src/
```

If `--fix-dry-run` is not accepted locally by the repo ESLint version, use this fallback in implementation:

```bash
cd apps/dashboard
npm ci
npx eslint src/
```

The fallback is acceptable because the existing `Frontend` job already enforces frontend lint/build, and the primary security objective is removing CI write capability and auto-commit behavior.

## Tasks

### Task 1: Prepare Branch And Linear State

**Files:**
- No file changes.

- [ ] **Step 1: Move Linear issue to In Progress**

Use Linear to update `OPE-314` to `In Progress`.

- [ ] **Step 2: Create branch**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git checkout main
git pull --ff-only
git checkout -b fix/OPE-314-ci-auto-format-hardening
```

Expected: branch `fix/OPE-314-ci-auto-format-hardening` exists and tracks no remote yet.

### Task 2: Harden CI Auto-Format Job

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Replace write permissions**

Change:

```yaml
permissions:
  contents: write
```

to:

```yaml
permissions:
  contents: read
```

inside `jobs.auto-format`.

- [ ] **Step 2: Replace PR branch checkout**

Change the checkout step from:

```yaml
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6
  with:
    ref: ${{ github.head_ref }}
    token: ${{ secrets.GITHUB_TOKEN }}
```

to:

```yaml
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6
```

- [ ] **Step 3: Replace auto-fix Go step with formatting check**

Replace:

```yaml
- name: Auto-fix Go code
  run: |
    for dir in apps/api-server packages/allegro-go-sdk packages/inpost-go-sdk packages/order-engine packages/erli-go-sdk packages/ebay-go-sdk; do
      echo "Fixing $dir..."
      golangci-lint run --fix --timeout=5m "./$dir/..." || true
    done
```

with:

```yaml
- name: Check Go formatting
  run: |
    UNFORMATTED=$(gofmt -l apps/api-server packages)
    if [ -n "$UNFORMATTED" ]; then
      echo "::error::Go files need formatting. Run: gofmt -w -s apps/ packages/"
      echo "$UNFORMATTED"
      exit 1
    fi
```

- [ ] **Step 4: Replace frontend auto-fix with read-only lint check**

Replace:

```yaml
- name: Auto-fix frontend code
  working-directory: apps/dashboard
  run: |
    npm ci
    npx eslint --fix src/ || true
```

with:

```yaml
- name: Check frontend lint
  working-directory: apps/dashboard
  run: |
    npm ci
    npx eslint src/
```

- [ ] **Step 5: Remove git auto-commit action**

Delete this entire step:

```yaml
- uses: stefanzweifel/git-auto-commit-action@04702edda442b2e678b25b537cec683a1493fcb9 # v7
  with:
    commit_message: "style: auto-format code"
    commit_author: "github-actions[bot] <github-actions[bot]@users.noreply.github.com>"
```

Expected result:

- No `contents: write` inside `jobs.auto-format`.
- No `git-auto-commit-action` in `ci.yml`.
- No `golangci-lint run --fix` in `ci.yml`.
- No `eslint --fix` in `ci.yml`.
- The job name remains `Auto-format`.

### Task 3: Update CI Documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update CI/CD table wording**

Replace wording that says CI includes `auto-format` with wording that says CI includes format verification.

Expected shape:

```markdown
| CI/CD | GitHub Actions (lint, test, security scan, format verification, Trivy) |
```

- [ ] **Step 2: Update repository structure comment**

Replace:

```markdown
│   ├── ci.yml                 # Lint, test, security scan, auto-format
```

with:

```markdown
│   ├── ci.yml                 # Lint, test, security scan, format checks
```

### Task 4: Validate Locally

**Files:**
- No file changes unless validation uncovers a bug.

- [ ] **Step 1: Validate no forbidden CI patterns remain**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
rg -n "git-auto-commit-action|contents: write|golangci-lint run --fix|eslint --fix|github.head_ref" .github/workflows/ci.yml
```

Expected:

- No `git-auto-commit-action`.
- No `golangci-lint run --fix`.
- No `eslint --fix`.
- No `github.head_ref`.
- `contents: write` may still appear for `auto-merge-gate`; confirm manually that it is not inside `jobs.auto-format`.

- [ ] **Step 2: Validate action pinning**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
scripts/validate-github-actions-pinning.sh
```

Expected: exit 0.

- [ ] **Step 3: Validate YAML parses**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml"); puts "ci.yml ok"'
```

Expected:

```text
ci.yml ok
```

- [ ] **Step 4: Check patch whitespace**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git diff --check
```

Expected: no output, exit 0.

- [ ] **Step 5: Run local CI before push**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
./scripts/local-ci.sh
```

Expected: all checks pass.

### Task 5: Commit, PR, Review, Merge

**Files:**
- Commit modified files and this plan.

- [ ] **Step 1: Commit**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git add .github/workflows/ci.yml README.md docs/superpowers/plans/2026-05-16-ope-314-ci-auto-format-hardening.md
git commit -m "OPE-314: harden ci auto-format job"
```

- [ ] **Step 2: Push**

Run:

```bash
cd /Users/rafs/praca/openoms-dev/public
git push -u origin fix/OPE-314-ci-auto-format-hardening
```

Expected: pre-push quick validation passes.

- [ ] **Step 3: Open PR**

Title:

```text
OPE-314: harden CI auto-format job
```

PR body sections:

```markdown
## Summary
- replace write-capable auto-format commits with read-only verification
- keep the `Auto-format` check context for branch protection compatibility
- update CI docs to describe format checks instead of auto-format commits

## Test plan
- [ ] scripts/validate-github-actions-pinning.sh
- [ ] ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml"); puts "ci.yml ok"'
- [ ] git diff --check
- [ ] ./scripts/local-ci.sh

## Docs updated
- [x] README.md — CI wording now says format verification/checks
```

- [ ] **Step 4: Review CI and CodeRabbit**

Wait for PR checks. Inspect CodeRabbit comments and review threads before merge.

- [ ] **Step 5: Merge after green checks**

If all required checks pass and there are no actionable CodeRabbit comments, squash merge and delete branch.

## Risk And Rollback

Risk:

- Branch protection may require the `Auto-format` check context. We keep the job name `Auto-format` to avoid this.
- Removing auto-commit means contributors must fix formatting locally. This is intentional and safer.
- Running `npx eslint src/` duplicates the `Frontend` job lint work. That is acceptable short-term; the job is kept primarily for check-context stability.

Rollback:

- Revert the PR to restore the previous auto-format behavior.
- Safer rollback alternative: keep the read-only job but temporarily loosen branch protection if the check context changes unexpectedly.

## Self-Review Checklist

- [ ] Plan preserves the `Auto-format` check name.
- [ ] Plan removes write permission from the auto-format job.
- [ ] Plan removes automatic commits to PR branches.
- [ ] Plan does not touch Dependabot auto-merge policy.
- [ ] Plan includes validation for YAML, action pinning, diff hygiene, and local CI.
