#!/usr/bin/env bash
set -euo pipefail

script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly script_directory
repository_root="$(cd -- "${script_directory}/.." && pwd)"
readonly repository_root
readonly buf_version="1.72.0"

cd "${repository_root}/proto"
if command -v buf >/dev/null && [[ "$(buf --version)" == "${buf_version}" ]]; then
  buf generate
else
  go run "github.com/bufbuild/buf/cmd/buf@v${buf_version}" generate
fi
cd "${repository_root}"

git diff --exit-code -- internal/neoshowcase/gen
