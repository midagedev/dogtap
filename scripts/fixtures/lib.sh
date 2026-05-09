#!/usr/bin/env bash

fixture_repo_root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "${script_dir}/../.." && pwd
}

fixture_artifact_dir() {
  local root="$1"
  printf '%s\n' "${DOGTAP_FIXTURE_ARTIFACT_DIR:-${root}/testdata/g1-evidence/latest}"
}

fixture_write_config() {
  local root="$1"
  local artifact_dir="$2"
  local config_path="${artifact_dir}/dogtap.capture.yaml"
  mkdir -p "${artifact_dir}"
  cat > "${config_path}" <<EOF
mode: local
server:
  httpAddr: "127.0.0.1:${DOGTAP_HTTP_PORT:-18080}"
  apmAddr: "127.0.0.1:${DOGTAP_APM_PORT:-18126}"
  otlpHttpAddr: "127.0.0.1:${DOGTAP_OTLP_HTTP_PORT:-14318}"
  grpcAddr: "127.0.0.1:${DOGTAP_OTLP_GRPC_PORT:-14317}"
storage:
  kind: file
  path: "${artifact_dir}/events-store.json"
  maxEvents: 2000
  ttl: 2h
validation:
  required:
    serviceTags: true
    rum:
      - userId
      - accountId
      - workspaceId
    logs:
      - service
      - env
    apm:
      - service
      - env
      - version
    otlp:
      - service
  pii:
    enabled: true
    failOn:
      - access_token
      - authorization
      - refresh_token
forwarding:
  enabled: false
  site: datadoghq.com
security:
  allowRawPayloads: true
  maxBodyBytes: 10485760
EOF
  printf '%s\n' "${config_path}"
}

fixture_base_url() {
  printf 'http://127.0.0.1:%s\n' "${DOGTAP_HTTP_PORT:-18080}"
}

fixture_apm_url() {
  printf 'http://127.0.0.1:%s\n' "${DOGTAP_APM_PORT:-18126}"
}

fixture_otlp_http_url() {
  printf 'http://127.0.0.1:%s\n' "${DOGTAP_OTLP_HTTP_PORT:-14318}"
}

fixture_otlp_grpc_endpoint() {
  printf '127.0.0.1:%s\n' "${DOGTAP_OTLP_GRPC_PORT:-14317}"
}

fixture_wait_for_dogtap() {
  local base_url="$1"
  local attempts=80
  local i
  for ((i = 1; i <= attempts; i++)); do
    if curl -fsS "${base_url}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

fixture_start_dogtap() {
  local root="$1"
  local artifact_dir="$2"
  local base_url
  base_url="$(fixture_base_url)"

  if curl -fsS "${base_url}/healthz" >/dev/null 2>&1; then
    DOGTAP_STARTED_BY_FIXTURE=""
    return 0
  fi

  local config_path
  config_path="$(fixture_write_config "${root}" "${artifact_dir}")"
  mkdir -p "${artifact_dir}/logs"
  (
    cd "${root}"
    go run ./cmd/dogtap serve -config "${config_path}"
  ) > "${artifact_dir}/logs/dogtap.log" 2>&1 &
  DOGTAP_STARTED_BY_FIXTURE="$!"
  export DOGTAP_STARTED_BY_FIXTURE

  if ! fixture_wait_for_dogtap "${base_url}"; then
    printf 'Dogtap did not become healthy. See %s\n' "${artifact_dir}/logs/dogtap.log" >&2
    return 1
  fi
}

fixture_stop_dogtap() {
  if [ -n "${DOGTAP_STARTED_BY_FIXTURE:-}" ]; then
    kill "${DOGTAP_STARTED_BY_FIXTURE}" >/dev/null 2>&1 || true
    wait "${DOGTAP_STARTED_BY_FIXTURE}" >/dev/null 2>&1 || true
  fi
}

fixture_write_not_run() {
  local path="$1"
  shift
  mkdir -p "$(dirname "${path}")"
  {
    printf 'This fixture capture did not run.\n\n'
    printf '%s\n' "$@"
  } > "${path}"
}

