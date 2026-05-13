#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dockerfile="$repo_root/apps/dashboard/Dockerfile"
release_workflow="$repo_root/.github/workflows/release.yml"

fail() {
  echo "dashboard build secret check failed: $*" >&2
  exit 1
}

if grep -n -i -E '^[[:space:]]*(ARG|ENV)[[:space:]]+SENTRY_AUTH_TOKEN([[:space:]=]|$)' "$dockerfile"; then
  fail "SENTRY_AUTH_TOKEN must not be declared as Docker ARG or ENV"
fi

if awk '
  /^[[:space:]]*release-dashboard:/ { in_dashboard = 1 }
  in_dashboard && /^[[:space:]]*[a-zA-Z0-9_-]+:/ && $0 !~ /^[[:space:]]*(release-dashboard|steps|with|env):/ {
    if ($0 ~ /^[[:space:]]{2}[a-zA-Z0-9_-]+:/) in_dashboard = 0
  }
  in_dashboard && /build-args:/ { in_build_args = 1; next }
  in_build_args && /^[[:space:]]*[a-zA-Z0-9_-]+:/ { in_build_args = 0 }
  in_dashboard && in_build_args && /SENTRY_AUTH_TOKEN[[:space:]]*=/ { found = 1 }
  END { exit found ? 0 : 1 }
' "$release_workflow"; then
  fail "release dashboard build must not pass SENTRY_AUTH_TOKEN through build-args"
fi

if ! grep -q -- '--mount=type=secret,id=sentry_auth_token' "$dockerfile"; then
  fail "dashboard Dockerfile must consume Sentry auth through a BuildKit secret mount"
fi

if ! grep -q -E 'sentry_auth_token=.*secrets\.SENTRY_AUTH_TOKEN|SENTRY_AUTH_TOKEN=.*secrets\.SENTRY_AUTH_TOKEN' "$release_workflow"; then
  fail "release workflow must expose SENTRY_AUTH_TOKEN as a BuildKit secret"
fi

echo "dashboard_build_secret_check=pass"
