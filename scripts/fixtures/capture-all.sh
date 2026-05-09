#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/capture-logs.sh"
"${SCRIPT_DIR}/capture-rum-browser.sh"
"${SCRIPT_DIR}/capture-apm-node-tracer.sh"
"${SCRIPT_DIR}/capture-apm-dd-test-agent-reference.sh"
"${SCRIPT_DIR}/capture-otlp-node-sdk.sh"

