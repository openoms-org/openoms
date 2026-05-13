#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  echo "dashboard sentry guard test failed: $*" >&2
  exit 1
}

assert_fails() {
  local name="$1"
  shift
  if "$@"; then
    fail "$name unexpectedly passed"
  fi
  echo "$name=pass"
}

assert_passes() {
  local name="$1"
  shift
  if ! "$@"; then
    fail "$name unexpectedly failed"
  fi
  echo "$name=pass"
}

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

bundle_with_placeholder="$tmp_dir/bundle-with-placeholder"
mkdir -p "$bundle_with_placeholder/.next/server"
printf 'const dsn = "SENTRY_DSN_PLACEHOLDER";\n' > "$bundle_with_placeholder/.next/server/app.js"
ln -s "$bundle_with_placeholder/missing-target" "$bundle_with_placeholder/.next/server/broken-link"

placeholder_output="$tmp_dir/placeholder.out"
if "$repo_root/scripts/check-dashboard-bundle-placeholders.sh" "$bundle_with_placeholder" >"$placeholder_output" 2>&1; then
  fail "bundle placeholder scanner passed with SENTRY_DSN_PLACEHOLDER and a broken symlink"
fi
grep -q "SENTRY_DSN_PLACEHOLDER" "$placeholder_output" || fail "bundle placeholder scanner did not report the placeholder match"
echo "bundle_placeholder_negative=pass"

bundle_without_placeholder="$tmp_dir/bundle-without-placeholder"
mkdir -p "$bundle_without_placeholder/.next/server"
printf 'const dsn = "https://example.ingest.sentry.io/123";\n' > "$bundle_without_placeholder/.next/server/app.js"
ln -s "$bundle_without_placeholder/missing-target" "$bundle_without_placeholder/.next/server/broken-link"
assert_passes "bundle_placeholder_positive" "$repo_root/scripts/check-dashboard-bundle-placeholders.sh" "$bundle_without_placeholder"

assert_fails "release_config_rejects_placeholder" env -i PATH="$PATH" \
  NEXT_PUBLIC_SENTRY_DSN="SENTRY_DSN_PLACEHOLDER" \
  SENTRY_ORG="openoms" \
  SENTRY_PROJECT="dashboard" \
  SENTRY_AUTH_TOKEN="token" \
  "$repo_root/scripts/check-dashboard-release-config.sh"

assert_fails "release_config_rejects_mixed_case_placeholder" env -i PATH="$PATH" \
  NEXT_PUBLIC_SENTRY_DSN="https://Sentry_Dsn_PlaceHolder.ingest.sentry.io/123" \
  SENTRY_ORG="openoms" \
  SENTRY_PROJECT="dashboard" \
  SENTRY_AUTH_TOKEN="token" \
  "$repo_root/scripts/check-dashboard-release-config.sh"

assert_fails "release_config_requires_token" env -i PATH="$PATH" \
  NEXT_PUBLIC_SENTRY_DSN="https://example.ingest.sentry.io/123" \
  SENTRY_ORG="openoms" \
  SENTRY_PROJECT="dashboard" \
  SENTRY_AUTH_TOKEN="" \
  "$repo_root/scripts/check-dashboard-release-config.sh"

assert_passes "release_config_accepts_valid_values" env -i PATH="$PATH" \
  NEXT_PUBLIC_SENTRY_DSN="https://example.ingest.sentry.io/123" \
  SENTRY_ORG="openoms" \
  SENTRY_PROJECT="dashboard" \
  SENTRY_AUTH_TOKEN="token" \
  "$repo_root/scripts/check-dashboard-release-config.sh"

guard_repo="$tmp_dir/build-secret-guard"
mkdir -p "$guard_repo/scripts" "$guard_repo/apps/dashboard" "$guard_repo/.github/workflows"
cp "$repo_root/scripts/check-dashboard-build-secrets.sh" "$guard_repo/scripts/check-dashboard-build-secrets.sh"
chmod +x "$guard_repo/scripts/check-dashboard-build-secrets.sh"

cat > "$guard_repo/apps/dashboard/Dockerfile" <<'DOCKERFILE'
FROM scratch
RUN --mount=type=secret,id=sentry_auth_token true
DOCKERFILE

cat > "$guard_repo/.github/workflows/release.yml" <<'YAML'
name: Release
jobs:
  release-api:
    runs-on: ubuntu-latest
    steps:
      - run: ./scripts/check-dashboard-release-config.sh
  release-dashboard:
    runs-on: ubuntu-latest
    steps:
      - uses: docker/build-push-action@v5
        with:
          secrets: |
            sentry_auth_token=${{ secrets.SENTRY_AUTH_TOKEN }}
YAML

assert_fails "build_secret_rejects_preflight_outside_dashboard_job" \
  "$guard_repo/scripts/check-dashboard-build-secrets.sh"

echo "dashboard_sentry_guard_tests=pass"
