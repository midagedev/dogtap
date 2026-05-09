#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

ROOT="$(fixture_repo_root)"
ARTIFACT_ROOT="$(fixture_artifact_dir "${ROOT}")"
OUT_DIR="${ARTIFACT_ROOT}/apm"
APP_DIR="${ROOT}/testdata/apm-node"
BASE_URL="$(fixture_base_url)"
APM_URL="$(fixture_apm_url)"

mkdir -p "${OUT_DIR}"

if [ ! -d "${APP_DIR}/node_modules/dd-trace" ]; then
  fixture_write_not_run "${OUT_DIR}/README-not-run.txt" \
    "Missing local Datadog Node tracer dependency." \
    "" \
    "Run:" \
    "  npm --prefix testdata/apm-node install" \
    "  scripts/fixtures/capture-apm-node-tracer.sh" \
    "" \
    "Expected artifact:" \
    "  testdata/g1-evidence/latest/apm/events.json"
  printf 'APM node tracer capture dependencies are missing; wrote %s\n' "${OUT_DIR}/README-not-run.txt"
  exit 0
fi

fixture_start_dogtap "${ROOT}" "${ARTIFACT_ROOT}"
trap fixture_stop_dogtap EXIT

(
  cd "${APP_DIR}"
  DD_TRACE_AGENT_URL="${APM_URL}" \
    DD_SERVICE="api-service" \
    DD_ENV="local" \
    DD_VERSION="g1-fixture" \
    npm run emit
) > "${OUT_DIR}/dd-trace.log" 2>&1

sleep 1
curl -fsS "${BASE_URL}/api/events?source=apm&limit=20" -o "${OUT_DIR}/events.json"

cat > "${OUT_DIR}/manifest.json" <<EOF
{
  "schemaVersion": 1,
  "capture": "apm-node-tracer",
  "status": "captured-local-dogtap",
  "dogtapApmUrl": "${APM_URL}",
  "artifacts": [
    "dd-trace.log",
    "events.json"
  ],
  "notes": "Datadog Node tracer emitted to local Dogtap APM intake only."
}
EOF

printf 'APM fixture evidence written to %s\n' "${OUT_DIR}"

