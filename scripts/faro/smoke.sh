#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"

http_port="${DOGTAP_FARO_HTTP_PORT:-19081}"
apm_port="${DOGTAP_FARO_APM_PORT:-19127}"
otlp_http_port="${DOGTAP_FARO_OTLP_HTTP_PORT:-19328}"
grpc_port="${DOGTAP_FARO_GRPC_PORT:-19327}"
frontend_port="${DOGTAP_FARO_FRONTEND_PORT:-13080}"
base_url="http://127.0.0.1:${http_port}"
frontend_url="http://127.0.0.1:${frontend_port}"
sdk_bundle="${DOGTAP_FARO_SDK_BUNDLE:-${repo_root}/web/node_modules/@grafana/faro-web-sdk/dist/bundle/faro-web-sdk.iife.js}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/dogtap-faro-smoke.XXXXXX")"
dogtap_log="${tmp_dir}/dogtap.log"
frontend_log="${tmp_dir}/frontend.log"
dogtap_pid=""
frontend_pid=""

cleanup() {
  if [ -n "${frontend_pid}" ] && kill -0 "${frontend_pid}" >/dev/null 2>&1; then
    kill "${frontend_pid}" >/dev/null 2>&1 || true
    wait "${frontend_pid}" >/dev/null 2>&1 || true
  fi
  if [ -n "${dogtap_pid}" ] && kill -0 "${dogtap_pid}" >/dev/null 2>&1; then
    kill "${dogtap_pid}" >/dev/null 2>&1 || true
    wait "${dogtap_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

require_port_free() {
  local port="$1"
  if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port ${port} is already in use. Override DOGTAP_FARO_*_PORT or stop the listener." >&2
    exit 2
  fi
}

require_file() {
  local path="$1"
  if [ ! -f "${path}" ]; then
    echo "Missing Faro SDK bundle: ${path}" >&2
    echo "Run npm --prefix web ci, or set DOGTAP_FARO_SDK_BUNDLE." >&2
    exit 2
  fi
}

wait_for_url() {
  local url="$1"
  local log_file="$2"
  local pid="$3"
  local attempt
  for attempt in $(seq 1 80); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      echo "Process for ${url} exited before becoming healthy." >&2
      cat "${log_file}" >&2 || true
      exit 3
    fi
    sleep 0.5
  done
  echo "Timed out waiting for ${url}" >&2
  cat "${log_file}" >&2 || true
  exit 3
}

assert_events() {
  local events="$1"
  for expected in \
    '"source":"faro"' \
    '"payloadKind":"event"' \
    '"payloadKind":"log"' \
    '"payloadKind":"metric"' \
    '"service":"faro-smoke-frontend"' \
    '"env":"local"' \
    '"version":"dev"' \
    '"userId":"faro-user-1"' \
    '"accountId":"faro-account-1"' \
    '"workspaceId":"faro-workspace-1"' \
    '"caseId":"faro-case-1"' \
    '"sessionId":"faro-session-1"' \
    '"route":"/faro"' \
    '"name":"faro.workflow.duration"' \
    '"value":42.5' \
    'Faro workflow log'; do
    if ! printf '%s' "${events}" | grep -q "${expected}"; then
      echo "Missing expected Faro event marker: ${expected}" >&2
      printf '%s\n' "${events}" >&2
      exit 1
    fi
  done
}

main() {
  require_port_free "${http_port}"
  require_port_free "${apm_port}"
  require_port_free "${otlp_http_port}"
  require_port_free "${grpc_port}"
  require_port_free "${frontend_port}"
  require_file "${sdk_bundle}"

  (
    cd "${repo_root}"
    DOGTAP_MODE=local \
      DOGTAP_HTTP_ADDR="127.0.0.1:${http_port}" \
      DOGTAP_APM_ADDR="127.0.0.1:${apm_port}" \
      DOGTAP_OTLP_HTTP_ADDR="127.0.0.1:${otlp_http_port}" \
      DOGTAP_GRPC_ADDR="127.0.0.1:${grpc_port}" \
      DOGTAP_STORAGE_KIND=memory \
      go run ./cmd/dogtap serve
  ) >"${dogtap_log}" 2>&1 &
  dogtap_pid="$!"

  wait_for_url "${base_url}/healthz" "${dogtap_log}" "${dogtap_pid}"

  (
    cd "${repo_root}"
    PORT="${frontend_port}" \
      FARO_COLLECTOR_URL="${base_url}/collect/faro-smoke" \
      FARO_SDK_BUNDLE="${sdk_bundle}" \
      node examples/external-injection-smoke/frontend/server.mjs
  ) >"${frontend_log}" 2>&1 &
  frontend_pid="$!"

  wait_for_url "${frontend_url}/healthz" "${frontend_log}" "${frontend_pid}"

  (
    cd "${repo_root}"
    DOGTAP_FARO_E2E=1 \
      DOGTAP_FARO_FRONTEND_URL="${frontend_url}" \
      DOGTAP_FARO_DOGTAP_URL="${base_url}" \
      npm --prefix web run test:e2e -- faro-sdk.spec.ts --project=chromium
  )

  events="$(curl -fsS "${base_url}/api/events?source=faro&limit=100")"
  assert_events "${events}"

  echo "Dogtap Faro SDK smoke passed."
  echo "Frontend: ${frontend_url}/faro"
  echo "Collector: ${base_url}/collect/faro-smoke"
}

main "$@"
