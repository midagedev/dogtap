#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

require_contains() {
  local file="$1"
  local expected="$2"
  if ! grep -Fq -- "${expected}" "${repo_root}/${file}"; then
    echo "Expected ${file} to contain: ${expected}" >&2
    return 1
  fi
}

require_contains "specs/000-product/spec.md" "Release-candidate baseline."
require_contains "specs/000-product/plan.md" "Active implementation baseline."

require_contains "specs/000-product/data-model.md" "- \`faro\`"
require_contains "specs/000-product/data-model.md" "MetricEntry"
require_contains "specs/000-product/data-model.md" "Diagnostics Snapshot"
require_contains "specs/000-product/data-model.md" "Workflow Contract"

require_contains "specs/000-product/quickstart.md" "-workflow-contract configs/contracts/login.yaml"
require_contains "specs/000-product/contracts/intake-api.md" "workflowContracts"
require_contains "docs/ROADMAP.md" "Next Implementation Roadmap"
require_contains "docs/WORKFLOW_CONTRACTS.md" "examples/github-actions/workflow-contract.yml"
