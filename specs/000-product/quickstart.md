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

Configure Datadog Browser RUM to use Dogtap as a proxy through runtime config
where possible.

```bash
DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

Then pass the value to the existing Datadog RUM init object:

```ts
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
export DD_TRACE_AGENT_URL=http://localhost:8126
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
  metric samples, browser session timeline, validation results, workflow
  contract status, and copyable Datadog search hints.

## CI Mode

Fixture replay validates static payloads:

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

Live diagnostics validates a running Dogtap instance after an app or E2E suite
has sent telemetry:

```bash
go run ./cmd/dogtap diagnose \
  -base-url http://127.0.0.1:8080 \
  -output .dogtap/diagnostics/local \
  -expect-non-empty \
  -expect-source rum,logs,apm,otlp \
  -expect-payload-kind replay,metric
```

Workflow contracts assert named user paths:

```bash
go run ./cmd/dogtap diagnose \
  -base-url http://127.0.0.1:8080 \
  -output .dogtap/diagnostics/login \
  -workflow-contract configs/contracts/login.yaml \
  -fail-on-workflow-contract \
  -expect-non-empty
```

The same diagnostics can be collected by API when Dogtap is running inside
Docker Compose:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/diagnostics/archive \
  -H 'Content-Type: application/json' \
  -d '{"useDefaultWorkflowContracts":true,"expect":{"nonEmpty":true}}' \
  -o dogtap-diagnostics.zip
```

Expected exit codes:

- `0`: no blocking validation failures
- `1`: validation or explicit workflow contract assertion failed
- `2`: configuration error
- `3`: intake startup error
- `4`: replay/report tool error
