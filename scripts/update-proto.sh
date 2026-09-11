#!/usr/bin/env bash
set -euo pipefail

readonly upstream_repository="https://raw.githubusercontent.com/traPtitech/NeoShowcase"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly script_directory
repository_root="$(cd -- "${script_directory}/.." && pwd)"
readonly repository_root
revision_file="${repository_root}/proto/upstream-revision"
readonly revision_file
upstream_commit="${1:-$(tr -d '[:space:]' <"${revision_file}")}"
readonly upstream_commit
readonly destination="${repository_root}/proto/neoshowcase/protobuf"
temporary_directory="$(mktemp -d)"
readonly temporary_directory

cleanup() {
  rm -rf -- "${temporary_directory}"
}
trap cleanup EXIT

for filename in gateway.proto null.proto; do
  curl --fail --location --silent --show-error \
    "${upstream_repository}/${upstream_commit}/api/proto/neoshowcase/protobuf/${filename}" \
    --output "${temporary_directory}/${filename}"
done

install -m 0644 "${temporary_directory}/gateway.proto" "${destination}/gateway.proto"
install -m 0644 "${temporary_directory}/null.proto" "${destination}/null.proto"
printf '%s\n' "${upstream_commit}" >"${revision_file}"

printf 'Updated protobuf sources to NeoShowcase commit %s.\n' "${upstream_commit}"
printf 'Run nix run .#proto to regenerate the Go bindings.\n'
