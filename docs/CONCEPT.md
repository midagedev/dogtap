# Dogtap Concept

## One-Line Description

Dogtap is a Datadog-compatible telemetry tap and workflow contract validator
for local development, CI, and production-safe forwarding experiments.

## Product Positioning

Dogtap helps teams that already use Datadog answer a practical question:

> Did our application emit the telemetry we need, with the right context, without leaking unsafe data?

Dogtap is intentionally not a Datadog clone. It does not replace Datadog dashboards, monitors, notebooks, or long-term storage. It sits before Datadog or next to Datadog and makes intake behavior visible.

The strongest product promise is workflow observability testing: after a user
path such as login or checkout runs, Dogtap can assert whether RUM, Session
Replay, backend logs, traces, metrics, correlation, and privacy-safe context
arrived in a debuggable shape.

## Why the Name Works

- `Dog` makes the Datadog relationship obvious.
- `tap` suggests inspection, teeing, proxying, and flow visibility.
- The name still works if Dogtap later supports OpenTelemetry or other observability tools.

## Core Product Loop

1. Point an app, SDK, tracer, or collector to Dogtap.
2. Exercise a workflow.
3. Inspect what arrived.
4. Fix missing context, unsafe fields, or correlation gaps.
5. Lock the expected telemetry contract into CI.
6. Optionally forward or tee safely in staging and production.

## Design Philosophy

### Make hidden telemetry visible

Developers should not need Datadog access or production deployment to see what their instrumentation emits.

### Validate semantics, not just delivery

Receiving a payload is not enough. Dogtap should explain whether it is useful for debugging.

### Respect production risk

Telemetry often contains sensitive context. Dogtap should default to redaction, bounded storage, and fail-open forwarding behavior.

### Stay reversible

Applications should use standard Datadog and OpenTelemetry configuration. Removing Dogtap should not require application code changes.
