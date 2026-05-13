#!/usr/bin/env bash
set -euo pipefail

bundle_dir="${1:-}"
if [ -z "$bundle_dir" ]; then
  echo "usage: $0 <bundle-dir>" >&2
  exit 2
fi

if [ ! -d "$bundle_dir" ]; then
  echo "dashboard bundle placeholder check failed: bundle directory does not exist: $bundle_dir" >&2
  exit 1
fi

fail() {
  echo "dashboard bundle placeholder check failed: $*" >&2
  exit 1
}

pattern='NEXT_PUBLIC_API_URL_PLACEHOLDER|WS_CSP_HOST_PLACEHOLDER|SENTRY_DSN_PLACEHOLDER|http://localhost:8080'
matches_file="$(mktemp)"
cleanup() {
  rm -f "$matches_file"
}
trap cleanup EXIT

while IFS= read -r -d '' file; do
  set +e
  output="$(grep -I -H -n -E "$pattern" "$file" 2>&1)"
  rc=$?
  set -e

  if [ "$rc" -eq 0 ]; then
    printf "%s\n" "$output" >> "$matches_file"
    continue
  fi

  if [ "$rc" -ne 1 ]; then
    fail "could not scan $file: $output"
  fi
done < <(find "$bundle_dir" -type f -print0)

if [ -s "$matches_file" ]; then
  cat "$matches_file"
  fail "runtime bundle contains forbidden API, Sentry, or localhost placeholder"
fi

echo "bundle_placeholder_check=pass"
