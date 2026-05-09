# ADR 0008: Faro Compatibility Smoke

## Status

Accepted

## Context

Dogtap's core product remains a Datadog-compatible telemetry intake inspector
with OTLP as the preferred escape hatch for broader collector interoperability.
The external-injection smoke stack now also needs to prove that a real Grafana
Faro Web SDK can send telemetry through Dogtap during local and CI integration
smoke.

Faro has two practical adoption paths:

- Native Faro-compatible intake into Dogtap for narrow integration smoke.
- Grafana Alloy `faro.receiver`, followed by OTLP export into Dogtap, for
  production-grade collection and schema handling.

The native path is useful for proving that Dogtap can receive and inspect SDK
payloads from a browser workflow. It should not imply that Dogtap is a
production Faro collector or an Alloy replacement.

## Decision

Dogtap will include an experimental native Faro intake for compatibility smoke
only:

- `POST /faro`
- `POST /collect`
- `POST /collect/`

The smoke workflow lives in `examples/external-injection-smoke/frontend` at
`/faro` and is verified with `make smoke-faro`. The workflow sends sanitized
Faro SDK event, measurement, and log telemetry with representative user,
account, workspace, case, session, and route context.

For production-grade Faro usage, Dogtap documentation will recommend routing
through Grafana Alloy `faro.receiver` and exporting OTLP to Dogtap. That keeps
Faro protocol compatibility, buffering, and schema drift handling in the
collector designed for that surface while Dogtap continues to inspect normalized
telemetry.

## Consequences

- Dogtap can run a deterministic Faro SDK integration smoke without adding a
  Dogtap-specific browser SDK.
- `/faro`, `/collect`, and `/collect/` are documented as experimental smoke
  endpoints, not a stable production receiver contract.
- Production guidance stays aligned with the constitution: prefer OTLP and
  collector interoperability where compatibility surfaces are broader than the
  first Dogtap implementation.
- Future promotion from smoke to supported native Faro intake requires fixtures,
  retention and redaction evidence, forwarding or tee behavior decisions, and
  explicit gate coverage.
