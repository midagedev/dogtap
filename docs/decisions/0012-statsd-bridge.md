# ADR 0012: StatsD Bridge For DogStatsD-Style Metrics

## Status

Accepted

## Context

Dogtap supports OTLP metrics but does not bind UDP `8125` and does not
implement Datadog Agent DogStatsD behavior. Many services still emit
DogStatsD-style counters and gauges to a Datadog Agent in local, staging, or
production environments.

The product needs a reversible validation path for those metrics without
turning Dogtap into a UDP metrics daemon or claiming full Datadog Agent parity.

## Decision

Dogtap will keep StatsD UDP collection outside the Dogtap runtime and ship an
adoption recipe that uses the OpenTelemetry Collector Contrib `statsd`
receiver.

The supported recipe:

- receives UDP StatsD/DogStatsD-style metric lines in the Collector
- preserves low-cardinality `service`, `env`, `version`, and route tags
- exports to Dogtap as OTLP HTTP JSON metrics
- is verified by `make smoke-statsd-bridge`

The recipe uses OTLP HTTP JSON because Dogtap can inspect that payload shape in
detail today. It validates metric presence and useful context, not exact
Datadog Agent aggregation semantics.

## Consequences

- Dogtap still does not expose UDP `8125`.
- Existing production Datadog Agent DogStatsD paths can remain unchanged.
- Teams can add local or CI metric contract checks without adding a Dogtap SDK.
- DogStatsD events, service checks, Agent integrations, horizontal receiver
  scaling, and exact Datadog distribution semantics remain out of scope until
  there is fixture-backed support and explicit production safety review.
