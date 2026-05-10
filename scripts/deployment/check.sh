#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

require_file() {
  local path="$1"
  if [ ! -f "${root}/${path}" ]; then
    echo "Missing deployment example: ${path}" >&2
    exit 1
  fi
}

require_text() {
  local path="$1"
  local text="$2"
  if ! grep -Fq "${text}" "${root}/${path}"; then
    echo "Expected ${path} to mention: ${text}" >&2
    exit 1
  fi
}

require_file "examples/deployment/README.md"
require_file "examples/deployment/helm-values-sidecar.yaml"
require_file "examples/deployment/helm-values-companion.yaml"
require_file "examples/deployment/ecs-task-definition.json"

# Ruby gives us YAML and JSON parsing from the standard library on GitHub's
# hosted runners without adding a yq dependency to this repository.
ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }' \
  "${root}/examples/deployment/helm-values-sidecar.yaml" \
  "${root}/examples/deployment/helm-values-companion.yaml"

ruby -rjson -e 'JSON.parse(File.read(ARGV.fetch(0)))' \
  "${root}/examples/deployment/ecs-task-definition.json"

for path in \
  "examples/deployment/README.md" \
  "examples/deployment/helm-values-sidecar.yaml" \
  "examples/deployment/helm-values-companion.yaml" \
  "examples/deployment/ecs-task-definition.json"; do
  require_text "${path}" "DOGTAP_STORAGE_MAX_EVENTS"
  require_text "${path}" "DOGTAP_STORAGE_TTL"
  require_text "${path}" "DOGTAP_SAMPLING_RATE"
  require_text "${path}" "DOGTAP_ALLOW_RAW_PAYLOADS"
  require_text "${path}" "DOGTAP_FORWARDING_ENABLED"
done

require_text "examples/deployment/README.md" "private"
require_text "examples/deployment/helm-values-sidecar.yaml" "Private-network warning"
require_text "examples/deployment/helm-values-companion.yaml" "Private-network warning"
require_text "examples/deployment/ecs-task-definition.json" "dogtap.warning.private-network"
require_text "examples/deployment/ecs-task-definition.json" "\"essential\": false"

echo "Deployment examples passed syntax and safety-marker checks."
