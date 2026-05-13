#!/usr/bin/env bash
set -euo pipefail

image="${1:-}"
if [ -z "$image" ]; then
  echo "usage: $0 <dashboard-image>" >&2
  exit 2
fi

fail() {
  echo "dashboard image check failed: $*" >&2
  exit 1
}

image_entrypoint="$(docker image inspect "$image" --format '{{json .Config.Entrypoint}}')"
image_cmd="$(docker image inspect "$image" --format '{{json .Config.Cmd}}')"
image_user="$(docker image inspect "$image" --format '{{.Config.User}}')"

echo "dashboard_image=$image"
echo "image_entrypoint=$image_entrypoint"
echo "image_cmd=$image_cmd"
echo "image_user=$image_user"

case "$image_entrypoint" in
  *sh*|*docker-entrypoint*) fail "entrypoint must not use shell tooling: $image_entrypoint" ;;
esac

if docker run --rm --entrypoint /bin/sh "$image" -c 'true' >/tmp/openoms-dashboard-shell-check.out 2>&1; then
  fail "runtime image unexpectedly contains /bin/sh"
fi
echo "shell_check=absent"

tmp_dir="$(mktemp -d)"
container_id=""
smoke_id=""
cleanup() {
  if [ -n "$container_id" ]; then
    docker rm "$container_id" >/dev/null 2>&1 || true
  fi
  if [ -n "$smoke_id" ]; then
    docker stop "$smoke_id" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

container_id="$(docker create "$image")"
docker cp "$container_id:/app/.next" "$tmp_dir/.next"
docker cp "$container_id:/app/server.js" "$tmp_dir/server.js"
docker rm "$container_id" >/dev/null
container_id=""

"$(dirname "${BASH_SOURCE[0]}")/check-dashboard-bundle-placeholders.sh" "$tmp_dir"

smoke_id="$(docker run -d --rm --read-only \
  --tmpfs /tmp:rw,size=100m \
  --tmpfs /app/.next/cache:rw,size=100m \
  -p 127.0.0.1::3000 \
  "$image")"

runtime_path="$(docker inspect "$smoke_id" --format '{{.Path}}')"
runtime_args="$(docker inspect "$smoke_id" --format '{{json .Args}}')"
runtime_user="$(docker inspect "$smoke_id" --format '{{.Config.User}}')"
echo "runtime_path=$runtime_path"
echo "runtime_args=$runtime_args"
echo "runtime_user=$runtime_user"

case "$runtime_path" in
  *node*) ;;
  *) fail "expected default runtime process to be Node, got $runtime_path" ;;
esac

case "$runtime_args" in
  *server.js*) ;;
  *) fail "expected default runtime args to include server.js, got $runtime_args" ;;
esac

case "$runtime_user" in
  ""|"0"|"root"|"0:0"|"root:root") fail "expected non-root runtime user, got '${runtime_user:-<empty>}'" ;;
esac

host_port=""
for _ in $(seq 1 20); do
  host_port="$(docker port "$smoke_id" 3000/tcp 2>/dev/null | awk -F: 'NR==1 {print $NF}')"
  if [ -n "$host_port" ]; then
    break
  fi
  sleep 0.5
done
[ -n "$host_port" ] || fail "could not resolve mapped dashboard port"

for _ in $(seq 1 40); do
  status="$(curl -s -o /tmp/openoms-dashboard-image-smoke.html -w '%{http_code}' "http://127.0.0.1:${host_port}/login" || true)"
  case "$status" in
    200|307|308)
      echo "read_only_smoke_http=$status"
      exit 0
      ;;
  esac
  sleep 1
done

docker logs "$smoke_id" >&2 || true
fail "dashboard did not serve /login from read-only root filesystem"
