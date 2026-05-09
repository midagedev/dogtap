#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

ROOT="$(fixture_repo_root)"
ARTIFACT_ROOT="$(fixture_artifact_dir "${ROOT}")"
OUT_DIR="${ARTIFACT_ROOT}/rum"
APP_DIR="${ROOT}/testdata/rum-browser"
BASE_URL="$(fixture_base_url)"

mkdir -p "${OUT_DIR}"

if [ ! -d "${APP_DIR}/node_modules/@datadog/browser-rum" ] || [ ! -d "${APP_DIR}/node_modules/playwright" ]; then
  fixture_write_not_run "${OUT_DIR}/README-not-run.txt" \
    "Missing local browser RUM capture dependencies." \
    "" \
    "Run:" \
    "  npm --prefix testdata/rum-browser install" \
    "  scripts/fixtures/capture-rum-browser.sh" \
    "" \
    "Expected artifact:" \
    "  testdata/g1-evidence/latest/rum/events.json"
  printf 'RUM browser capture dependencies are missing; wrote %s\n' "${OUT_DIR}/README-not-run.txt"
  exit 0
fi

fixture_start_dogtap "${ROOT}" "${ARTIFACT_ROOT}"
trap fixture_stop_dogtap EXIT

(
  cd "${APP_DIR}"
  DOGTAP_BASE_URL="${BASE_URL}" RUM_APP_PORT="${RUM_APP_PORT:-18081}" node server.mjs
) > "${OUT_DIR}/rum-app.log" 2>&1 &
RUM_APP_PID="$!"

cleanup_rum_app() {
  kill "${RUM_APP_PID}" >/dev/null 2>&1 || true
  wait "${RUM_APP_PID}" >/dev/null 2>&1 || true
  fixture_stop_dogtap
}
trap cleanup_rum_app EXIT

for _ in $(seq 1 80); do
  if curl -fsS "http://127.0.0.1:${RUM_APP_PORT:-18081}/" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

(
  cd "${ROOT}"
  DOGTAP_FIXTURE_ARTIFACT_DIR="${OUT_DIR}" RUM_APP_URL="http://127.0.0.1:${RUM_APP_PORT:-18081}/" npm --prefix testdata/rum-browser run capture
)

curl -fsS "${BASE_URL}/api/events?source=rum&limit=50" -o "${OUT_DIR}/events.json"

cat > "${OUT_DIR}/manifest.json" <<EOF
{
  "schemaVersion": 1,
  "capture": "rum-browser",
  "status": "captured-local-dogtap",
  "dogtapBaseUrl": "${BASE_URL}",
  "artifacts": [
    "browser-network.json",
    "events.json"
  ],
  "notes": "Browser RUM SDK payload was generated in a local browser app and proxied to Dogtap."
}
EOF

printf 'RUM fixture evidence written to %s\n' "${OUT_DIR}"

