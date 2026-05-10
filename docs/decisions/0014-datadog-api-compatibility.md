# Decision 0014: Datadog API Compatibility Layer

Date: 2026-05-11

## Status

Accepted

## Context

Dogtap already has its own diagnostics API, workflow contracts, and dashboard
APIs. Those are useful, but many existing tools, prompts, and agent workflows
already know Datadog API paths. A Dogtap-specific query API would make agents
learn a second surface. A full Datadog API clone would violate the product
constitution and create broad compatibility and security risk.

## Decision

Dogtap will provide a read-only Datadog API compatibility subset for retained
telemetry search:

- `POST /api/v2/logs/events/search`
- `POST /api/v2/rum/events/search`
- `POST /api/v2/spans/events/search`
- `GET /api/v1/query`

The compatibility layer maps Datadog-shaped requests and responses onto
Dogtap's bounded event store. It supports the query fields needed for local and
CI debugging: service, env, version, trace ID, span ID, session ID, user ID,
workspace/account/case IDs, route/resource, source/type/status, and simple
free-text matching.

## Non-Goals

- Full Datadog query language parity
- Cursor pagination, indexes, storage tiers, facets, formulas, rollups, or
  permissions
- Quoted phrase matching and advanced boolean query parsing
- Mutating Datadog APIs such as monitors, dashboards, users, service
  definitions, incidents, notebooks, or API key management
- Long-term telemetry retention
- API key validation against Datadog

## Consequences

Agents and existing Datadog-oriented snippets can point at Dogtap for local
debugging with fewer prompt and tooling changes.

The layer must keep returning explicit warnings in response metadata so users do
not mistake it for full Datadog API parity. Future endpoint additions should be
fixture-backed and documented in `docs/DATADOG_API_COMPATIBILITY.md`.
