#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

ROOT="$(fixture_repo_root)"
ARTIFACT_ROOT="$(fixture_artifact_dir "${ROOT}")"
OUT_DIR="${ARTIFACT_ROOT}/otlp"
APP_DIR="${ROOT}/testdata/otlp-node"
BASE_URL="$(fixture_base_url)"
OTLP_HTTP_URL="$(fixture_otlp_http_url)"
OTLP_GRPC_ENDPOINT="$(fixture_otlp_grpc_endpoint)"

mkdir -p "${OUT_DIR}"

if [ ! -d "${APP_DIR}/node_modules/@opentelemetry" ]; then
  fixture_write_not_run "${OUT_DIR}/README-not-run.txt" \
    "Missing local OpenTelemetry Node SDK dependencies." \
    "" \
    "Run:" \
    "  npm --prefix testdata/otlp-node install" \
    "  scripts/fixtures/capture-otlp-node-sdk.sh" \
    "" \
    "Expected artifact:" \
    "  testdata/g1-evidence/latest/otlp/events.json"
  printf 'OTLP node SDK capture dependencies are missing; wrote %s\n' "${OUT_DIR}/README-not-run.txt"
  exit 0
fi

fixture_start_dogtap "${ROOT}" "${ARTIFACT_ROOT}"
trap fixture_stop_dogtap EXIT

(
  cd "${APP_DIR}"
  OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="${OTLP_HTTP_URL}/v1/traces" npm run emit:http
) > "${OUT_DIR}/otlp-http.log" 2>&1

(
  cd "${APP_DIR}"
  OTEL_EXPORTER_OTLP_GRPC_ENDPOINT="http://${OTLP_GRPC_ENDPOINT}" npm run emit:grpc
) > "${OUT_DIR}/otlp-grpc.log" 2>&1

sleep 1
curl -fsS "${BASE_URL}/api/events?source=otlp&limit=20" -o "${OUT_DIR}/events.json"

cat > "${OUT_DIR}/manifest.json" <<EOF
{
  "schemaVersion": 1,
  "capture": "otlp-node-sdk",
  "status": "captured-local-dogtap",
  "dogtapOtlpHttpUrl": "${OTLP_HTTP_URL}",
  "dogtapOtlpGrpcEndpoint": "${OTLP_GRPC_ENDPOINT}",
  "artifacts": [
    "otlp-http.log",
    "otlp-grpc.log",
    "events.json"
  ],
  "notes": "OpenTelemetry Node SDK exported OTLP HTTP and gRPC traces to local Dogtap only."
}
EOF

printf 'OTLP fixture evidence written to %s\n' "${OUT_DIR}"

