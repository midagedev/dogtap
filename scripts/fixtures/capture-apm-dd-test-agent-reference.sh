#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

ROOT="$(fixture_repo_root)"
ARTIFACT_ROOT="$(fixture_artifact_dir "${ROOT}")"
OUT_DIR="${ARTIFACT_ROOT}/apm-dd-test-agent"

mkdir -p "${OUT_DIR}"

fixture_write_not_run "${OUT_DIR}/README-not-run.txt" \
  "This reference path is documented but not automated yet." \
  "" \
  "Use dd-apm-test-agent as a reference receiver when comparing Dogtap APM behavior:" \
  "  git clone https://github.com/DataDog/dd-apm-test-agent tmp/dd-apm-test-agent" \
  "  cd tmp/dd-apm-test-agent" \
  "  ./run.sh" \
  "" \
  "Then run a Datadog tracer sample against both dd-apm-test-agent and Dogtap." \
  "Keep traffic local and store sanitized comparison artifacts under:" \
  "  testdata/g1-evidence/latest/apm-dd-test-agent/" \
  "" \
  "Dogtap primary local tracer path:" \
  "  npm --prefix testdata/apm-node install" \
  "  scripts/fixtures/capture-apm-node-tracer.sh"

printf 'APM dd-apm-test-agent reference instructions written to %s\n' "${OUT_DIR}/README-not-run.txt"

