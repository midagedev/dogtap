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
- [GitHub Actions Workflow Contract Example](../examples/github-actions/):
  run Dogtap beside an app E2E suite and assert telemetry afterward
- [RUM Proxy Canary](runbooks/RUM_PROXY_CANARY.md): safely test Browser RUM and
  Session Replay proxying through Dogtap
- [Support Matrix](SUPPORT_MATRIX.md): supported surfaces and explicit limits
- [Production Safety](PRODUCTION_SAFETY.md): safety model for forwarding and tee modes
- [Roadmap](ROADMAP.md): current phase and release blockers

## Product And Planning

- [PRD](PRD.md)
- [Final Goal](FINAL_GOAL.md)
- [Testing Strategy](TESTING.md)
- [Workflow Contracts](WORKFLOW_CONTRACTS.md)
- [Support Matrix](SUPPORT_MATRIX.md)
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
- [Release Candidate](runbooks/RELEASE_CANDIDATE.md)

## Evidence And Gates

Gate evidence lives under `docs/gates/`.

Fixture evidence lives under `docs/fixtures/`, with promoted fixture payloads
under `fixtures/`.

Current release-candidate state:

- [G8 Generic Adoption Smoke](gates/G8_GENERIC_ADOPTION_SMOKE.md): generic
  quickstart subset passed
- [G8 External Injection Adoption](gates/G8_EXTERNAL_INJECTION_ADOPTION.md):
  Datadog-preserving strategy and sanitized adoption profile passed
- [G8 Sanitized Adoption Profile](gates/G8_SANITIZED_ADOPTION_PROFILE.md):
  public-safe frontend/backend adoption evidence
- [G8 Release Candidate](gates/G8_RELEASE_CANDIDATE.md): first public
  release-candidate evidence passed

## Decisions

Architecture and product decisions live under `docs/decisions/`.

Current notable decisions:

- Go backend plus embedded React dashboard
- bounded memory/file storage
- practical Datadog compatibility instead of full private endpoint parity
- generic adoption kit instead of a Dogtap-specific SDK
- Datadog-preserving external injection before broader Agent parity
- workflow observability contracts as an additive diagnostics artifact
