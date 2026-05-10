# Dogtap Generic Adoption Kit

This folder contains copyable templates for applying Dogtap to a normal
frontend plus backend development stack.

The preferred path for an app that already uses Datadog is external injection:

1. Start Dogtap as a sidecar or standalone process.
2. Override standard Datadog or OTLP endpoints from environment, Compose,
   Kubernetes, or runtime config.
3. Point browser RUM at the Dogtap RUM proxy where the app already exposes the
   RUM proxy setting.
4. Point backend telemetry at Dogtap through OTLP or Datadog tracer settings.
5. Send logs either through OTLP logs, Datadog logs HTTP intake, or a collector
   bridge.
6. Open the Dogtap dashboard and verify that events correlate by service,
   route, trace ID, user, workspace, or case context.

No Dogtap application SDK is required. Removing Dogtap should be a config-only
change that restores the original Datadog or OTLP endpoints.

## Files

| File | Use |
| --- | --- |
| `compose.dogtap.yaml` | Dogtap sidecar for Docker Compose projects |
| `compose.override.template.yaml` | Compose override template that injects Dogtap env into existing services |
| `dogtap.local.yaml` | Local persistent Dogtap config |
| `datadog-preserve.env` | Datadog-preserving env overlay for existing tracers and optional OTLP exporters |
| `backend-otel-http.env` | Backend OTLP HTTP defaults |
| `backend-otel-grpc.env` | Backend OTLP gRPC defaults |
| `backend-datadog-tracer.env` | Existing Datadog tracer defaults |
| `frontend-rum.md` | Browser RUM proxy snippets |
| `frontend-runtime-config.md` | Runtime-config pattern for externally injected RUM proxy values |
| `logs-http.md` | Logs HTTP intake examples |
| `log-forwarder-overrides.md` | Patterns for logs when Datadog Agent or a collector owns log tailing |
| `kubernetes/deployment-sidecar.template.yaml` | Kubernetes same-pod sidecar fragment |
| `otel-collector-tee.yaml` | OpenTelemetry Collector config with Datadog primary and Dogtap secondary exporters |
| `compose.otel-collector-tee.yaml` | Compose wrapper for the Collector tee pattern |
| `otel-collector-tee.md` | Collector tee usage, sampling, and safety notes |
| `otel-filelog-bridge.yaml` | OpenTelemetry Collector filelog receiver config for stdout/file log bridge inspection |
| `compose.otel-filelog-bridge.yaml` | Runnable Dogtap plus Collector filelog bridge smoke stack |
| `otel-filelog-bridge.md` | Filelog bridge usage, field expectations, and rollback notes |
| `otel-statsd-bridge.yaml` | OpenTelemetry Collector StatsD receiver config for DogStatsD-style metric bridge inspection |
| `compose.otel-statsd-bridge.yaml` | Runnable Dogtap plus Collector StatsD bridge smoke stack |
| `otel-statsd-bridge.md` | StatsD bridge usage, metric tag expectations, and rollback notes |

## Compose Use

From an application repository that has this file copied under `.dogtap/`:

```bash
DOGTAP_REPO=../dogtap docker compose -f .dogtap/compose.dogtap.yaml up --build
```

If Dogtap is published as an image in your environment, replace the `build`
section in `compose.dogtap.yaml` with the image name and run without
`DOGTAP_REPO`.

For backend containers in the same Compose project, use `dogtap` as the host.
For host processes outside Docker, use `localhost`.

For existing Compose applications, prefer the override template:

```bash
cp examples/adoption-kit/compose.override.template.yaml ../your-app/.dogtap/compose.override.dogtap.yaml
cd ../your-app
docker compose -f compose.yaml -f .dogtap/compose.override.dogtap.yaml up
```

Rename the placeholder `your-backend` service and merge the env values into each
service you want Dogtap to inspect.

For an executable frontend/backend example of the same pattern, see
`examples/external-injection-smoke/` or run:

```bash
make smoke-external-injection
```

For applications that already send OTLP to an OpenTelemetry Collector, use
`otel-collector-tee.md` to keep Datadog as the primary exporter while sending a
bounded inspection copy to Dogtap.

For applications whose logs only reach Datadog because an Agent or collector
tails stdout/files, use `otel-filelog-bridge.md` or run:

```bash
make smoke-log-bridge
```

For applications that emit DogStatsD or StatsD-style metrics to the Datadog
Agent, use `otel-statsd-bridge.md` or run:

```bash
make smoke-statsd-bridge
```

Before routing Browser RUM or Session Replay through Dogtap outside a local-only
environment, follow `docs/runbooks/RUM_PROXY_CANARY.md` for Browser SDK version,
raw-body, header stripping, `ddforward` allowlist, and rollback checks.
