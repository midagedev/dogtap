# G3 Protocol Intake Evidence

## Status

Passed on 2026-05-08.

## Coverage

- RUM endpoint accepts Browser RUM SDK proxy traffic, including `text/plain`
  JSON batches and request-level `ddforward` / `ddtags` metadata.
- Logs endpoint accepts JSON fixture payloads.
- Logs endpoint accepts text and gzip payloads.
- APM endpoint accepts Datadog tracer msgpack payloads on `/v0.4/traces`.
- OTLP HTTP endpoint accepts OpenTelemetry SDK trace exports.
- OTLP gRPC endpoint accepts OpenTelemetry SDK trace exports.
- gzip decoding is covered by `internal/server` tests.
- Unsupported content types return useful 400 errors.
- Fixture replay is deterministic for bundled smoke and promoted G1 fixtures.

## Verification

```bash
go test ./...
go run ./cmd/dogtap replay fixtures/rum/browser-rum-sdk-batch.json
go run ./cmd/dogtap replay fixtures/rum/login.json fixtures/logs/json-log.json fixtures/apm/trace.json fixtures/otlp/traces.json
```
