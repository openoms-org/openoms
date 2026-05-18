#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="${repo_root}/scripts/ensure-release-run-for-sha.sh"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

if [[ ! -x "${target}" ]]; then
  echo "fallback script is missing or not executable: ${target}" >&2
  exit 1
fi

make_fake_gh() {
  local mode="$1"
  local fake_bin="${tmp_dir}/${mode}/bin"
  mkdir -p "${fake_bin}"

  cat > "${fake_bin}/gh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

mode="${OPENOMS_FAKE_GH_MODE:?}"
log_file="${OPENOMS_FAKE_GH_LOG:?}"

printf '%s\n' "$*" >> "${log_file}"

case "${mode}:${1:-}:${2:-}" in
  existing:run:list)
    cat <<'JSON'
[{"databaseId":26006443776,"workflowName":"Release","event":"push","status":"completed","conclusion":"success","headSha":"0123456789abcdef0123456789abcdef01234567","url":"https://github.com/openoms-org/openoms/actions/runs/26006443776"}]
JSON
    ;;
  missing:run:list)
    printf '[]\n'
    ;;
  missing:workflow:run)
    printf 'dispatch ok\n'
    ;;
  dispatch-fails:run:list)
    printf '[]\n'
    ;;
  dispatch-fails:workflow:run)
    echo "dispatch failed" >&2
    exit 42
    ;;
  *)
    echo "unexpected gh call for mode ${mode}: $*" >&2
    exit 99
    ;;
esac
SH

  chmod +x "${fake_bin}/gh"
  printf '%s\n' "${fake_bin}"
}

run_with_fake_gh() {
  local mode="$1"
  local log_file="${tmp_dir}/${mode}.log"
  local fake_bin
  fake_bin="$(make_fake_gh "${mode}")"

  OPENOMS_FAKE_GH_MODE="${mode}" \
  OPENOMS_FAKE_GH_LOG="${log_file}" \
  PATH="${fake_bin}:${PATH}" \
  GITHUB_REPOSITORY="openoms-org/openoms" \
  RELEASE_SHA="0123456789abcdef0123456789abcdef01234567" \
  RELEASE_WORKFLOW="release.yml" \
  RELEASE_REF="main" \
    "${target}"

  cat "${log_file}"
}

existing_calls="$(run_with_fake_gh existing)"
if grep -Fq "workflow run" <<<"${existing_calls}"; then
  echo "existing release run should not dispatch" >&2
  exit 1
fi

missing_calls="$(run_with_fake_gh missing)"
if ! grep -Fq "workflow run release.yml --repo openoms-org/openoms --ref main" <<<"${missing_calls}"; then
  echo "missing release run should dispatch release.yml on main" >&2
  echo "${missing_calls}" >&2
  exit 1
fi

if GITHUB_REPOSITORY="openoms-org/openoms" RELEASE_SHA="" RELEASE_WORKFLOW="release.yml" RELEASE_REF="main" "${target}" >/tmp/openoms-ope435.out 2>/tmp/openoms-ope435.err; then
  echo "empty RELEASE_SHA should fail" >&2
  exit 1
fi

fake_bin="$(make_fake_gh dispatch-fails)"
if OPENOMS_FAKE_GH_MODE="dispatch-fails" \
  OPENOMS_FAKE_GH_LOG="${tmp_dir}/dispatch-fails.log" \
  PATH="${fake_bin}:${PATH}" \
  GITHUB_REPOSITORY="openoms-org/openoms" \
  RELEASE_SHA="0123456789abcdef0123456789abcdef01234567" \
  RELEASE_WORKFLOW="release.yml" \
  RELEASE_REF="main" \
    "${target}" >/tmp/openoms-ope435-dispatch.out 2>/tmp/openoms-ope435-dispatch.err; then
  echo "dispatch failure should fail the fallback job" >&2
  exit 1
fi
