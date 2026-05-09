# G8 External Injection Adoption Subset

Date: 2026-05-09

## Status

Partial.

Dogtap now has a documented Datadog-preserving external injection strategy and
copyable first-pass templates. This subset is not complete until executable
fixtures prove that a normal app can enable and remove Dogtap by changing only
external overlays.

## Goal

Enable a team that already uses Datadog to run Dogtap in local development or
CI without adopting a Dogtap SDK or rewriting instrumentation.

The target workflow is:

1. Add Dogtap as a sidecar, Compose service, Kubernetes patch, or CI service.
2. Override standard Datadog/OTLP endpoints.
3. Verify RUM, Session Replay, logs, traces, and metrics in Dogtap.
4. Remove Dogtap by deleting the override or restoring original endpoint values.

## Current Evidence

- External injection ADR:
  `docs/decisions/0007-external-injection-adoption.md`
- External injection runbook:
  `docs/runbooks/EXTERNAL_INJECTION_ADOPTION.md`
- Compose override template:
  `examples/adoption-kit/compose.override.template.yaml`
- Kubernetes sidecar template:
  `examples/adoption-kit/kubernetes/deployment-sidecar.template.yaml`
- Datadog-preserving env overlay:
  `examples/adoption-kit/datadog-preserve.env`
- Frontend runtime config guidance:
  `examples/adoption-kit/frontend-runtime-config.md`
- Log forwarder bridge guidance:
  `examples/adoption-kit/log-forwarder-overrides.md`

## Source-Backed Compatibility Notes

- Datadog Browser RUM supports a `proxy` init option and routes the intake path
  through `ddforward`.
- Datadog Java tracer supports `DD_TRACE_AGENT_URL`; it takes precedence over
  `DD_AGENT_HOST` and `DD_TRACE_AGENT_PORT`.
- Datadog Docker and Kubernetes log collection are Agent behaviors based on log
  collection settings, mounts, DaemonSets, Helm/Operator config, and
  Autodiscovery.
- DogStatsD is an Agent-side UDP metrics path and is not currently supported by
  Dogtap.
- OpenTelemetry Collector supports an agent pattern where SDKs send OTLP to a
  Collector running alongside the app or on the same host.

Reference links are collected in `docs/references/datadog.md`.

## Remaining Tasks

- Add an executable Compose adoption fixture with a placeholder app service,
  Dogtap override, and rollback check.
- Add an OTel Collector tee example that keeps Datadog primary and sends a
  sampled local copy to Dogtap.
- Add a RUM proxy canary guide covering Browser SDK version, raw-body
  preservation, sensitive header stripping, origin allowlisting, and rollback.
- Capture one realistic sanitized adoption profile and publish only safe
  summaries, commands, and screenshots.

## Gate Decision

G8 remains blocked. This subset improves the adoption contract but does not
replace the required realistic sanitized adoption evidence.
