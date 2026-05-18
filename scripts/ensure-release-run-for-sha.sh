#!/usr/bin/env bash
set -euo pipefail

repo="${GITHUB_REPOSITORY:-}"
sha="${RELEASE_SHA:-}"
workflow="${RELEASE_WORKFLOW:-release.yml}"
ref="${RELEASE_REF:-main}"
dry_run="${DRY_RUN:-0}"

require_env() {
  local name="$1"
  local value="$2"

  if [[ -z "${value}" ]]; then
    echo "${name} is required" >&2
    exit 2
  fi
}

require_env "GITHUB_REPOSITORY" "${repo}"
require_env "RELEASE_SHA" "${sha}"
require_env "RELEASE_WORKFLOW" "${workflow}"
require_env "RELEASE_REF" "${ref}"

existing_runs="$(
  gh run list \
    --repo "${repo}" \
    --workflow "${workflow}" \
    --commit "${sha}" \
    --limit 20 \
    --json databaseId,workflowName,event,status,conclusion,headSha,url
)"

if OPENOMS_RELEASE_RUNS_JSON="${existing_runs}" python3 - "${sha}" <<'PY'; then
import json
import os
import sys

sha = sys.argv[1]
runs = json.loads(os.environ["OPENOMS_RELEASE_RUNS_JSON"])
active_statuses = {"queued", "in_progress", "completed", "waiting", "requested", "pending"}

for run in runs:
    if run.get("headSha") == sha and run.get("status") in active_statuses:
        print(
            f"Release run already exists for {sha}: "
            f"{run.get('databaseId')} {run.get('status')} {run.get('conclusion') or ''} {run.get('url') or ''}"
        )
        raise SystemExit(0)

raise SystemExit(1)
PY
  exit 0
fi

echo "No release run found for ${sha}; dispatching ${workflow} on ${ref}"

if [[ "${dry_run}" == "1" ]]; then
  echo "DRY_RUN=1, not dispatching"
  exit 0
fi

gh workflow run "${workflow}" --repo "${repo}" --ref "${ref}"
