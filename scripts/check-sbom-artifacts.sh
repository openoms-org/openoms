#!/usr/bin/env bash
set -euo pipefail

sha="${1:-}"
sbom_dir="${SBOM_DIR:-sbom}"

if [[ -z "${sha}" ]]; then
  echo "usage: $0 <git-sha>" >&2
  exit 2
fi

if [[ ! -d "${sbom_dir}" ]]; then
  echo "SBOM directory not found: ${sbom_dir}" >&2
  exit 1
fi

validate_sbom_json() {
  local file="$1"
  local expected_component="$2"
  local expected_sha="$3"

  python3 - "$file" "$expected_component" "$expected_sha" <<'PY'
import json
import sys

path = sys.argv[1]
expected_component = sys.argv[2]
expected_sha = sys.argv[3]

with open(path, encoding="utf-8") as handle:
    data = json.load(handle)

if data.get("bomFormat") != "CycloneDX":
    raise SystemExit(f"{path}: expected CycloneDX bomFormat")

if not str(data.get("specVersion", "")).strip():
    raise SystemExit(f"{path}: missing specVersion")

metadata = data.get("metadata")
if not isinstance(metadata, dict):
    raise SystemExit(f"{path}: missing metadata")

component = metadata.get("component")
if not isinstance(component, dict):
    raise SystemExit(f"{path}: missing metadata.component")

name = component.get("name")
if name != expected_component:
    raise SystemExit(f"{path}: expected component {expected_component}, got {name!r}")

version = component.get("version")
if version not in (expected_sha, f"sha256:{expected_sha}"):
    raise SystemExit(f"{path}: expected component version to include release sha")

components = data.get("components")
if not isinstance(components, list) or not components:
    raise SystemExit(f"{path}: expected non-empty components list")

if expected_component == "openoms-dashboard":
    unknown_npm = []
    for item in components:
        purl = str(item.get("purl") or "")
        version = str(item.get("version") or "").strip()
        if purl.startswith("pkg:npm/") and (not version or version.upper() == "UNKNOWN"):
            locations = [
                str(prop.get("value") or "")
                for prop in item.get("properties", [])
                if str(prop.get("name") or "").startswith("syft:location")
            ]
            unknown_npm.append((item.get("name"), purl, locations))

    if unknown_npm:
        details = "; ".join(
            f"{name} ({purl}) at {', '.join(locations) or 'unknown location'}"
            for name, purl, locations in unknown_npm[:10]
        )
        raise SystemExit(
            f"{path}: dashboard SBOM contains npm components with unknown versions: {details}"
        )
PY
}

validate_digest() {
  local file="$1"
  local image="$2"
  local expected_ref="ghcr.io/openoms-org/${image}@sha256:"
  local digest
  local hex_digest

  digest="$(tr -d '\r\n' < "${file}")"
  if [[ "${digest}" != "${expected_ref}"* ]]; then
    echo "${file}: invalid digest reference: ${digest}" >&2
    exit 1
  fi

  hex_digest="${digest#"${expected_ref}"}"
  if [[ ! "${hex_digest}" =~ ^[a-f0-9]{64}$ ]]; then
    echo "${file}: invalid sha256 digest: ${digest}" >&2
    exit 1
  fi
}

for image in openoms-api openoms-dashboard openoms-migrate; do
  sbom_file="${sbom_dir}/${image}-${sha}.cdx.json"
  digest_file="${sbom_dir}/${image}-${sha}.digest.txt"

  if [[ ! -s "${sbom_file}" ]]; then
    echo "missing or empty SBOM file: ${sbom_file}" >&2
    exit 1
  fi

  if [[ ! -s "${digest_file}" ]]; then
    echo "missing or empty digest file: ${digest_file}" >&2
    exit 1
  fi

  validate_sbom_json "${sbom_file}" "${image}" "${sha}"
  validate_digest "${digest_file}" "${image}"
done

echo "SBOM artifacts validated for ${sha}"
