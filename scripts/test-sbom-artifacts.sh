#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
checker="${repo_root}/scripts/check-sbom-artifacts.sh"
sha="0123456789abcdef0123456789abcdef01234567"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

write_valid_sbom() {
  local component="$1"
  local file="$2"

  python3 - "$component" "$sha" "$file" <<'PY'
import json
import sys

component = sys.argv[1]
sha = sys.argv[2]
path = sys.argv[3]

data = {
    "bomFormat": "CycloneDX",
    "specVersion": "1.6",
    "version": 1,
    "metadata": {
        "component": {
            "type": "container",
            "name": component,
            "version": sha,
        }
    },
    "components": [
        {
            "type": "library",
            "name": "example-package",
            "version": "1.0.0",
        }
    ],
}

with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle)
PY
}

write_valid_fixture() {
  local sbom_dir="$1"
  mkdir -p "${sbom_dir}"

  local image
  for image in openoms-api openoms-dashboard openoms-migrate; do
    write_valid_sbom "${image}" "${sbom_dir}/${image}-${sha}.cdx.json"
    printf 'ghcr.io/openoms-org/%s@sha256:%064d\n' "${image}" 1 > "${sbom_dir}/${image}-${sha}.digest.txt"
  done
}

expect_failure() {
  local label="$1"
  local output_file="${tmp_dir}/openoms-sbom-test.out"
  local error_file="${tmp_dir}/openoms-sbom-test.err"
  shift

  if "$@" >"${output_file}" 2>"${error_file}"; then
    echo "expected failure: ${label}" >&2
    cat "${output_file}" >&2
    cat "${error_file}" >&2
    exit 1
  fi
}

valid_dir="${tmp_dir}/valid"
write_valid_fixture "${valid_dir}"
SBOM_DIR="${valid_dir}" "${checker}" "${sha}"

missing_digest_dir="${tmp_dir}/missing-digest"
write_valid_fixture "${missing_digest_dir}"
rm "${missing_digest_dir}/openoms-api-${sha}.digest.txt"
expect_failure "missing digest" env SBOM_DIR="${missing_digest_dir}" "${checker}" "${sha}"

invalid_bom_dir="${tmp_dir}/invalid-bom"
write_valid_fixture "${invalid_bom_dir}"
python3 - "${invalid_bom_dir}/openoms-dashboard-${sha}.cdx.json" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, encoding="utf-8") as handle:
    data = json.load(handle)
data["bomFormat"] = "SPDX"
with open(path, "w", encoding="utf-8") as handle:
    json.dump(data, handle)
PY
expect_failure "invalid CycloneDX format" env SBOM_DIR="${invalid_bom_dir}" "${checker}" "${sha}"

invalid_digest_dir="${tmp_dir}/invalid-digest"
write_valid_fixture "${invalid_digest_dir}"
printf 'ghcr.io/openoms-org/openoms-migrate:latest\n' > "${invalid_digest_dir}/openoms-migrate-${sha}.digest.txt"
expect_failure "invalid digest reference" env SBOM_DIR="${invalid_digest_dir}" "${checker}" "${sha}"

echo "sbom artifact checks passed"
