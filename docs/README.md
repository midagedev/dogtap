# Dogtap Documentation

This directory contains the public product and engineering documentation for
Dogtap.

## Start Here

- [Concept](CONCEPT.md): product positioning and core loop
- [Architecture](ARCHITECTURE.md): runtime shape and module boundaries
- [Adopting Dogtap](runbooks/ADOPTING_DOGTAP.md): generic frontend/backend setup
- [External Injection Adoption](runbooks/EXTERNAL_INJECTION_ADOPTION.md):
  preserve existing Datadog usage with sidecars and endpoint overrides
- [Workflow Contracts](WORKFLOW_CONTRACTS.md): assert real user paths through
  received RUM, replay, logs, traces, metrics, and correlation evidence
- [Datadog API Compatibility](DATADOG_API_COMPATIBILITY.md): read-only
  Datadog-shaped search/query endpoints for retained Dogtap telemetry
- [SQLite Storage Decision](decisions/0015-sqlite-storage.md): bounded
  persistent storage for local, CI, isolated E2E, and dev-cluster inspection
- [GitHub Actions Workflow Contract Example](../examples/github-actions/):
  run Dogtap beside an app E2E suite and assert telemetry afterward
- [Deployment Examples](../examples/deployment/): Helm and ECS trial shapes
  with retention, sampling, forwarding, and private-network warnings
- [EKS Dev Cluster](runbooks/EKS_DEV_CLUSTER.md): Kustomize overlay and smoke
  path for a private dev-cluster Dogtap service
- [RUM Proxy Canary](runbooks/RUM_PROXY_CANARY.md): safely test Browser RUM and
  Session Replay proxying through Dogtap
- [Support Matrix](SUPPORT_MATRIX.md): supported surfaces and explicit limits
- [Production Safety](PRODUCTION_SAFETY.md): safety model for forwarding and tee modes
- [Roadmap](ROADMAP.md): current phase and release blockers
- [Next Implementation Roadmap](ROADMAP.md#next-implementation-roadmap):
  implementation candidates discovered from spec/docs/code alignment work

## Product And Planning

- [PRD](PRD.md)
- [Final Goal](FINAL_GOAL.md)
- [Testing Strategy](TESTING.md)
- [Workflow Contracts](WORKFLOW_CONTRACTS.md)
- [Support Matrix](SUPPORT_MATRIX.md)
- [Next Implementation Roadmap](ROADMAP.md#next-implementation-roadmap)
- [Agent Orchestration](AGENT_ORCHESTRATION.md)

Canonical spec artifacts live under `specs/000-product/`:

- `spec.md`
- `plan.md`
- `tasks.md`
- `gates.md`
- `quickstart.md`
- `data-model.md`
- `contracts/intake-api.md`

## Runbooks

- [Generic Local Adoption](runbooks/ADOPTING_DOGTAP.md)
- [External Injection Adoption](runbooks/EXTERNAL_INJECTION_ADOPTION.md)
- [RUM Proxy Canary](runbooks/RUM_PROXY_CANARY.md)
- [Local Development](runbooks/local-dev.md)
- [Production Deployment](runbooks/PRODUCTION_DEPLOYMENT.md)
- [EKS Dev Cluster](runbooks/EKS_DEV_CLUSTER.md)
- [Release Candidate](runbooks/RELEASE_CANDIDATE.md)

## Evidence And Gates

Gate evidence lives under `docs/gates/`.

Fixture evidence lives under `docs/fixtures/`, with promoted fixture payloads
under `fixtures/`.

Current release-candidate state:

- [G2 SQLite Storage](gates/G2_SQLITE_STORAGE.md): persistent queryable store
  subset passed
- [G5 Datadog API Compatibility](gates/G5_DATADOG_API_COMPATIBILITY.md):
  read-only compatibility, structured debugging, and quoted query hardening
  slices passed
- [G4 Dashboard Observability UX](gates/G4_DASHBOARD_OBSERVABILITY_UX.md):
  structured log and metric chart subset passed
- [G8 Generic Adoption Smoke](gates/G8_GENERIC_ADOPTION_SMOKE.md): generic
  quickstart subset passed
- [G8 External Injection Adoption](gates/G8_EXTERNAL_INJECTION_ADOPTION.md):
  Datadog-preserving strategy and sanitized adoption profile passed
- [G8 Sanitized Adoption Profile](gates/G8_SANITIZED_ADOPTION_PROFILE.md):
  public-safe frontend/backend adoption evidence
- [G8 Public Deployment Packaging](gates/G8_PUBLIC_DEPLOYMENT_PACKAGING.md):
  Helm and ECS trial examples with checked safety markers
- [G8 EKS Dev Cluster](gates/G8_EKS_DEV_CLUSTER.md): private Kustomize
  overlay and smoke runbook readiness
- [G8 Release Candidate](gates/G8_RELEASE_CANDIDATE.md): first public
  release-candidate evidence passed

## Decisions

Architecture and product decisions live under `docs/decisions/`.

Current notable decisions:

- Go backend plus embedded React dashboard
- bounded memory/file/SQLite storage
- practical Datadog compatibility instead of full private endpoint parity
- generic adoption kit instead of a Dogtap-specific SDK
- Datadog-preserving external injection before broader Agent parity
- workflow observability contracts as an additive diagnostics artifact
- read-only Datadog API compatibility for local retained telemetry search
- opt-in SQLite storage for restart-safe bounded retained telemetry
