#!/usr/bin/env bash
set -euo pipefail

readonly upstream_repository="https://raw.githubusercontent.com/traPtitech/NeoShowcase"
readonly upstream_commit="${1:-16eda27a8cda8858811406411bcc7f2f508e9efc}"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly script_directory
repository_root="$(cd -- "${script_directory}/.." && pwd)"
readonly repository_root
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

printf 'Updated protobuf sources to NeoShowcase commit %s.\n' "${upstream_commit}"
printf 'Update proto/UPSTREAM.md, then run make proto.\n'
