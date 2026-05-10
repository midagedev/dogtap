# Log Forwarder Overrides

Dogtap does not tail container stdout, Kubernetes log files, or arbitrary app
log files by itself. If an application currently relies on the Datadog Agent for
log collection, preserve that production behavior and add a bridge when Dogtap
inspection is needed.

## Preferred Dogtap Inputs

Use one of these paths:

- OTLP logs to `http://dogtap:4318/v1/logs`
- Datadog logs HTTP to `http://dogtap:8080/api/v2/logs`

## Existing Datadog Trace/Log Correlation

Keep trace ID injection enabled:

```bash
DD_LOGS_INJECTION=true
```

This only enriches application logs. A log sender, collector, or direct HTTP
client still has to deliver logs to Dogtap.

## Collector Bridge Pattern

```text
app stdout/file logs
  -> existing collector or log forwarder
  -> Dogtap OTLP logs or Datadog logs HTTP
```

For local and CI adoption, the collector can send only to Dogtap. For staging
experiments, use a tee only if the collector supports bounded retry, sampling,
and fail-open behavior.

An executable OpenTelemetry Collector filelog version of this pattern is
available in:

- `examples/adoption-kit/otel-filelog-bridge.yaml`
- `examples/adoption-kit/compose.otel-filelog-bridge.yaml`
- `examples/adoption-kit/otel-filelog-bridge.md`

Verify it with:

```bash
make smoke-log-bridge
```

## Docker Notes

Datadog's Docker log collection commonly relies on Agent environment variables
such as `DD_LOGS_ENABLED=true` and
`DD_LOGS_CONFIG_CONTAINER_COLLECT_ALL=true`, plus Docker/container log mounts.
Dogtap does not implement that Agent behavior.

## Kubernetes Notes

Datadog's Kubernetes log collection commonly relies on the Agent DaemonSet,
Operator, Helm values, and Autodiscovery annotations. Dogtap does not implement
that DaemonSet behavior.

For Kubernetes local or preview environments, prefer an OpenTelemetry Collector,
Fluent Bit, Vector, or another existing log forwarder that can export a copy as
OTLP logs.

## Definition Of Done

A log adoption profile should prove:

- one successful application log reaches Dogtap
- one error log reaches Dogtap
- logs include `service` and `env`
- logs can correlate with traces through trace or request identifiers
- removing the log bridge restores the original Datadog log path
