# Dogtap Documentation

This directory contains the public product and engineering documentation for
Dogtap.

## Start Here

- [Concept](CONCEPT.md): product positioning and core loop
- [Architecture](ARCHITECTURE.md): runtime shape and module boundaries
- [Adopting Dogtap](runbooks/ADOPTING_DOGTAP.md): generic frontend/backend setup
- [Production Safety](PRODUCTION_SAFETY.md): safety model for forwarding and tee modes
- [Roadmap](ROADMAP.md): current phase and release blockers

## Product And Planning

- [PRD](PRD.md)
- [Final Goal](FINAL_GOAL.md)
- [Testing Strategy](TESTING.md)
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
- [Local Development](runbooks/local-dev.md)
- [Production Deployment](runbooks/PRODUCTION_DEPLOYMENT.md)

## Evidence And Gates

Gate evidence lives under `docs/gates/`.

Fixture evidence lives under `docs/fixtures/`, with promoted fixture payloads
under `fixtures/`.

## Decisions

Architecture and product decisions live under `docs/decisions/`.

Current notable decisions:

- Go backend plus embedded React dashboard
- bounded memory/file storage
- practical Datadog compatibility instead of full private endpoint parity
- generic adoption kit instead of a Dogtap-specific SDK
