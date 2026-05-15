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
