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

RESULTS_FILE="/tmp/openoms-local-ci-results.txt"
QUICK=false
[[ "${1:-}" == "--quick" ]] && QUICK=true

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
    {
        echo "TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%S)"
        echo "DURATION=$((SECONDS))s"
        echo "STATUS=$([ $FAILED -eq 0 ] && echo 'pass' || echo 'fail')"
        printf "%b" "$RESULTS"
    } > "$RESULTS_FILE"
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

# ── 5. next build (skip in quick mode) ──
if ! $QUICK; then
    run_check "next-build" bash -c '
        cd "'"$REPO_ROOT"'/apps/dashboard" && npx next build 2>&1 | tail -20
    '
fi

# ── 6. go test — packages without DB (skip in quick mode) ──
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
