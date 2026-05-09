#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"

compose_dir="${repo_root}/examples/external-injection-smoke"
base_compose="${compose_dir}/compose.yaml"
override_compose="${compose_dir}/compose.override.dogtap.yaml"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/dogtap-external-injection.XXXXXX")"
project_name="dogtap-external-injection-${RANDOM}"

http_port="${DOGTAP_EXTERNAL_HTTP_PORT:-19080}"
apm_port="${DOGTAP_EXTERNAL_APM_PORT:-19126}"
otlp_grpc_port="${DOGTAP_EXTERNAL_OTLP_GRPC_PORT:-19317}"
otlp_http_port="${DOGTAP_EXTERNAL_OTLP_HTTP_PORT:-19318}"

cleanup() {
  docker compose -p "${project_name}" \
    -f "${base_compose}" \
    -f "${override_compose}" \
    down -v --remove-orphans >/dev/null 2>&1 || true
  docker compose -p "${project_name}" \
    -f "${base_compose}" \
    down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

require_port_free() {
  local port="$1"
  if lsof -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port ${port} is already in use. Override DOGTAP_EXTERNAL_*_PORT or stop the listener." >&2
    exit 2
  fi
}

compose_base() {
  docker compose -p "${project_name}" -f "${base_compose}" "$@"
}

compose_injected() {
  DOGTAP_EXTERNAL_HTTP_PORT="${http_port}" \
    DOGTAP_EXTERNAL_APM_PORT="${apm_port}" \
    DOGTAP_EXTERNAL_OTLP_GRPC_PORT="${otlp_grpc_port}" \
    DOGTAP_EXTERNAL_OTLP_HTTP_PORT="${otlp_http_port}" \
    docker compose -p "${project_name}" \
      -f "${base_compose}" \
      -f "${override_compose}" \
      "$@"
}

assert_contains() {
  local path="$1"
  local marker="$2"
  if ! grep -q "${marker}" "${path}"; then
    echo "Expected ${path} to contain ${marker}" >&2
    sed -n '1,260p' "${path}" >&2
    exit 1
  fi
}

assert_not_contains() {
  local path="$1"
  local marker="$2"
  if grep -q "${marker}" "${path}"; then
    echo "Expected ${path} not to contain ${marker}" >&2
    sed -n '1,260p' "${path}" >&2
    exit 1
  fi
}

require_service() {
  local path="$1"
  local service="$2"
  if ! grep -qx "${service}" "${path}"; then
    echo "Expected service ${service} in ${path}" >&2
    cat "${path}" >&2
    exit 1
  fi
}

wait_for_host_health() {
  local url="$1"
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "Timed out waiting for ${url}" >&2
  compose_injected logs dogtap >&2 || true
  exit 3
}

wait_for_frontend() {
  local attempt
  for attempt in $(seq 1 60); do
    if compose_base exec -T frontend node -e "
      fetch('http://127.0.0.1:3000/healthz')
        .then(r => process.exit(r.ok ? 0 : 1))
        .catch(() => process.exit(1));
    " >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "Timed out waiting for frontend health." >&2
  compose_base logs frontend backend >&2 || true
  exit 3
}

exercise_frontend() {
  compose_base exec -T frontend node -e "
    const response = await fetch('http://127.0.0.1:3000/exercise');
    const text = await response.text();
    console.log(text);
    if (!response.ok) process.exit(1);
  "
}

assert_events() {
  local events="$1"
  for expected in \
    '"source":"rum"' \
    '"source":"logs"' \
    '"source":"apm"' \
    '"source":"otlp"' \
    '"payloadKind":"replay"' \
    '"payloadKind":"metric"' \
    '"segmentEncoding":"zlib"' \
    '"status":"fail"' \
    'required.rum.userId' \
    'External Injection Workflow' \
    'external_injection.workflow.duration' \
    '"service":"external-smoke-frontend"' \
    '"service":"external-smoke-backend"'; do
    if ! printf '%s' "${events}" | grep -q "${expected}"; then
      echo "Missing expected event marker: ${expected}" >&2
      printf '%s\n' "${events}" >&2
      exit 1
    fi
  done
}

main() {
  require_port_free "${http_port}"
  require_port_free "${apm_port}"
  require_port_free "${otlp_grpc_port}"
  require_port_free "${otlp_http_port}"

  compose_base config --services >"${tmp_dir}/base-services.txt"
  require_service "${tmp_dir}/base-services.txt" "backend"
  require_service "${tmp_dir}/base-services.txt" "frontend"
  if grep -qx "dogtap" "${tmp_dir}/base-services.txt"; then
    echo "Base compose unexpectedly contains dogtap." >&2
    cat "${tmp_dir}/base-services.txt" >&2
    exit 1
  fi

  compose_base config >"${tmp_dir}/base-config.yaml"
  assert_not_contains "${tmp_dir}/base-config.yaml" "DD_TRACE_AGENT_URL"
  assert_not_contains "${tmp_dir}/base-config.yaml" "DATADOG_RUM_PROXY_URL"
  assert_not_contains "${tmp_dir}/base-config.yaml" "OTEL_EXPORTER_OTLP_ENDPOINT"

  compose_base up -d --quiet-pull
  wait_for_frontend
  base_response="$(exercise_frontend)"
  if ! printf '%s' "${base_response}" | grep -q '"frontendTelemetryEnabled":false'; then
    echo "Base frontend should run without Dogtap telemetry." >&2
    printf '%s\n' "${base_response}" >&2
    exit 1
  fi
  if ! printf '%s' "${base_response}" | grep -q '"telemetryEnabled":false'; then
    echo "Base backend should run without Dogtap telemetry." >&2
    printf '%s\n' "${base_response}" >&2
    exit 1
  fi
  compose_base down -v --remove-orphans >/dev/null

  compose_injected config --services >"${tmp_dir}/injected-services.txt"
  require_service "${tmp_dir}/injected-services.txt" "backend"
  require_service "${tmp_dir}/injected-services.txt" "frontend"
  require_service "${tmp_dir}/injected-services.txt" "dogtap"

  compose_injected config >"${tmp_dir}/injected-config.yaml"
  assert_contains "${tmp_dir}/injected-config.yaml" "DD_TRACE_AGENT_URL: http://dogtap:8126"
  assert_contains "${tmp_dir}/injected-config.yaml" "DATADOG_RUM_PROXY_URL: http://dogtap:8080/datadog-intake-proxy"
  assert_contains "${tmp_dir}/injected-config.yaml" "OTEL_EXPORTER_OTLP_ENDPOINT: http://dogtap:4318"
  assert_contains "${tmp_dir}/injected-config.yaml" "condition: service_healthy"

  compose_injected up -d --build --quiet-pull
  wait_for_host_health "http://127.0.0.1:${http_port}/healthz"
  wait_for_frontend

  injected_response="$(exercise_frontend)"
  if ! printf '%s' "${injected_response}" | grep -q '"frontendTelemetryEnabled":true'; then
    echo "Injected frontend should send RUM telemetry." >&2
    printf '%s\n' "${injected_response}" >&2
    exit 1
  fi
  if ! printf '%s' "${injected_response}" | grep -q '"telemetryEnabled":true'; then
    echo "Injected backend should send telemetry." >&2
    printf '%s\n' "${injected_response}" >&2
    exit 1
  fi

  events="$(curl -fsS "http://127.0.0.1:${http_port}/api/events?limit=100")"
  assert_events "${events}"

  diagnostics_dir="${DOGTAP_ARTIFACT_DIR:-${tmp_dir}/diagnostics}"
  (
    cd "${repo_root}"
    go run ./cmd/dogtap diagnose \
      -base-url "http://127.0.0.1:${http_port}" \
      -output "${diagnostics_dir}" \
      -expect-non-empty \
      -expect-source rum,logs,apm,otlp \
      -expect-payload-kind replay,metric \
      -expect-service external-smoke-frontend,external-smoke-backend \
      -expect-metric external_injection.workflow.duration
  )
  compose_injected logs dogtap frontend backend >"${diagnostics_dir}/compose.log" 2>&1 || true

  compose_injected down -v --remove-orphans >/dev/null

  compose_base config --services >"${tmp_dir}/rollback-services.txt"
  if grep -qx "dogtap" "${tmp_dir}/rollback-services.txt"; then
    echo "Rollback compose unexpectedly contains dogtap." >&2
    cat "${tmp_dir}/rollback-services.txt" >&2
    exit 1
  fi

  echo "Dogtap external injection smoke passed."
  echo "Base stack: frontend and backend run without Dogtap-specific settings."
  echo "Injected stack: override adds Dogtap plus standard Datadog/OTLP endpoints."
  echo "Rollback: omitting the override removes Dogtap and endpoint overrides."
  echo "Diagnostics: ${diagnostics_dir}"
}

main "$@"
