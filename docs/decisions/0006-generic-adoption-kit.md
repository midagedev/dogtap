# ADR 0006: Generic Adoption Kit

## Status

Accepted

## Context

Dogtap's first adoption work focused on one project-specific local stack. That is
useful evidence, but it does not solve the broader product problem: a team with
a common frontend plus backend application should be able to apply Dogtap
without learning Dogtap internals or adding a Dogtap-only SDK.

The constitution requires integration to be reversible and the product spec
already says Dogtap must be usable without a Dogtap-specific SDK. The practical
integration surfaces are therefore the existing vendor-neutral or Datadog-native
ones:

- Datadog Browser RUM proxy configuration for frontend telemetry and replay
- OTLP HTTP or gRPC for backend traces, logs, and metrics
- Datadog agent-compatible trace intake for existing Datadog tracers
- Datadog logs HTTP intake for direct log sender or forwarder use

## Decision

Dogtap will ship a generic adoption kit before deeper project-specific
automation. The kit is documentation plus copyable templates, not a new runtime
SDK. It will include:

- a Compose sidecar template for local Dogtap with persistent storage
- frontend RUM proxy snippets
- backend OTLP HTTP and gRPC environment snippets
- backend Datadog tracer environment snippets
- logs HTTP intake examples
- a smoke verification script that exercises the generic intake surfaces
- a compact dashboard target summary for the active local intake endpoints

## Consequences

- Dogtap adoption remains reversible: removing Dogtap means restoring the
  original Datadog or OTLP endpoints.
- The first generic path prefers OTLP for new backend instrumentation because it
  works across languages and avoids Datadog private payload drift.
- Existing Datadog tracers remain supported through the APM-compatible port, but
  broad tracer compatibility still depends on fixture evidence.
- The kit deliberately does not become an OpenTelemetry Collector replacement,
  a language-specific auto-instrumentation manager, or a Dogtap SDK.
