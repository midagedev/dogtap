# G1 Fixture Evidence

## Status

Passed on 2026-05-08.

G1 is passed for the first product slice with real local evidence from Browser
RUM, Datadog APM tracer, logs HTTP, and OpenTelemetry SDK exports. Captures were
local-only and did not call external Datadog intake APIs.

## Evidence

- RUM: `@datadog/browser-rum` 6.0.0 captured through local
  `/datadog-intake-proxy`; sanitized fixture promoted at
  `fixtures/rum/browser-rum-sdk-batch.json`.
- Logs: JSON, text, and gzip logs captured through local Dogtap by
  `scripts/fixtures/capture-logs.sh`.
- APM: official Datadog Node tracer `dd-trace` 5.44.0 captured msgpack
  `/v0.4/traces`; sanitized evidence promoted under `fixtures/apm/`.
- OTLP: OpenTelemetry Node SDK exported HTTP `/v1/traces` and gRPC traces;
  sanitized evidence promoted under `fixtures/otlp/`.

## Verification

```bash
scripts/fixtures/capture-logs.sh
npm --prefix testdata/rum-browser install
scripts/fixtures/capture-rum-browser.sh
npm --prefix testdata/apm-node install
scripts/fixtures/capture-apm-node-tracer.sh
npm --prefix testdata/otlp-node install
scripts/fixtures/capture-otlp-node-sdk.sh
go run ./cmd/dogtap replay fixtures/rum/browser-rum-sdk-batch.json
go test ./...
```

## Scope Notes

- Java/Spring APM evidence and `dd-apm-test-agent` comparison are deferred by
  `docs/decisions/0005-apm-fixture-scope.md`.
- Run-local raw capture artifacts remain under ignored
  `testdata/g1-evidence/latest/`; only reviewed/sanitized derivatives should be
  promoted into `fixtures/`.
