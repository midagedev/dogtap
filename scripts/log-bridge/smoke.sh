#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"

compose_file="${repo_root}/examples/adoption-kit/compose.otel-filelog-bridge.yaml"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/dogtap-log-bridge.XXXXXX")"
project_name="dogtap-log-bridge-${RANDOM}"

http_port="${DOGTAP_LOG_BRIDGE_HTTP_PORT:-19082}"
otlp_http_port="${DOGTAP_LOG_BRIDGE_OTLP_HTTP_PORT:-19319}"

cleanup() {
  compose down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

compose() {
  env \
    DOGTAP_REPO="${repo_root}" \
    DOGTAP_LOG_BRIDGE_HTTP_PORT="${http_port}" \
    DOGTAP_LOG_BRIDGE_OTLP_HTTP_PORT="${otlp_http_port}" \
    docker compose -p "${project_name}" -f "${compose_file}" "$@"
}

require_port_free() {
  local port="$1"
  if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port ${port} is already in use. Override DOGTAP_LOG_BRIDGE_*_PORT or stop the listener." >&2
    exit 2
  fi
}

wait_for_health() {
  local url="$1"
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "Timed out waiting for ${url}" >&2
  compose logs dogtap otel-collector log-writer >&2 || true
  exit 3
}

events_have_expected_logs() {
  local path="$1"
  for expected in \
    '"source":"otlp"' \
    '"payloadKind":"log"' \
    '"service":"filelog-bridge-backend"' \
    '"env":"local"' \
    '"traceId":"4bf92f3577b34da6a3ce929d0e0e4736"' \
    'dogtap filelog bridge started' \
    'dogtap filelog bridge failed' \
    '/bridge/health' \
    '/bridge/jobs'; do
    if ! grep -q "${expected}" "${path}"; then
      return 1
    fi
  done
}

wait_for_events() {
  local attempt
  local events_path="${tmp_dir}/events.json"
  for attempt in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:${http_port}/api/events?limit=100" -o "${events_path}" 2>/dev/null &&
      events_have_expected_logs "${events_path}"; then
      return 0
    fi
    sleep 0.5
  done
  echo "Timed out waiting for filelog bridge events." >&2
  sed -n '1,260p' "${events_path}" >&2 || true
  compose logs dogtap otel-collector log-writer >&2 || true
  exit 4
}

main() {
  require_port_free "${http_port}"
  require_port_free "${otlp_http_port}"

  compose config >"${tmp_dir}/compose-config.yaml"
  compose up -d --build --quiet-pull
  wait_for_health "http://127.0.0.1:${http_port}/healthz"
  wait_for_events

  diagnostics_dir="${DOGTAP_ARTIFACT_DIR:-${tmp_dir}/diagnostics}"
  mkdir -p "${diagnostics_dir}"
  (
    cd "${repo_root}"
    go run ./cmd/dogtap diagnose \
      -base-url "http://127.0.0.1:${http_port}" \
      -output "${diagnostics_dir}" \
      -expect-non-empty \
      -expect-source otlp \
      -expect-payload-kind log \
      -expect-service filelog-bridge-backend \
      -expect-route /bridge/jobs \
      -expect-trace 4bf92f3577b34da6a3ce929d0e0e4736
  )
  compose logs dogtap otel-collector log-writer >"${diagnostics_dir}/compose.log" 2>&1 || true

  echo "Dogtap filelog bridge smoke passed."
  echo "Dogtap retained OTLP log events from a collector-tailed file."
  echo "Diagnostics: ${diagnostics_dir}"
}

main "$@"
