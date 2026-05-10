# OpenTelemetry Filelog Bridge

Dogtap does not tail container stdout, Kubernetes log files, or arbitrary
application log files by itself. If a team currently gets logs in Datadog
because the Datadog Agent tails those files, add a collector bridge when Dogtap
inspection is needed.

This recipe tails JSON log files with the OpenTelemetry Collector Contrib
`filelog` receiver and exports OTLP HTTP logs to Dogtap.

## When To Use This

Use this when:

- an app writes structured JSON logs to stdout or a file
- production log collection is owned by Datadog Agent, Fluent Bit, Vector, or a
  Collector
- you want local, CI, or preview Dogtap inspection without adding a Dogtap SDK

Do not use this as proof that Dogtap replaces the Datadog Agent. The Agent can
keep running in production while this bridge is added only to local or
validation stacks.

## Run The Smoke

From the Dogtap repository:

```bash
make smoke-log-bridge
```

The smoke starts:

- Dogtap
- a tiny log writer that creates two JSON log lines
- an OpenTelemetry Collector that tails `/logs/*.log` and sends OTLP logs to
  `http://dogtap:4318/v1/logs`

It then asserts that Dogtap retained OTLP log events for
`filelog-bridge-backend`.

## Copy Into An App Stack

Copy these files into an application repository or `.dogtap/` folder:

- `otel-filelog-bridge.yaml`
- `compose.otel-filelog-bridge.yaml`

Set the file include pattern and Dogtap endpoint:

```bash
DOGTAP_FILELOG_INCLUDE=/logs/*.log
DOGTAP_OTLP_HTTP_ENDPOINT=http://dogtap:4318
```

The sample Collector exporter uses `encoding: json` because Dogtap can inspect
OTLP HTTP JSON payloads in detail. OTLP gRPC is also decoded by Dogtap. OTLP
HTTP protobuf is accepted, but currently retained as byte metadata rather than
rich log details.

## Expected JSON Fields

The default parser expects JSON log lines similar to:

```json
{"timestamp":"2026-05-10T00:00:00.000Z","level":"info","service":"backend","env":"local","message":"request completed","http.route":"/api/example","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"}
```

The `service`, `env`, `version`, `http.route`, and `http.method` fields stay as
OTLP log attributes so Dogtap can normalize and validate them. The `message`
field is promoted to the OTLP log body for a cleaner log viewer.

If your logs use a different timestamp format or field names, adjust the
Collector operators rather than changing application instrumentation for
Dogtap.

## Rollback

Remove the Collector bridge or stop mounting the bridge Compose file. The app's
normal Datadog Agent or collector path can remain unchanged.
