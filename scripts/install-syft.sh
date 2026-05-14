#!/usr/bin/env bash
set -euo pipefail

version="${1:-v1.44.0}"
install_dir="${SYFT_INSTALL_DIR:-/usr/local/bin}"

if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid Syft version: ${version}" >&2
  exit 2
fi

version_number="${version#v}"
checksums="syft_${version_number}_checksums.txt"
base_url="https://github.com/anchore/syft/releases/download/${version}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

case "$(uname -s)" in
  Linux)
    os="linux"
    ;;
  Darwin)
    os="darwin"
    ;;
  *)
    echo "unsupported OS for Syft install: $(uname -s)" >&2
    exit 2
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64)
    arch="amd64"
    ;;
  arm64 | aarch64)
    arch="arm64"
    ;;
  *)
    echo "unsupported architecture for Syft install: $(uname -m)" >&2
    exit 2
    ;;
esac

archive="syft_${version_number}_${os}_${arch}.tar.gz"

curl --retry 3 --retry-all-errors -fsSLo "${tmp_dir}/${archive}" "${base_url}/${archive}"
curl --retry 3 --retry-all-errors -fsSLo "${tmp_dir}/${checksums}" "${base_url}/${checksums}"

(
  cd "${tmp_dir}"
  grep " ${archive}$" "${checksums}" | sha256sum -c -
  tar -xzf "${archive}" syft
)

if [[ -w "${install_dir}" ]]; then
  install -m 0755 "${tmp_dir}/syft" "${install_dir}/syft"
else
  sudo install -m 0755 "${tmp_dir}/syft" "${install_dir}/syft"
fi

syft version
