# Quickstart

This quickstart describes the generic local adoption path. It should work for a
typical browser frontend plus backend service without adding a Dogtap-specific
SDK.

## Start Dogtap

From this repository:

```bash
docker compose up --build
```

Open:

```text
http://localhost:8080
```

For source development with the generic local profile:

```bash
go run ./cmd/dogtap serve -config configs/generic-local.yaml
```

## Browser RUM Target

Configure Datadog Browser RUM to use Dogtap as a proxy.

```ts
datadogRum.init({
  applicationId: "local",
  clientToken: "local",
  site: "datadoghq.com",
  service: "your-frontend",
  env: "local",
  version: "local",
  proxy: "http://localhost:8080/datadog-intake-proxy",
});
```

## Backend OTLP Target

For host processes:

```bash
export OTEL_SERVICE_NAME=your-backend
export OTEL_RESOURCE_ATTRIBUTES=deployment.environment=local,service.version=local
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_TRACES_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
```

For Docker Compose containers in the same project, use `http://dogtap:4318`.

## Existing Datadog Tracer Target

For host processes:

```bash
export DD_AGENT_HOST=localhost
export DD_TRACE_AGENT_PORT=8126
export DD_ENV=local
export DD_SERVICE=your-backend
export DD_VERSION=local
export DD_TRACE_SAMPLE_RATE=1
export DD_LOGS_INJECTION=true
```

For Docker Compose containers in the same project, use `DD_AGENT_HOST=dogtap`.

## Logs HTTP Target

```bash
curl -sS -X POST http://localhost:8080/api/v2/logs \
  -H 'Content-Type: application/json' \
  -d '{"service":"your-backend","env":"local","version":"local","status":"info","message":"dogtap log smoke"}'
```

## Smoke Verification

```bash
make smoke-adoption
```

Expected evidence:

- RUM payloads appear under `source=rum`.
- Logs appear under `source=logs`.
- Datadog tracer payloads appear under `source=apm`.
- OTLP traces/logs/metrics appear under `source=otlp`.
- The dashboard shows service map, traffic, trace spans, logs, replay payloads,
  metric samples, validation results, and copyable Datadog search hints.

## CI Mode

Fixture replay is available today:

```bash
go run ./cmd/dogtap replay \
  -config configs/generic-local.yaml \
  -output dogtap-report.md \
  -format markdown \
  fixtures/rum/login.json \
  fixtures/logs/json-log.json \
  fixtures/apm/trace.json \
  fixtures/otlp/traces.json
```

Expected exit codes:

- `0`: no blocking validation failures
- `1`: validation failed
- `2`: configuration error
- `3`: intake startup error
- `4`: replay/report tool error
