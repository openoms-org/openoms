#!/usr/bin/env bash
# local-ci.sh — replicate GitHub Actions CI checks locally
# Catches lint/vet/build errors before push, saving CI round-trips.
#
# Usage:
#   ./scripts/local-ci.sh          # run all checks
#   ./scripts/local-ci.sh --quick  # skip next build and go test (fast ~15s)
#
# Exit code: 0 = all passed, 1 = something failed

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

QUICK=false
[[ "${1:-}" == "--quick" ]] && QUICK=true

if $QUICK; then
    MODE="quick"
else
    MODE="full"
fi

RESULTS_FILE="/tmp/openoms-local-ci-${MODE}-results.txt"
LATEST_RESULTS_FILE="/tmp/openoms-local-ci-results.txt"

# colors
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
CYAN="\033[36m"
DIM="\033[2m"
BOLD="\033[1m"
RESET="\033[0m"

FAILED=0
CHECKS_RUN=0
CHECKS_PASSED=0
RESULTS=""

run_check() {
    local name="$1"
    shift
    CHECKS_RUN=$((CHECKS_RUN + 1))
    printf "  ${CYAN}⟳${RESET} %-20s " "$name"

    local output
    local start_time=$SECONDS
    if output=$("$@" 2>&1); then
        local elapsed=$((SECONDS - start_time))
        printf "${GREEN}✓${RESET} ${DIM}${elapsed}s${RESET}\n"
        CHECKS_PASSED=$((CHECKS_PASSED + 1))
        RESULTS="${RESULTS}${name}=pass\n"
    else
        local elapsed=$((SECONDS - start_time))
        printf "${RED}✗${RESET} ${DIM}${elapsed}s${RESET}\n"
        # show first 15 lines of error output
        echo "$output" | head -15 | sed 's/^/    /'
        if [ "$(echo "$output" | wc -l)" -gt 15 ]; then
            local total
            total=$(echo "$output" | wc -l)
            printf "    ${DIM}... and %d more lines${RESET}\n" $((total - 15))
        fi
        FAILED=1
        RESULTS="${RESULTS}${name}=fail\n"
    fi
}

save_results() {
    local git_branch git_commit git_dirty
    git_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    git_commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    if [ -n "$(git status --porcelain --untracked-files=no 2>/dev/null)" ]; then
        git_dirty="true"
    else
        git_dirty="false"
    fi

    {
        echo "TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        echo "MODE=$MODE"
        echo "GIT_BRANCH=$git_branch"
        echo "GIT_COMMIT=$git_commit"
        echo "GIT_DIRTY=$git_dirty"
        echo "DURATION=$((SECONDS))s"
        echo "STATUS=$([ $FAILED -eq 0 ] && echo 'pass' || echo 'fail')"
        printf "%b" "$RESULTS"
    } > "$RESULTS_FILE"
    cp "$RESULTS_FILE" "$LATEST_RESULTS_FILE"
}

# ── header ──
echo ""
printf "${BOLD}  Local CI${RESET}"
if $QUICK; then printf " ${DIM}(quick mode)${RESET}"; fi
echo ""
printf "  ${DIM}────────────────────────────────────${RESET}\n"

# Go module directories matching CI
GO_LINT_DIRS=(
    "apps/api-server"
    "packages/allegro-go-sdk"
    "packages/inpost-go-sdk"
    "packages/order-engine"
    "packages/erli-go-sdk"
    "packages/ebay-go-sdk"
)

GO_TEST_DIRS=(
    "packages/allegro-go-sdk"
    "packages/inpost-go-sdk"
    "packages/order-engine"
    "packages/dhl-go-sdk"
    "packages/ksef-go-sdk"
    "packages/amazon-sp-sdk"
    "packages/dpd-go-sdk"
    "packages/gls-go-sdk"
    "packages/btp-go-sdk"
    "packages/erli-go-sdk"
    "packages/ebay-go-sdk"
)

# ── 1. gofmt ──
run_check "gofmt" bash -c '
    unformatted=$(gofmt -l apps/ packages/ 2>/dev/null | grep -v vendor || true)
    if [ -n "$unformatted" ]; then
        echo "Files not formatted:"
        echo "$unformatted"
        exit 1
    fi
'

# ── 2. go vet ──
run_check "go-vet" bash -c '
    errors=""
    for dir in '"$(printf '%s ' "${GO_LINT_DIRS[@]}")"'; do
        if [ -d "'"$REPO_ROOT"'/$dir" ]; then
            output=$(cd "'"$REPO_ROOT"'/$dir" && go vet ./... 2>&1) || errors="${errors}${output}\n"
        fi
    done
    if [ -n "$errors" ]; then
        printf "%b" "$errors"
        exit 1
    fi
'

# ── 3. golangci-lint ──
run_check "golangci-lint" bash -c '
    errors=""
    for dir in '"$(printf '%s ' "${GO_LINT_DIRS[@]}")"'; do
        if [ -d "'"$REPO_ROOT"'/$dir" ]; then
            output=$(cd "'"$REPO_ROOT"'/$dir" && golangci-lint run --timeout=5m ./... 2>&1) || errors="${errors}${output}\n"
        fi
    done
    if [ -n "$errors" ]; then
        printf "%b" "$errors"
        exit 1
    fi
'

# ── 4. eslint (frontend) ──
run_check "eslint" bash -c '
    cd "'"$REPO_ROOT"'/apps/dashboard" && npx eslint --quiet src/ 2>&1
'

# ── 5. dashboard build secret guard ──
run_check "dashboard-secrets" bash -c '
    cd "'"$REPO_ROOT"'" && ./scripts/check-dashboard-build-secrets.sh
'

# ── 6. dashboard Sentry release guard tests ──
run_check "dashboard-sentry-guards" bash -c '
    cd "'"$REPO_ROOT"'" && ./scripts/test-dashboard-sentry-guards.sh
'

# ── 7. GitHub Actions pinning guard ──
run_check "actions-pinning" bash -c '
    cd "'"$REPO_ROOT"'" && ./scripts/validate-github-actions-pinning.sh
'

# ── 8. next build (skip in quick mode) ──
if ! $QUICK; then
    run_check "next-build" bash -c '
        cd "'"$REPO_ROOT"'/apps/dashboard"
        tmp_output=$(mktemp)
        if npx next build >"$tmp_output" 2>&1; then
            rm -f "$tmp_output"
            exit 0
        fi
        status=$?
        tail -20 "$tmp_output"
        rm -f "$tmp_output"
        exit "$status"
    '
fi

# ── 9. go test — packages without DB (skip in quick mode) ──
if ! $QUICK; then
    run_check "go-test" bash -c '
        errors=""
        for dir in '"$(printf '%s ' "${GO_TEST_DIRS[@]}")"'; do
            if [ -d "'"$REPO_ROOT"'/$dir" ]; then
                output=$(cd "'"$REPO_ROOT"'/$dir" && go test ./... 2>&1) || errors="${errors}[${dir}]\n${output}\n"
            fi
        done
        if [ -n "$errors" ]; then
            printf "%b" "$errors"
            exit 1
        fi
    '
fi

# ── results ──
echo ""
if [ $FAILED -eq 0 ]; then
    printf "  ${GREEN}${BOLD}All %d checks passed${RESET}\n" "$CHECKS_RUN"
else
    printf "  ${RED}${BOLD}%d/%d checks failed${RESET}\n" $((CHECKS_RUN - CHECKS_PASSED)) "$CHECKS_RUN"
fi
printf "  ${DIM}%ds total${RESET}\n\n" $((SECONDS))

save_results
exit $FAILED
