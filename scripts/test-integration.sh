#!/usr/bin/env bash
set -euo pipefail

readonly upstream_repository="https://github.com/traPtitech/NeoShowcase.git"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly script_directory
repository_root="$(cd -- "${script_directory}/.." && pwd)"
readonly repository_root
revision_file="${repository_root}/proto/upstream-revision"
readonly revision_file
upstream_revision="$(tr -d '[:space:]' <"${revision_file}")"
readonly upstream_revision
temporary_directory="$(mktemp -d)"
readonly temporary_directory
certificates_directory="${temporary_directory}/certificates"
readonly certificates_directory
upstream_directory="${temporary_directory}/neoshowcase"
readonly upstream_directory
compose_project="tfns-${PPID}-$$"
readonly compose_project
export COMPOSE_PROJECT_NAME="${compose_project}"
export COMPOSE_PROGRESS="${COMPOSE_PROGRESS:-plain}"
export NEOSHOWCASE_TEST_CERTS="${certificates_directory}"

compose_started=false
compose=(
  docker compose
  --project-directory "${upstream_directory}"
  -f "${upstream_directory}/compose.yaml"
  -f "${repository_root}/acceptance/compose.yaml"
  -p "${compose_project}"
)

cleanup() {
  local exit_status=$?
  trap - EXIT INT TERM
  if [[ "${compose_started}" == true ]]; then
    if ((exit_status != 0)); then
      "${compose[@]}" logs --no-color --tail=300 >&2 || true
    fi
    "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
  fi
  rm -rf -- "${temporary_directory}"
  exit "${exit_status}"
}
trap cleanup EXIT INT TERM

for command in curl docker git go jq openssl ssh-keyscan; do
  if ! command -v "${command}" >/dev/null 2>&1; then
    printf 'Required command not found: %s\n' "${command}" >&2
    exit 1
  fi
done

mkdir -p -- "${certificates_directory}"
touch "${certificates_directory}/known_hosts"
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "${certificates_directory}/gitea.key" \
  -out "${certificates_directory}/gitea.crt" \
  -days 1 \
  -subj '/CN=gitea' \
  -addext 'subjectAltName=DNS:gitea,IP:127.0.0.1' >/dev/null 2>&1
chmod 0644 "${certificates_directory}/gitea.crt" "${certificates_directory}/gitea.key"

if [[ -n "${NEOSHOWCASE_SOURCE:-}" ]]; then
  git -C "${NEOSHOWCASE_SOURCE}" cat-file -e "${upstream_revision}^{commit}"
  git clone --quiet --shared --no-checkout "${NEOSHOWCASE_SOURCE}" "${upstream_directory}"
  git -C "${upstream_directory}" checkout --quiet --detach "${upstream_revision}"
else
  git init --quiet "${upstream_directory}"
  git -C "${upstream_directory}" remote add origin "${upstream_repository}"
  git -C "${upstream_directory}" fetch --quiet --depth=1 origin "${upstream_revision}"
  git -C "${upstream_directory}" checkout --quiet --detach FETCH_HEAD
fi

actual_revision="$(git -C "${upstream_directory}" rev-parse HEAD)"
if [[ "${actual_revision}" != "${upstream_revision}" ]]; then
  printf 'NeoShowcase revision mismatch: got %s, want %s\n' "${actual_revision}" "${upstream_revision}" >&2
  exit 1
fi

# The upstream development configuration uses a global network name. Give the
# disposable test environment its own network so parallel and local workloads
# cannot be affected by the controller.
sed -i "s/network: neoshowcase_apps/network: ${compose_project}_apps/" \
  "${upstream_directory}/.local-dev/config/ns.yaml"

"${compose[@]}" config --quiet
compose_started=true
"${compose[@]}" up --detach --build --wait gitea ns-controller ns-gateway

gitea_binding="$("${compose[@]}" port gitea 3000)"
gitea_ssh_binding="$("${compose[@]}" port gitea 2222)"
gateway_binding="$("${compose[@]}" port ns-gateway 8080)"
readonly gitea_binding gitea_ssh_binding gateway_binding
gitea_endpoint="https://${gitea_binding}"
gateway_endpoint="http://${gateway_binding}"
readonly gitea_endpoint gateway_endpoint

readonly gitea_admin_user="integration-admin"
readonly gitea_admin_password="integration-admin-password"
readonly gitea_reader_one="git-reader-one"
readonly gitea_reader_one_password="reader-one-password"
readonly gitea_reader_two="git-reader-two"
readonly gitea_reader_two_password="reader-two-password"

gitea_ssh_port="${gitea_ssh_binding##*:}"
readonly gitea_ssh_port
for _ in $(seq 1 60); do
  if ssh-keyscan -p "${gitea_ssh_port}" 127.0.0.1 2>/dev/null |
    sed -E 's/^\[[^]]+\]:[0-9]+/[gitea]:2222/' >"${certificates_directory}/known_hosts" &&
    [[ -s "${certificates_directory}/known_hosts" ]]; then
    break
  fi
  sleep 1
