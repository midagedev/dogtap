# Adopting Dogtap In A Generic App

This runbook is for a normal local development stack with a browser frontend and
one or more backend services. It avoids Dogtap-specific runtime code; use
Datadog Browser RUM, Datadog tracer, or OpenTelemetry configuration.

If the application already uses Datadog, start with
[External Injection Adoption](EXTERNAL_INJECTION_ADOPTION.md). The preferred
path is to add Dogtap as a sidecar or service and override standard telemetry
endpoints from environment, Compose, Kubernetes, CI, or runtime config.

## Start Dogtap

From the Dogtap repository:

```bash
go run ./cmd/dogtap serve -config configs/generic-local.yaml
```

Or with Docker Compose from the Dogtap repository:

```bash
docker compose up --build
```

From another repository, copy `examples/adoption-kit/compose.dogtap.yaml`
into `.dogtap/` and run:

```bash
DOGTAP_REPO=../dogtap docker compose -f .dogtap/compose.dogtap.yaml up --build
```

Open:

```text
http://localhost:8080
```

## Frontend RUM

Point the existing Datadog Browser RUM SDK at Dogtap through runtime config
where possible:

```bash
DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

Then pass that value to the existing initialization:

```ts
const rumProxy = runtimeConfig.DATADOG_RUM_PROXY_URL;

datadogRum.init({
  applicationId: "local",
  clientToken: "local",
  site: "datadoghq.com",
  service: "your-frontend",
  env: "local",
  version: "local",
  ...(rumProxy ? { proxy: rumProxy } : {}),
});
```

If the app hardcodes RUM initialization, make the proxy externally configurable
once and keep later Dogtap enable/disable operations config-only.

## Backend, Preferred Path: OTLP

For backend containers in the same Compose project:

```bash
OTEL_SERVICE_NAME=your-backend
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=local,service.version=local
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_EXPORTER_OTLP_ENDPOINT=http://dogtap:4318
OTEL_TRACES_EXPORTER=otlp
OTEL_LOGS_EXPORTER=otlp
OTEL_METRICS_EXPORTER=otlp
```

For host processes, use:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

Use `examples/adoption-kit/backend-otel-http.env` or
`examples/adoption-kit/backend-otel-grpc.env` as copyable starting points.

## Backend, Existing Datadog Tracer Path

For backend containers in the same Compose project:

```bash
DD_TRACE_AGENT_URL=http://dogtap:8126
DD_AGENT_HOST=dogtap
DD_TRACE_AGENT_PORT=8126
DD_ENV=local
DD_SERVICE=your-backend
DD_VERSION=local
DD_TRACE_SAMPLE_RATE=1
DD_LOGS_INJECTION=true
```

For host processes, use `DD_AGENT_HOST=localhost`.

## Logs

If logs already flow through OTLP, keep them there. For Datadog logs HTTP:

```bash
curl -sS -X POST http://localhost:8080/api/v2/logs \
  -H 'Content-Type: application/json' \
  -d '{"service":"your-backend","env":"local","version":"local","status":"info","message":"dogtap log smoke"}'
```

Backend containers should send to `http://dogtap:8080/api/v2/logs`.

## Verify

Run the smoke script from the Dogtap repository:

```bash
make smoke-adoption
```

Expected dashboard evidence:

- RUM events appear under `source=rum`.
- Logs appear under `source=logs`.
- Trace spans appear under `source=apm` or `source=otlp`.
- Metrics appear under `source=otlp` with `payloadKind=metric`.
- Validation failures explain missing service tags or required context.

## Remove Dogtap

Removal should be configuration-only:

- Restore the frontend RUM proxy to the normal Datadog path or remove the local
  proxy override.
- Restore backend `OTEL_EXPORTER_OTLP_*` endpoints or `DD_AGENT_HOST` to the
  normal collector or Datadog agent.
- Stop the Dogtap sidecar and remove the local volume if needed.
