# OPE-294 Pin Public GitHub Actions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pin all external GitHub Actions used by the public OpenOMS workflows to full commit SHA refs and add local/CI guards that prevent mutable action tags from returning.

**Architecture:** Treat the public repo as the second and final tranche of `OPE-294` after the enterprise PR. Replace every remote `uses: owner/repo@tag` reference with `uses: owner/repo@<40-char-sha> # tag`, preserving the human-readable source tag as a comment. Add a repo-local validator and wire it into both GitHub CI and `scripts/local-ci.sh`, because public pushes already require local CI evidence.

**Tech Stack:** GitHub Actions, Bash, Ruby YAML parser, public OpenOMS local CI, pinned action SHAs resolved from GitHub tag refs on 2026-05-15.

---

## Scope

Public repo only:

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.github/workflows/e2e.yml`
- `.github/workflows/scheduled-scan.yml`
- `.github/workflows/dependabot-auto-merge.yml`
- `scripts/validate-github-actions-pinning.sh`
- `scripts/local-ci.sh`
- `.claude/context/SECURITY_POSTURE.md` (local AI context, ignored by git)
- `docs/system-documentation.md`
- `docs/superpowers/plans/2026-05-15-ope-294-pin-public-github-actions.md`

Enterprise is already complete in `openoms-enterprise` PR #111. After this public PR is merged, `OPE-294` can be moved to Done.

## Resolved Action Pins

Use these full commit SHAs:

```text
actions/checkout@v6                         de0fac2e4500dabe0009e67214ff5f5447ce83dd
actions/download-artifact@v8                3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c
actions/setup-go@v6                         4a3601121dd01d1626a1e23e37211e3254c1c06c
actions/setup-node@v6                       48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e
actions/upload-artifact@v7                  043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
aquasecurity/trivy-action@v0.36.0           ed142fd0673e97e23eac54620cfb913e5ce36c25
azure/setup-helm@v5                         dda3372f752e03dde6b3237bc9431cdc2f7a02a2
dependabot/fetch-metadata@v3                25dd0e34f4fe68f24cc83900b1fe3fe149efef98
docker/build-push-action@v7                 bcafcacb16a39f128d818304e6c9c0c18556b85f
docker/login-action@v4                      4907a6ddec9925e35a0a9e82d7399ccc52663121
docker/setup-buildx-action@v4               4d04d5d9486b7bd6fa91e7baf45bbb4f8b9deedd
dorny/paths-filter@v4                       fbd0ab8f3e69293af611ebaee6363fc25e6d187d
github/codeql-action/upload-sarif@v4        9e0d7b8d25671d64c341c19c0152d693099fb5ba
golangci/golangci-lint-action@v9            1e7e51e771db61008b38414a730f564565cf7c20
peter-evans/repository-dispatch@v4          28959ce8df70de7be546dd1250a005dd32156697
stefanzweifel/git-auto-commit-action@v7     04702edda442b2e678b25b537cec683a1493fcb9
```

Note: for annotated tags, use the peeled commit SHA (`refs/tags/<tag>^{}`), not the tag object SHA.

## Task 1: Pin Public Workflow Actions

- [ ] **Step 1: Replace remote action tag refs**

In `.github/workflows/*.yml`, replace every remote action tag ref with the resolved full SHA and keep the original tag comment.

Examples:

```yaml
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6
```

```yaml
uses: docker/build-push-action@bcafcacb16a39f128d818304e6c9c0c18556b85f # v7
```

```yaml
uses: github/codeql-action/upload-sarif@9e0d7b8d25671d64c341c19c0152d693099fb5ba # v4
```

- [ ] **Step 2: Keep existing SHA-pinned artifact actions valid**

Existing pinned refs such as:

```yaml
uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
```

may stay pinned as-is or receive the readability comment:

```yaml
uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7
```

- [ ] **Step 3: Verify no remote tag refs remain**

Run:

```bash
rg -n "uses:" .github/workflows
```

Expected: every remote `uses:` ref ends with a 40-character SHA, optionally followed by a tag comment.

## Task 2: Add Pinning Validation

- [ ] **Step 1: Create `scripts/validate-github-actions-pinning.sh`**

Create:

```bash
#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

failed=0

while IFS= read -r line; do
  file="${line%%:*}"
  rest="${line#*:}"
  line_no="${rest%%:*}"
  value="$(printf '%s' "${rest#*:}" | sed -E 's/^[[:space:]-]*uses:[[:space:]]*//; s/[[:space:]]+#.*$//')"

  case "${value}" in
    ./*|docker://*)
      continue
      ;;
  esac

  if [[ "${value}" != *@* ]]; then
    echo "${file}:${line_no}: remote action must include an @ ref: ${value}" >&2
    failed=1
    continue
  fi

  ref="${value##*@}"
  if ! [[ "${ref}" =~ ^[0-9a-f]{40}$ ]]; then
    echo "${file}:${line_no}: remote action must be pinned to a full commit SHA, got ${value}" >&2
    failed=1
  fi
done < <(grep -RInE 'uses:[[:space:]]*[^[:space:]#]+' .github/workflows)

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi

echo "GitHub Actions pinning validation passed"
```

- [ ] **Step 2: Make the script executable**

Run:

```bash
chmod +x scripts/validate-github-actions-pinning.sh
```

- [ ] **Step 3: Add a CI job to `.github/workflows/ci.yml`**

Add:

```yaml
  github-actions-pinning:
    name: GitHub Actions Pinning
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6
      - name: Validate GitHub Actions pinning
        run: scripts/validate-github-actions-pinning.sh
```

- [ ] **Step 4: Add the validator to `scripts/local-ci.sh`**

Add a check after the dashboard Sentry guard:

```bash
# ── 7. GitHub Actions pinning guard ──
run_check "actions-pinning" bash -c '
    cd "'"$REPO_ROOT"'" && ./scripts/validate-github-actions-pinning.sh
'
```

Renumber the following comment headers only if it keeps the file readable; do not change local CI behavior otherwise.

## Task 3: Update Public Security Context And Docs

- [ ] **Step 1: Add a factual note to `.claude/context/SECURITY_POSTURE.md`**

Add a short supply-chain note:

```markdown
- Public and enterprise GitHub Actions workflows pin external actions to full commit SHAs, with CI validation that rejects mutable action refs.
```

Keep it factual, not aspirational.

- [ ] **Step 2: Add a trackable note to `docs/system-documentation.md`**

Add this sentence near the CI/CD release pipeline hardening notes:

```markdown
Publiczne workflowy GitHub Actions pinują zewnetrzne akcje do pelnych commit SHA zamiast mutowalnych tagow semver. `scripts/validate-github-actions-pinning.sh` jest uruchamiany lokalnie przez `scripts/local-ci.sh` oraz w publicznym CI, zeby blokowac regresje do tagow typu `@v4`/`@v7`.
```

## Task 4: Local Verification

- [ ] **Step 1: Run the pinning validator**

Run:

```bash
scripts/validate-github-actions-pinning.sh
```

Expected:

```text
GitHub Actions pinning validation passed
```

- [ ] **Step 2: Parse all public workflow YAML files**

Run:

```bash
ruby -e 'require "yaml"; Dir[".github/workflows/*.{yml,yaml}"].sort.each { |f| YAML.load_file(f); puts "parsed #{f}" }'
```

Expected: every workflow parses without exceptions.

- [ ] **Step 3: Run grep audit**

Run:

```bash
grep -RInE 'uses:[[:space:]]*[^[:space:]#]+@[0-9a-f]{40}([[:space:]]+#.*)?$' .github/workflows || true
```

Expected: only 40-character SHA refs, optionally followed by comments.

- [ ] **Step 4: Run whitespace check**

Run:

```bash
git diff --check
```

Expected: no output and exit code 0.

- [ ] **Step 5: Run mandatory public local CI**

Run:

```bash
./scripts/local-ci.sh
```

Expected: `STATUS=pass` in `/tmp/openoms-local-ci-full-results.txt`.

## Task 5: Commit, PR, Review, Merge

- [ ] **Step 1: Commit**

```bash
git add .github/workflows scripts/validate-github-actions-pinning.sh scripts/local-ci.sh docs/system-documentation.md docs/superpowers/plans/2026-05-15-ope-294-pin-public-github-actions.md
git commit -m "OPE-294: pin public GitHub Actions"
```

- [ ] **Step 2: Open PR**

Title:

```text
OPE-294: pin public GitHub Actions
```

Body:

```markdown
## Summary
- Pin all external GitHub Actions used by public workflows to full commit SHA refs.
- Keep source tags as YAML comments for readability.
- Add local and CI guards that reject remote action refs not pinned to full SHA.

## Test Plan
- [x] scripts/validate-github-actions-pinning.sh
- [x] ruby YAML parse for all public workflows
- [x] grep audit of workflow uses refs
- [x] git diff --check
- [x] ./scripts/local-ci.sh

## Docs updated
- [x] docs/system-documentation.md - recorded GitHub Actions pinning guard
- [x] docs/superpowers/plans/2026-05-15-ope-294-pin-public-github-actions.md - public tranche plan
```

- [ ] **Step 3: Review gate**

Before merge:

```bash
gh pr view <PR> --json statusCheckRollup,comments,reviews,mergeable,isDraft,state
gh api graphql -f query='query($owner:String!, $repo:String!, $number:Int!) { repository(owner:$owner, name:$repo) { pullRequest(number:$number) { reviewThreads(first:100) { nodes { isResolved isOutdated comments(first:20) { nodes { author { login } body path line originalLine } } } } } } } }' -F owner=openoms-org -F repo=openoms -F number=<PR>
```

Expected:

- Required CI green.
- No unresolved review threads.
- No actionable CodeRabbit comments left.

- [ ] **Step 4: Merge and close Linear**

After merge, verify the public release/deploy path if it triggers from `main`. Then move `OPE-294` to Done because both enterprise and public tranches are complete.

## Risks and Rollback

- Risk: a pinned action SHA later needs an update. Mitigation: bump SHA in a reviewed PR and keep the tag comment current.
- Risk: release workflow syntax breaks after pinning. Mitigation: YAML parse, local CI, GitHub CI, and review gate before merge.
- Risk: `scripts/local-ci.sh` becomes slower. Mitigation: the new guard is a fast grep/shell check and should add less than one second.
- Rollback: revert the PR if the workflow pinning breaks CI unexpectedly. Preferred remediation is a targeted SHA correction rather than returning to mutable tags.
