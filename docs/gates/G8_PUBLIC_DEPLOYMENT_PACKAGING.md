# G8 Public Deployment Packaging

Date: 2026-05-11

## Scope

This gate covers the first public deployment packaging slice. It makes Dogtap
easier to trial in common deployment environments without claiming to provide a
complete Helm chart, Terraform module, Datadog Agent replacement, or production
observability backend.

## Evidence

Implemented:

- Deployment examples:
  `examples/deployment/`
- Helm sidecar values fragment:
  `examples/deployment/helm-values-sidecar.yaml`
- Helm companion-service values model:
  `examples/deployment/helm-values-companion.yaml`
- ECS/Fargate task definition example:
  `examples/deployment/ecs-task-definition.json`
- EKS dev-cluster Kustomize overlay:
  `examples/deployment/eks-dev/`
- Deployment syntax and safety-marker check:
  `scripts/deployment/check.sh`
- Maintainer command:
  `make deployment-check`
- Decision record:
  `docs/decisions/0013-public-deployment-packaging.md`

Safety markers checked across the examples:

- bounded retained event count
- bounded retained event TTL
- explicit sampling rate
- raw payload storage disabled
- forwarding disabled by default
- private-network guidance
- ECS Dogtap sidecar is non-essential
- EKS overlay uses private `ClusterIP`, NetworkPolicy, non-root security
  context, read-only root filesystem, and SQLite PVC retention

Verification:

```bash
make deployment-check
make shell-check
```

## Gate Status

Passed for the public deployment packaging subset.
