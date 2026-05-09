#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

ROOT="$(fixture_repo_root)"
ARTIFACT_ROOT="$(fixture_artifact_dir "${ROOT}")"
OUT_DIR="${ARTIFACT_ROOT}/logs"
BASE_URL="$(fixture_base_url)"

mkdir -p "${OUT_DIR}"
fixture_start_dogtap "${ROOT}" "${ARTIFACT_ROOT}"
trap fixture_stop_dogtap EXIT

curl -fsS \
  -H 'content-type: application/json' \
  -H 'dd-api-key: fixture-api-key' \
  -H 'authorization: Bearer fixture-token' \
  --data-binary @"${ROOT}/fixtures/logs/json-log.json" \
  "${BASE_URL}/api/v2/logs?access_token=fixture-query-token" \
  -o "${OUT_DIR}/response-json.json"

curl -fsS \
  -H 'content-type: text/plain' \
  -H 'authorization: Bearer fixture-token' \
  --data-binary @"${ROOT}/fixtures/logs/text-log.txt" \
  "${BASE_URL}/v1/input" \
  -o "${OUT_DIR}/response-text.json"

gzip -c "${ROOT}/fixtures/logs/gzip-log.json" > "${OUT_DIR}/log-gzip.json.gz"
curl -fsS \
  -H 'content-type: application/json' \
  -H 'content-encoding: gzip' \
  -H 'authorization: Bearer fixture-token' \
  --data-binary @"${OUT_DIR}/log-gzip.json.gz" \
  "${BASE_URL}/api/v2/logs" \
  -o "${OUT_DIR}/response-gzip.json"

curl -fsS "${BASE_URL}/api/events?source=logs&limit=20" -o "${OUT_DIR}/events.json"

cat > "${OUT_DIR}/manifest.json" <<EOF
{
  "schemaVersion": 1,
  "capture": "logs",
  "status": "captured-local-dogtap",
  "dogtapBaseUrl": "${BASE_URL}",
  "artifacts": [
    "response-json.json",
    "response-text.json",
    "response-gzip.json",
    "log-gzip.json.gz",
    "events.json"
  ],
  "notes": "JSON, text, and gzip log payloads were posted to local Dogtap only."
}
EOF

printf 'Logs fixture evidence written to %s\n' "${OUT_DIR}"

