# ADR 0005: APM Fixture Evidence Scope

## Status

Accepted

## Context

The original technical spike named a Java/Spring tracer fixture. During G1,
the available local harness produced real Datadog APM wire traffic with the
official Node tracer (`dd-trace` 5.44.0) against local Dogtap. This evidence
covered the G1 gate requirement of a payload captured from a real Datadog
tracer and exercised msgpack `/v0.4/traces` intake.

## Decision

For the first G1 gate, a real Datadog tracer fixture from Node is sufficient
APM compatibility evidence. Java/Spring tracer capture and `dd-apm-test-agent`
reference comparison remain useful follow-up work, but they do not block the
initial runtime, dashboard, CI, or forwarding slices.

## Consequences

- G1 can pass with the promoted Node tracer evidence under `fixtures/apm/`.
- Java/Spring evidence should be revisited during adoption profiles if
  those services are the first production candidates.
- APM forwarding remains deferred until broader tracer compatibility evidence
  exists.