done
if [[ ! -s "${certificates_directory}/known_hosts" ]]; then
  printf 'Gitea SSH server did not become ready\n' >&2
  exit 1
fi

"${compose[@]}" exec --no-TTY --user git gitea gitea admin user create \
  --username "${gitea_admin_user}" \
  --password "${gitea_admin_password}" \
  --email integration-admin@example.invalid \
  --admin \
  --must-change-password=false
"${compose[@]}" exec --no-TTY --user git gitea gitea admin user create \
  --username "${gitea_reader_one}" \
  --password "${gitea_reader_one_password}" \
  --email reader-one@example.invalid \
  --must-change-password=false
"${compose[@]}" exec --no-TTY --user git gitea gitea admin user create \
  --username "${gitea_reader_two}" \
  --password "${gitea_reader_two_password}" \
  --email reader-two@example.invalid \
  --must-change-password=false

gitea_api() {
  curl --fail-with-body --silent --show-error \
    --cacert "${certificates_directory}/gitea.crt" \
    --user "${gitea_admin_user}:${gitea_admin_password}" \
    --header 'Content-Type: application/json' \
    "$@"
}

gitea_api --request POST --data '{"name":"public-one","private":false,"auto_init":true,"default_branch":"main"}' \
  "${gitea_endpoint}/api/v1/user/repos" >/dev/null
gitea_api --request POST --data '{"name":"public-two","private":false,"auto_init":true,"default_branch":"main"}' \
  "${gitea_endpoint}/api/v1/user/repos" >/dev/null
gitea_api --request POST --data '{"name":"private","private":true,"auto_init":true,"default_branch":"main"}' \
  "${gitea_endpoint}/api/v1/user/repos" >/dev/null
system_public_key="$(<"${upstream_directory}/.local-dev/keys/id_ed25519.pub")"
readonly system_public_key
gitea_api --request POST \
  --data "$(jq --null-input --arg key "${system_public_key}" '{title: "neoshowcase-system-key", key: $key, read_only: true}')" \
  "${gitea_endpoint}/api/v1/user/keys" >/dev/null
gitea_api --request PUT --data '{"permission":"read"}' \
  "${gitea_endpoint}/api/v1/repos/${gitea_admin_user}/private/collaborators/${gitea_reader_one}" >/dev/null
gitea_api --request PUT --data '{"permission":"read"}' \
  "${gitea_endpoint}/api/v1/repos/${gitea_admin_user}/private/collaborators/${gitea_reader_two}" >/dev/null

for _ in $(seq 1 60); do
  if curl --fail --silent --show-error \
    --header 'Content-Type: application/json' \
    --header 'X-Showcase-User: integration-readiness' \
    --data '{}' \
    "${gateway_endpoint}/neoshowcase.protobuf.APIService/GetMe" >/dev/null; then
    break
  fi
  sleep 1
done
curl --fail-with-body --silent --show-error \
  --header 'Content-Type: application/json' \
  --header 'X-Showcase-User: integration-readiness' \
  --data '{}' \
  "${gateway_endpoint}/neoshowcase.protobuf.APIService/GetSystemInfo" >/dev/null

additional_owner_id="$(
  curl --fail-with-body --silent --show-error \
    --header 'Content-Type: application/json' \
    --header "X-Showcase-User: ${compose_project}-additional-owner" \
    --data '{}' \
    "${gateway_endpoint}/neoshowcase.protobuf.APIService/GetMe" | jq --exit-status --raw-output '.id'
)"
readonly additional_owner_id

export TF_ACC=1
export NEOSHOWCASE_ACC=1
export NEOSHOWCASE_TEST_ENDPOINT="${gateway_endpoint}"
export NEOSHOWCASE_TEST_RUN_ID="${compose_project}"
export NEOSHOWCASE_TEST_PUBLIC_URL_ONE="https://gitea:3000/${gitea_admin_user}/public-one.git"
export NEOSHOWCASE_TEST_PUBLIC_URL_TWO="https://gitea:3000/${gitea_admin_user}/public-two.git"
export NEOSHOWCASE_TEST_PRIVATE_URL="https://gitea:3000/${gitea_admin_user}/private.git"
export NEOSHOWCASE_TEST_PRIVATE_SSH_URL="ssh://git@gitea:2222/${gitea_admin_user}/private.git"
export NEOSHOWCASE_TEST_ADDITIONAL_OWNER_ID="${additional_owner_id}"
export NEOSHOWCASE_TEST_BASIC_USER_ONE="${gitea_reader_one}"
export NEOSHOWCASE_TEST_BASIC_PASSWORD_ONE="${gitea_reader_one_password}"
export NEOSHOWCASE_TEST_BASIC_USER_TWO="${gitea_reader_two}"
export NEOSHOWCASE_TEST_BASIC_PASSWORD_TWO="${gitea_reader_two_password}"

go test -count=1 -tags=acceptance -run '^TestAccReal' -v ./acceptance
