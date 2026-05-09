#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

http_port="${DOGTAP_SMOKE_HTTP_PORT:-18080}"
apm_port="${DOGTAP_SMOKE_APM_PORT:-18126}"
otlp_http_port="${DOGTAP_SMOKE_OTLP_HTTP_PORT:-14318}"
grpc_port="${DOGTAP_SMOKE_GRPC_PORT:-14317}"
base_url="http://127.0.0.1:${http_port}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/dogtap-generic-smoke.XXXXXX")"
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
    echo "Port ${port} is already in use. Override DOGTAP_SMOKE_*_PORT or stop the listener." >&2
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

post_fixture() {
  local url="$1"
  local path="$2"
  curl -fsS -X POST "${url}" \
    -H "Content-Type: application/json" \
    --data-binary @"${repo_root}/${path}" >/dev/null
}

post_metric() {
  curl -fsS -X POST "http://127.0.0.1:${otlp_http_port}/v1/metrics" \
    -H "Content-Type: application/json" \
    --data-binary @- >/dev/null <<'JSON'
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          { "key": "service.name", "value": { "stringValue": "quickstart-backend" } },
          { "key": "deployment.environment", "value": { "stringValue": "local" } },
          { "key": "service.version", "value": { "stringValue": "local" } }
        ]
      },
      "scopeMetrics": [
        {
          "metrics": [
            {
              "name": "http.server.request.duration",
              "unit": "ms",
              "gauge": {
                "dataPoints": [
                  {
                    "asDouble": 42.5,
                    "attributes": [
                      { "key": "http.route", "value": { "stringValue": "GET /health" } }
                    ]
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
JSON
}

assert_events() {
  local events
  events="$(curl -fsS "${base_url}/api/events?limit=50")"

  for expected in '"source":"rum"' '"source":"logs"' '"source":"apm"' '"source":"otlp"' '"payloadKind":"metric"'; do
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
  require_port_free "${otlp_http_port}"
  require_port_free "${grpc_port}"

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

  post_fixture "${base_url}/datadog-intake-proxy?ddforward=/api/v2/rum" "fixtures/rum/login.json"
  post_fixture "${base_url}/api/v2/logs" "fixtures/logs/json-log.json"
  post_fixture "http://127.0.0.1:${apm_port}/v0.5/traces" "fixtures/apm/trace.json"
  post_fixture "http://127.0.0.1:${otlp_http_port}/v1/traces" "fixtures/otlp/traces.json"
  post_metric

  assert_events

  diagnostics_dir="${DOGTAP_ARTIFACT_DIR:-${tmp_dir}/diagnostics}"
  (
    cd "${repo_root}"
    go run ./cmd/dogtap diagnose \
      -base-url "${base_url}" \
      -output "${diagnostics_dir}" \
      -expect-non-empty \
      -expect-source rum,logs,apm,otlp \
      -expect-payload-kind metric \
      -expect-service web-frontend,api-service,quickstart-backend \
      -expect-trace 123456789 \
      -expect-metric http.server.request.duration
  )
  cp "${log_file}" "${diagnostics_dir}/dogtap.log"

  echo "Dogtap generic adoption smoke passed."
  echo "Dashboard: ${base_url}"
  echo "Diagnostics: ${diagnostics_dir}"
}

main "$@"
