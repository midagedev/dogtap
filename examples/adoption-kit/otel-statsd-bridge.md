# OpenTelemetry StatsD Bridge

Dogtap does not bind UDP `8125` and does not replace Datadog Agent DogStatsD.
If a service emits DogStatsD or StatsD-style metrics, keep the production Agent
or collector path and add a bridge only where Dogtap inspection is needed.

This recipe receives UDP StatsD metrics with the OpenTelemetry Collector
Contrib `statsd` receiver and exports OTLP HTTP metrics to Dogtap.

## When To Use This

Use this when:

- an app already emits StatsD or DogStatsD-style metrics
- you need local, CI, or preview validation that those metrics include useful
  service, env, route, or workflow tags
- you do not want Dogtap itself to own UDP metric collection

Do not use this to claim full Datadog Agent DogStatsD parity. Treat it as a
bridge for observable metric contract tests.

## Run The Smoke

From the Dogtap repository:

```bash
make smoke-statsd-bridge
```

The smoke starts:

- Dogtap
- an OpenTelemetry Collector listening on UDP `8125`
- a tiny metrics sender that emits a counter and gauge with DogStatsD-style
  tags

It then asserts that Dogtap retained OTLP metric events for
`statsd-bridge-backend` and observed `dogtap.bridge.request.count`.

## Copy Into An App Stack

Copy these files into an application repository or `.dogtap/` folder:

- `otel-statsd-bridge.yaml`
- `compose.otel-statsd-bridge.yaml`

Point the app's StatsD host and port at the Collector, not Dogtap:

```bash
DD_DOGSTATSD_HOST=otel-collector
DD_DOGSTATSD_PORT=8125
```

The Collector should export Dogtap's OTLP HTTP base endpoint:

```bash
DOGTAP_OTLP_HTTP_ENDPOINT=http://dogtap:4318
```

The sample exporter uses `encoding: json` so Dogtap can inspect OTLP HTTP
metrics in detail.

The OpenTelemetry Collector StatsD receiver is intended for agent-style
deployment. Do not run several independent bridge collectors behind the same UDP
traffic stream and expect exact global aggregation.

## Expected Metric Shape

DogStatsD-style metric lines should include low-cardinality tags that Dogtap
can normalize:

```text
dogtap.bridge.request.count:1|c|#service:backend,env:local,version:dev,http.route:/api/example
```

Avoid high-cardinality tags such as user IDs, emails, raw URLs, request IDs, or
trace IDs on metrics. Keep those in logs, spans, or RUM context where they can
be sampled and validated more safely.

This bridge covers StatsD/DogStatsD-style metrics. It does not cover DogStatsD
events, service checks, Datadog Agent integrations, or exact Datadog
distribution semantics.

## Rollback

Remove the Collector bridge or restore the app's StatsD host to the Datadog
Agent. Dogtap does not need to be removed from application code because this
recipe uses standard StatsD and OTLP surfaces.
