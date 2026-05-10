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
require_file "examples/deployment/eks-dev/kustomization.yaml"
require_file "examples/deployment/eks-dev/namespace.yaml"
require_file "examples/deployment/eks-dev/serviceaccount.yaml"
require_file "examples/deployment/eks-dev/pvc.yaml"
require_file "examples/deployment/eks-dev/deployment.yaml"
require_file "examples/deployment/eks-dev/service.yaml"
require_file "examples/deployment/eks-dev/networkpolicy.yaml"
require_file "docs/runbooks/EKS_DEV_CLUSTER.md"

# Ruby gives us YAML and JSON parsing from the standard library on GitHub's
# hosted runners without adding a yq dependency to this repository.
ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }' \
  "${root}/examples/deployment/helm-values-sidecar.yaml" \
  "${root}/examples/deployment/helm-values-companion.yaml" \
  "${root}/examples/deployment/eks-dev/kustomization.yaml" \
  "${root}/examples/deployment/eks-dev/namespace.yaml" \
  "${root}/examples/deployment/eks-dev/serviceaccount.yaml" \
  "${root}/examples/deployment/eks-dev/pvc.yaml" \
  "${root}/examples/deployment/eks-dev/deployment.yaml" \
  "${root}/examples/deployment/eks-dev/service.yaml" \
  "${root}/examples/deployment/eks-dev/networkpolicy.yaml"

ruby -rjson -e 'JSON.parse(File.read(ARGV.fetch(0)))' \
  "${root}/examples/deployment/ecs-task-definition.json"

for path in \
  "examples/deployment/README.md" \
  "examples/deployment/helm-values-sidecar.yaml" \
  "examples/deployment/helm-values-companion.yaml" \
  "examples/deployment/ecs-task-definition.json" \
  "examples/deployment/eks-dev/deployment.yaml" \
  "docs/runbooks/EKS_DEV_CLUSTER.md"; do
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
require_text "examples/deployment/eks-dev/deployment.yaml" "readOnlyRootFilesystem: true"
require_text "examples/deployment/eks-dev/deployment.yaml" "runAsNonRoot: true"
require_text "examples/deployment/eks-dev/deployment.yaml" "DOGTAP_STORAGE_KIND"
require_text "examples/deployment/eks-dev/service.yaml" "type: ClusterIP"
require_text "examples/deployment/eks-dev/networkpolicy.yaml" "kind: NetworkPolicy"
require_text "docs/runbooks/EKS_DEV_CLUSTER.md" "kubectl apply -k examples/deployment/eks-dev"

echo "Deployment examples passed syntax and safety-marker checks."
