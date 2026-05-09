# Decision 0001: Build Dogtap as an Intake Inspector, Not a Datadog Clone

## Status

Accepted

## Context

The project idea is close to several existing categories:

- Datadog mock server
- Datadog-compatible backend
- OpenTelemetry collector
- Observability dashboard
- Production telemetry proxy

Trying to cover all of these would create a large and unfocused system.

## Decision

Dogtap will be an intake inspector and validation tool.

It will focus on:

- local mock behavior
- payload inspection
- telemetry contract validation
- redaction and safety checks
- optional forwarding and teeing

It will not aim to replace Datadog.

## Consequences

Positive:

- Smaller MVP
- Clearer value for teams already paying for Datadog
- Lower maintenance burden
- Better fit for CI and debugging

Negative:

- Users looking for a self-hosted Datadog alternative will need another tool.
- Some Datadog UI semantics will remain out of scope.

