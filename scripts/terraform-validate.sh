#!/usr/bin/env bash
set -euo pipefail

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly script_directory
repository_root="$(cd -- "${script_directory}/.." && pwd)"
readonly repository_root
readonly fixture_directory="${repository_root}/internal/provider/testdata/terraform-validation"
temporary_directory="$(mktemp -d)"
readonly temporary_directory
readonly provider_directory="${temporary_directory}/providers"
readonly cli_config="${temporary_directory}/terraform.tfrc"

cleanup() {
  rm -rf -- "${temporary_directory}"
}
trap cleanup EXIT

mkdir -p -- "${provider_directory}"
go build -o "${provider_directory}/terraform-provider-neoshowcase" "${repository_root}"
sed "s|@PROVIDER_DIRECTORY@|${provider_directory}|g" \
  "${fixture_directory}/terraform.tfrc.tmpl" >"${cli_config}"

TF_CLI_CONFIG_FILE="${cli_config}" TF_IN_AUTOMATION=1 \
  terraform -chdir="${fixture_directory}" validate -no-color
