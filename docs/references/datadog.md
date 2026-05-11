# Datadog References

These are the first references to use while implementing Dogtap compatibility.

## RUM Proxy

Datadog Browser RUM supports proxying browser intake data through a custom endpoint.

- https://docs.datadoghq.com/real_user_monitoring/guide/proxy-rum-data/

Useful details:

- Datadog recommends Browser SDK `4.34.0` or later for proxy
  configurations.
- The Browser SDK `proxy` initialization parameter routes RUM requests through a
  proxy endpoint.
- The SDK includes the original intake path and query in `ddforward`.
- Proxy implementations should preserve the raw body and strip sensitive
  headers.
- Proxy implementations must avoid open-proxy behavior by allowing only the
  expected Datadog intake origins and paths.

## Browser Session Replay

Datadog Session Replay is part of the Browser RUM SDK and uses RUM proxying for
replay uploads.

- https://docs.datadoghq.com/real_user_monitoring/session_replay/browser/

## APM Agent API

Datadog traces are sent to the local Agent through HTTP APIs, commonly on port `8126`.

- https://docs.datadoghq.com/tracing/guide/send_traces_to_agent_by_api/
- https://docs.datadoghq.com/tracing/trace_collection/library_config/java/

Useful details:

- `DD_TRACE_AGENT_URL` can point tracers at an HTTP or Unix socket target.
- For Java tracer configuration, `DD_TRACE_AGENT_URL` takes precedence over
  `DD_AGENT_HOST` and `DD_TRACE_AGENT_PORT`.

## Logs HTTP Intake

Datadog logs support HTTP intake endpoints for JSON, text, gzip, and other formats.

- https://docs.datadoghq.com/api/latest/logs/

## Datadog Search and Query APIs

Dogtap's read-only compatibility layer targets documented Datadog query paths
for local retained telemetry search.

- Logs Events Search: https://docs.datadoghq.com/api/latest/logs/#search-logs
- RUM Events Search: https://docs.datadoghq.com/api/latest/rum/#search-rum-events
- Spans Events Search: https://docs.datadoghq.com/api/latest/spans/#search-spans
- Metrics Query: https://docs.datadoghq.com/api/latest/metrics/#query-timeseries-points
- Logs search syntax: https://docs.datadoghq.com/logs/explorer/search_syntax/

Useful details:

- Logs search syntax treats a quoted sequence such as `"hello world"` as a
  phrase.
- Attribute values with special characters can be queried with quotes, for
  example `@http.url_details.path:"/api/v1/test"`.

## Datadog Agent Log Collection

Datadog Agent log collection is an Agent behavior, not only an HTTP intake
surface.

- https://docs.datadoghq.com/containers/docker/log/
- https://docs.datadoghq.com/containers/kubernetes/log/

## OTLP Intake

Datadog supports direct OpenTelemetry protocol intake.

- https://docs.datadoghq.com/opentelemetry/setup/otlp_ingest/

## Datadog Agent

The Datadog Agent is open source and includes several components.

- https://opensource.datadoghq.com/projects/agent-integrations-tracers/
- https://github.com/DataDog/datadog-agent

## DogStatsD

DogStatsD metrics are typically sent to the Datadog Agent on UDP port `8125`.

- https://docs.datadoghq.com/extend/dogstatsd/

## OpenTelemetry Collector Deployment

The OpenTelemetry Collector supports agent, sidecar, DaemonSet, and gateway
deployment patterns that can bridge telemetry to one or more backends.

- https://opentelemetry.io/docs/collector/configuration/
- https://opentelemetry.io/docs/collector/components/processor/
- https://opentelemetry.io/docs/collector/deploy/agent/
- https://opentelemetry.io/docs/platforms/kubernetes/collector/
- https://docs.datadoghq.com/opentelemetry/setup/collector_exporter/install/

## dd-apm-test-agent

Datadog's APM test agent emulates APM endpoints and includes an optional Web UI.

- https://github.com/DataDog/dd-apm-test-agent
