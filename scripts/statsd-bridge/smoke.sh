#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"

compose_file="${repo_root}/examples/adoption-kit/compose.otel-statsd-bridge.yaml"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/dogtap-statsd-bridge.XXXXXX")"
project_name="dogtap-statsd-bridge-${RANDOM}"

http_port="${DOGTAP_STATSD_BRIDGE_HTTP_PORT:-19083}"
otlp_http_port="${DOGTAP_STATSD_BRIDGE_OTLP_HTTP_PORT:-19320}"
statsd_port="${DOGTAP_STATSD_BRIDGE_STATSD_PORT:-18125}"

cleanup() {
  compose down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

compose() {
  env \
    DOGTAP_REPO="${repo_root}" \
    DOGTAP_STATSD_BRIDGE_HTTP_PORT="${http_port}" \
    DOGTAP_STATSD_BRIDGE_OTLP_HTTP_PORT="${otlp_http_port}" \
    DOGTAP_STATSD_BRIDGE_STATSD_PORT="${statsd_port}" \
    docker compose -p "${project_name}" -f "${compose_file}" "$@"
}

require_tcp_port_free() {
  local port="$1"
  if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port ${port} is already in use. Override DOGTAP_STATSD_BRIDGE_*_PORT or stop the listener." >&2
    exit 2
  fi
}

require_udp_port_free() {
  local port="$1"
  if lsof -iUDP:"${port}" >/dev/null 2>&1; then
    echo "UDP port ${port} is already in use. Override DOGTAP_STATSD_BRIDGE_STATSD_PORT or stop the listener." >&2
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
  compose logs dogtap otel-collector metrics-sender >&2 || true
  exit 3
}

events_have_expected_metrics() {
  local path="$1"
  for expected in \
    '"source":"otlp"' \
    '"payloadKind":"metric"' \
    '"service":"statsd-bridge-backend"' \
    '"env":"local"' \
    'dogtap.bridge.request.count' \
    'dogtap.bridge.queue.depth' \
    '/bridge/metrics'; do
    if ! grep -q "${expected}" "${path}"; then
      return 1
    fi
  done
}

wait_for_events() {
  local attempt
  local events_path="${tmp_dir}/events.json"
  for attempt in $(seq 1 80); do
    if curl -fsS "http://127.0.0.1:${http_port}/api/events?limit=100" -o "${events_path}" 2>/dev/null &&
      events_have_expected_metrics "${events_path}"; then
      return 0
    fi
    sleep 0.5
  done
  echo "Timed out waiting for StatsD bridge metrics." >&2
  sed -n '1,260p' "${events_path}" >&2 || true
  compose logs dogtap otel-collector metrics-sender >&2 || true
  exit 4
}

main() {
  require_tcp_port_free "${http_port}"
  require_tcp_port_free "${otlp_http_port}"
  require_udp_port_free "${statsd_port}"

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
      -expect-payload-kind metric \
      -expect-service statsd-bridge-backend \
      -expect-route /bridge/metrics \
      -expect-metric dogtap.bridge.request.count \
      -expect-endpoint /v1/metrics
  )
  compose logs dogtap otel-collector metrics-sender >"${diagnostics_dir}/compose.log" 2>&1 || true

  echo "Dogtap StatsD bridge smoke passed."
  echo "Dogtap retained OTLP metric events from Collector-received StatsD metrics."
  echo "Diagnostics: ${diagnostics_dir}"
}

main "$@"
