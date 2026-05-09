# ADR 0007: Datadog-Preserving External Injection Adoption

## Status

Accepted

## Context

Dogtap should be useful to teams that already use Datadog without asking them to
rewrite instrumentation around a Dogtap SDK. The desired adoption path is:

1. Keep existing Datadog or OpenTelemetry SDKs.
2. Add Dogtap as a local sidecar, CI service, or bounded staging tap.
3. Override standard telemetry endpoints from the outside.
4. Remove Dogtap by restoring the original endpoints.

The stable external injection surfaces differ by signal:

- Datadog Browser RUM supports the `proxy` initialization parameter and routes
  RUM and Session Replay uploads through `ddforward`.
- Datadog tracers can target an agent-compatible trace receiver with
  `DD_TRACE_AGENT_URL` or `DD_AGENT_HOST` plus `DD_TRACE_AGENT_PORT`.
- OpenTelemetry SDKs and Collectors can send traces, logs, and metrics to an
  agent-style Collector endpoint using OTLP.
- Datadog Agent log collection and DogStatsD are agent behaviors, not just
  HTTP intake endpoints. Dogtap should not claim full Datadog Agent
  replacement until those surfaces have fixtures and safety gates.

## Decision

Dogtap will define a Datadog-preserving external injection adoption profile.
This profile is a deployment and configuration layer, not a Dogtap application
SDK.

The first supported profile is `local-direct`:

- Run Dogtap next to the application through Docker Compose, a Kubernetes
  sidecar, or a CI service container.
- Point existing Datadog tracers at Dogtap's APM-compatible port.
- Point existing Browser RUM proxy configuration at Dogtap.
- Send logs through Datadog logs HTTP intake or OTLP logs.
- Send metrics through OTLP metrics.
- Keep production Datadog Agent or OpenTelemetry Collector paths as the
  high-fidelity production lane.

Dogtap will also document a `collector-bridge` profile for teams whose current
Datadog usage depends on Agent-side log tailing or Collector pipelines. In that
profile, the existing collector or log forwarder is responsible for turning
stdout, file logs, or runtime metrics into HTTP/OTLP signals that Dogtap can
inspect.

## Compatibility Contract

| Existing practice | Dogtap external injection path | Current status |
| --- | --- | --- |
| Browser RUM SDK | Set or override the SDK `proxy` URL | Supported when the app exposes RUM runtime config |
| Browser Session Replay | Same RUM proxy path, with `ddforward=/api/v2/replay` | Partial replay timeline support |
| Datadog backend tracer | Set `DD_TRACE_AGENT_URL=http://dogtap:8126` or equivalent host/port | Supported for intake and span inspection |
| Datadog trace/log correlation | Keep `DD_LOGS_INJECTION=true`; send structured logs separately | Supported when logs reach Dogtap |
| Datadog logs HTTP sender | Send to Dogtap `/api/v2/logs` | Supported |
| DD Agent stdout/file log tailing | Use a collector/log-forwarder bridge into Dogtap | Not built into Dogtap |
| OTLP traces/logs/metrics | Set OTLP endpoint to Dogtap | Supported |
| DogStatsD metrics | Use OTLP metrics instead for Dogtap inspection | Not supported |
| Datadog Agent integrations | Keep Datadog Agent or Collector in the production lane | Not supported as Dogtap-native integrations |

## Consequences

- Dogtap adoption remains reversible and mostly configuration-only for apps
  that already externalize Datadog and OTLP endpoints.
- Some frontend apps need one small preparatory change: make the Datadog RUM
  `proxy` value runtime-configurable.
- Apps that rely on Datadog Agent log tailing need a bridge before Dogtap can
  inspect the same logs.
- DogStatsD and Agent integrations stay explicit release gaps instead of hidden
  compatibility promises.
- Future implementation PRs should prioritize injection templates, smoke tests,
  and collector bridge examples before claiming broader Datadog Agent parity.
