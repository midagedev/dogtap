# ADR 0011: Filelog Bridge For Agent-Tailed Logs

## Status

Accepted

## Context

Many teams see logs in Datadog because the Datadog Agent, Fluent Bit, Vector,
or an OpenTelemetry Collector tails container stdout, Kubernetes log files, or
application log files. Dogtap does not implement that tailing behavior and
should not claim to be a full Datadog Agent replacement without fixture-backed
parity and production safety gates.

The product still needs a practical adoption path for local and CI validation
when existing applications only write structured logs to stdout or files.

## Decision

Dogtap will keep log tailing outside the Dogtap runtime and ship an adoption
recipe that uses the OpenTelemetry Collector Contrib `filelog` receiver.

The first supported recipe:

- tails JSON log files with the Collector `filelog` receiver
- promotes `message` into the OTLP log body
- preserves `service`, `env`, `version`, route, method, status, and trace
  context as OTLP log attributes
- exports to Dogtap as OTLP HTTP JSON
- is verified by `make smoke-log-bridge`

The recipe uses OTLP HTTP JSON because Dogtap can inspect that payload shape in
detail today. OTLP gRPC is also decoded by Dogtap. OTLP HTTP protobuf remains
accepted but is currently retained as byte metadata rather than rich log
details.

## Consequences

- Dogtap adoption stays reversible: removing the bridge restores the original
  log path.
- Existing production Datadog Agent or collector behavior can stay unchanged.
- Dogtap remains an inspector and validator, not a log collector daemon.
- Future DogStatsD or Agent-behavior work should follow the same rule: add a
  fixture-backed bridge or explicit support decision before claiming parity.
