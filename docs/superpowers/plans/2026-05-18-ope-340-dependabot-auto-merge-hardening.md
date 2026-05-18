# OPE-340 Dependabot Auto-Merge Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Limit Dependabot auto-merge workflow write permissions to the smallest job that actually performs the merge.

**Architecture:** Keep the workflow split into a read-only metadata job and a separate merge job. The metadata job runs the pinned Dependabot metadata action with read permissions only; the merge job gets `contents: write` and `pull-requests: write` only after the patch-update condition is known.

**Tech Stack:** GitHub Actions, `gh` CLI, Dependabot metadata action pinned by commit SHA, existing local workflow validation scripts.

---

## Files and Responsibilities

- Modify: `.github/workflows/dependabot-auto-merge.yml`
  - Remove workflow-level write permissions.
  - Add job-level read permissions for metadata lookup.
  - Add job-level write permissions only for the auto-merge job.
  - Preserve the pinned `dependabot/fetch-metadata` action SHA.
- No docs/context update required beyond this plan because runtime product behavior does not change.

## Task 1: Split Metadata and Merge Permissions

**Files:**
- Modify: `.github/workflows/dependabot-auto-merge.yml`

- [ ] **Step 1: Inspect current workflow**

Run:

```bash
sed -n '1,120p' .github/workflows/dependabot-auto-merge.yml
```

Expected: workflow-level `contents: write` and `pull-requests: write` are present.

- [ ] **Step 2: Update workflow permissions**

Change the workflow to this structure:

```yaml
name: Dependabot Auto-Merge

on: pull_request

permissions:
  contents: read

jobs:
  metadata:
    runs-on: ubuntu-latest
    if: github.actor == 'dependabot[bot]'
    permissions:
      contents: read
      pull-requests: read
    outputs:
      update-type: ${{ steps.metadata.outputs.update-type }}
    steps:
      - name: Fetch Dependabot metadata
        id: metadata
        uses: dependabot/fetch-metadata@25dd0e34f4fe68f24cc83900b1fe3fe149efef98 # v3
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}

  auto-merge:
    runs-on: ubuntu-latest
    needs: metadata
    if: github.actor == 'dependabot[bot]' && needs.metadata.outputs.update-type == 'version-update:semver-patch'
    permissions:
      contents: write
      pull-requests: write
    steps:
      - name: Auto-merge patch updates
        run: gh pr merge --auto --squash "$PR_URL"
        env:
          PR_URL: ${{ github.event.pull_request.html_url }}
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Validate YAML parses**

Run:

```bash
ruby -e 'require "yaml"; Dir[".github/workflows/*.yml"].each { |f| YAML.load_file(f) }; puts "workflow yaml ok"'
```

Expected: `workflow yaml ok`.

- [ ] **Step 4: Validate action pinning**

Run:

```bash
./scripts/validate-github-actions-pinning.sh
```

Expected: script exits 0 and reports all actions pinned.

- [ ] **Step 5: Run focused diff checks**

Run:

```bash
git diff --check
git diff -- .github/workflows/dependabot-auto-merge.yml
```

Expected: no whitespace errors and only the intended permissions/job split.

## Task 2: Full Public Repo Validation and PR

**Files:**
- Modify: `.github/workflows/dependabot-auto-merge.yml`
- Existing full validation evidence: `/tmp/openoms-local-ci-full-results.txt`

- [ ] **Step 1: Run full local CI**

Run:

```bash
./scripts/local-ci.sh
```

Expected: `STATUS=pass` in `/tmp/openoms-local-ci-full-results.txt`.

- [ ] **Step 2: Commit with Linear ID**

Run:

```bash
git add .github/workflows/dependabot-auto-merge.yml docs/superpowers/plans/2026-05-18-ope-340-dependabot-auto-merge-hardening.md
git commit -m "OPE-340: scope dependabot auto-merge permissions"
```

- [ ] **Step 3: Push and open PR**

Run:

```bash
git push -u origin fix/OPE-340-dependabot-auto-merge-permissions
gh pr create --repo openoms-org/openoms --title "OPE-340: scope dependabot auto-merge permissions" --body-file /tmp/ope-340-pr-body.md
```

Expected: PR opens against `main` with docs section noting that this plan is the only doc update.

- [ ] **Step 4: Post-PR gate**

Run:

```bash
gh pr view <PR_NUMBER> --repo openoms-org/openoms --json statusCheckRollup,comments,reviews,mergeStateStatus
```

Expected: checks pass; CodeRabbit has no actionable comments or any actionable comments are fixed before merge.

## Risk and Rollback

- Risk: if `dependabot/fetch-metadata` needs broader permissions than `pull-requests: read`, the metadata job may fail on Dependabot PRs. The workflow can be rolled back by reverting this commit or by adding the minimum missing permission to the metadata job.
- Rollback: revert the OPE-340 PR; this restores the previous single-job permission model.
- Operational impact: no production runtime impact; only Dependabot PR automation changes.

## Self-Review Checklist

- The pinned Dependabot action SHA is preserved.
- Write permissions are not available to the metadata job.
- `gh pr merge` still runs with `contents: write` and `pull-requests: write`.
- No public runtime docs need updates because product behavior is unchanged.
