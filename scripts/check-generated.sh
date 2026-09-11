#!/usr/bin/env bash
set -euo pipefail

readonly script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly repository_root="$(cd -- "${script_directory}/.." && pwd)"

cd "${repository_root}/proto"
go run github.com/bufbuild/buf/cmd/buf@v1.72.0 generate
cd "${repository_root}"

git diff --exit-code -- internal/neoshowcase/gen
