#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "dashboard release config check failed: $*" >&2
  exit 1
}

require_non_empty() {
  local name="$1"
  local value="${!name:-}"
  if [ -z "$value" ]; then
    fail "$name must be configured for production dashboard releases"
  fi
}

require_non_empty NEXT_PUBLIC_SENTRY_DSN
require_non_empty SENTRY_ORG
require_non_empty SENTRY_PROJECT
require_non_empty SENTRY_AUTH_TOKEN

case "$NEXT_PUBLIC_SENTRY_DSN" in
  https://*) ;;
  *) fail "NEXT_PUBLIC_SENTRY_DSN must start with https://" ;;
esac

if printf '%s' "$NEXT_PUBLIC_SENTRY_DSN" | grep -qi 'placeholder'; then
  fail "NEXT_PUBLIC_SENTRY_DSN must not be a placeholder"
fi

echo "dashboard_release_config_check=pass"
