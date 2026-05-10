# Decision 0013: Public Deployment Packaging

Date: 2026-05-11

## Status

Accepted

## Context

Dogtap already has local Docker Compose adoption examples and bridge smokes.
Public users still need deployment-shaped examples before they can trial Dogtap
with a team service. These examples affect deployment safety because a copied
configuration can expose telemetry, store more data than intended, or make
Dogtap look like a production observability backend.

## Decision

Dogtap will publish deployment examples as copyable starting points, not as a
managed production platform:

- Helm sidecar and companion-service values examples live under
  `examples/deployment/`.
- An ECS/Fargate task definition example shows Dogtap as a non-essential
  internal inspection sidecar.
- An EKS dev-cluster Kustomize overlay shows Dogtap as a private companion
  service with SQLite PVC retention, NetworkPolicy, and rollback steps.
- Each recipe must include explicit retention, sampling, forwarding, raw
  payload, and private-network warnings.
- CI validates example syntax and required safety markers through
  `make deployment-check`.

## Consequences

Users get practical deployment shapes without Dogtap claiming to replace a
Datadog Agent, Datadog intake, OpenTelemetry Collector, Helm chart, or Terraform
module.

The examples are intentionally conservative:

- Dogtap is private by default.
- Raw payload storage is disabled.
- Retention and sampling are explicit.
- Forwarding is disabled unless an owner explicitly enables it with
  deployment-managed secrets.
- In ECS, the Dogtap container is non-essential so the app is not stopped by a
  Dogtap failure.
- In EKS dev clusters, Dogtap is single-replica, private `ClusterIP`, bounded
  by SQLite retention settings, and removable with its PVC when the diagnostic
  window ends.

Future production packaging can add real Helm charts or IaC modules only after
the examples have user feedback and fixture-backed safety checks.
