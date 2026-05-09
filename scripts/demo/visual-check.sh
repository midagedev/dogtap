#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"

http_port="${DOGTAP_DEMO_HTTP_PORT:-18090}"
apm_port="${DOGTAP_DEMO_APM_PORT:-18126}"
otlp_http_port="${DOGTAP_DEMO_OTLP_HTTP_PORT:-14318}"
grpc_port="${DOGTAP_DEMO_GRPC_PORT:-14317}"
base_url="http://127.0.0.1:${http_port}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/dogtap-demo-visual.XXXXXX")"
log_file="${tmp_dir}/dogtap.log"
pid=""

cleanup() {
  if [ -n "${pid}" ] && kill -0 "${pid}" >/dev/null 2>&1; then
    kill "${pid}" >/dev/null 2>&1 || true
    wait "${pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

require_port_free() {
  local port="$1"
  if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port ${port} is already in use. Override DOGTAP_DEMO_*_PORT or stop the listener." >&2
    exit 2
  fi
}

wait_for_health() {
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "${base_url}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    if [ -n "${pid}" ] && ! kill -0 "${pid}" >/dev/null 2>&1; then
      echo "Dogtap exited before becoming healthy." >&2
      cat "${log_file}" >&2 || true
      exit 3
    fi
    sleep 0.5
  done
  echo "Dogtap did not become healthy in time." >&2
  cat "${log_file}" >&2 || true
  exit 3
}

main() {
  require_port_free "${http_port}"
  require_port_free "${apm_port}"
  require_port_free "${otlp_http_port}"
  require_port_free "${grpc_port}"

  if [ "${DOGTAP_DEMO_SKIP_WEB_BUILD:-0}" != "1" ]; then
    npm --prefix "${repo_root}/web" run build
  fi

  (
    cd "${repo_root}"
    DOGTAP_MODE=local \
      DOGTAP_HTTP_ADDR="127.0.0.1:${http_port}" \
      DOGTAP_APM_ADDR="127.0.0.1:${apm_port}" \
      DOGTAP_OTLP_HTTP_ADDR="127.0.0.1:${otlp_http_port}" \
      DOGTAP_GRPC_ADDR="127.0.0.1:${grpc_port}" \
      DOGTAP_STORAGE_KIND=memory \
      go run ./cmd/dogtap serve
  ) >"${log_file}" 2>&1 &
  pid="$!"

  wait_for_health

  DOGTAP_DEMO_BASE_URL="${base_url}" \
    DOGTAP_DEMO_APM_URL="http://127.0.0.1:${apm_port}" \
    DOGTAP_DEMO_OTLP_URL="http://127.0.0.1:${otlp_http_port}" \
    bash "${script_dir}/seed.sh"

  diagnostics_dir="${DOGTAP_ARTIFACT_DIR:-${repo_root}/.dogtap/diagnostics/demo}"
  (
    cd "${repo_root}"
    go run ./cmd/dogtap diagnose \
      -base-url "${base_url}" \
      -output "${diagnostics_dir}" \
      -expect-non-empty \
      -expect-source rum,logs,apm,otlp \
      -expect-payload-kind replay,metric \
      -expect-service web-frontend,api-service,edge-gateway \
      -expect-session session-123 \
      -expect-trace 123456789 \
      -expect-metric http.server.request.duration
  )
  cp "${log_file}" "${diagnostics_dir}/dogtap.log"

  (
    cd "${repo_root}"
    DOGTAP_LIVE_E2E=1 \
      DOGTAP_E2E_BASE_URL="${base_url}" \
      npm --prefix web run test:e2e -- demo-live.spec.ts
  )

  echo "Dogtap demo visual check passed."
  echo "Screenshots: ${repo_root}/web/test-results"
  echo "Diagnostics: ${diagnostics_dir}"
}

main "$@"
