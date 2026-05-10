# Testing Strategy

## Test Goals

Dogtap must prove two things:

1. It can receive and decode realistic telemetry.
2. It can identify whether the telemetry is useful and safe.

## Test Layers

### Unit tests

Cover deterministic logic:

- config parsing
- redaction rules
- PII and token detection
- required field validation
- normalizer mappings
- query-string stripping
- bounded queue behavior

### Contract tests

Use fixture payloads from real SDKs and tracers:

- Datadog Browser RUM
- Datadog tracer fixture-backed payloads, currently from Node `dd-trace`
- Datadog logs HTTP intake
- OTLP HTTP
- OTLP gRPC

Java/Spring tracer evidence and `dd-apm-test-agent` comparison are deferred by
ADR 0005 and should be added only with fixture-backed evidence.

Each fixture should assert:

- accepted endpoint
- decoded payload shape
- normalized fields
- validation outcome
- redaction outcome

### Replay tests

Replay captured fixtures into Dogtap and compare reports. Replay tests protect against parser drift.

### Live diagnostics

Use the diagnostics API when a local dev server, isolated E2E stack, Docker
Compose environment, or external app adoption run is already sending telemetry
to a running Dogtap instance:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/diagnostics \
  -H 'Content-Type: application/json' \
  -d '{"expect":{"nonEmpty":true,"sources":["rum","logs","apm","otlp"],"payloadKinds":["replay","metric"]}}'
```

Download the same evidence as an archive:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/diagnostics/archive \
  -H 'Content-Type: application/json' \
  -d '{"expect":{"nonEmpty":true}}' \
  -o dogtap-diagnostics.zip
```

Use `dogtap diagnose` when a host-side artifact directory is more convenient:

```bash
go run ./cmd/dogtap diagnose \
  -base-url http://127.0.0.1:8080 \
  -output .dogtap/diagnostics/local-dev \
  -workflow-contract configs/contracts/login.yaml \
  -expect-source rum,logs,apm,otlp \
  -expect-payload-kind replay,metric
```

The API archive and CLI command write `summary.md`, `assertions.json`,
`events.json`, `report.json`, `debug-bundle.json`, and `metrics.txt` so humans
and agents can triage missing telemetry without scraping console output.
When diagnostics expectations fail, `assertions.json` includes `rootCauses`
with evidence and next checks for common missing browser, log, trace, metric,
OTLP exporter, endpoint routing, context, and Dogtap API failures.
When workflow contracts are requested they also write
`workflow-contracts.json`, which is the easiest file for an agent to inspect
when a real path such as login emitted incomplete telemetry.

### Spec and docs alignment

Dogtap is spec-driven, so CI also checks that high-signal documentation markers
stay aligned with implemented features:

```bash
make doc-check
```

This check is intentionally narrow. It verifies that the Spec Kit baseline is
marked as the current release-candidate/active implementation baseline and that
the data model and docs include implemented surfaces such as Faro, metrics,
diagnostics snapshots, workflow contracts, and workflow contract CI examples.

Workflow contract templates are validated separately:

```bash
make contract-check
```

### Integration tests

Run sample apps against Dogtap:

- browser app with Datadog RUM
- backend app with a Datadog tracer fixture; Java/Spring integration is
  deferred until it has captured evidence
- log sender
- OpenTelemetry sample

### Dashboard tests

Use browser-driven tests once UI exists:

- payload appears in stream
- filters work
- validation failure opens details
- debug bundle can be generated
- raw payload is hidden in production mode

The live demo visual check starts Dogtap, seeds representative public telemetry,
and verifies the real dashboard against the real HTTP API:

```bash
make demo-visual-check
```

It stores desktop and mobile screenshots under `web/test-results/` for visual
review and CI artifacts.

### Production safety tests

Use fault injection:

- Datadog forward target unavailable
- slow forward target
- queue full
- invalid gzip
- large payload
- malicious header values
- malformed JSON

Expected outcomes must be explicit for each mode.

## CI Exit Codes

- `0`: validation passed
- `1`: validation failed
- `2`: invalid Dogtap configuration
- `3`: intake server failed to start
- `4`: fixture replay failed due to tool error

## Reference Validation Profile

Initial workflows:

- login
- logout
- signup
- workspace switch
- subscription status
- payment action
- design case creation
- viewer open
- export request

Required fields by workflow should live in a separate validation config.

## Manual Test Checklist

- Start Dogtap locally.
- Send a RUM event.
- Send a trace.
- Send a log.
- Confirm all appear in dashboard.
- Run `make demo-seed` to populate replay, logs, spans, metrics, service map,
  traffic, and validation failure examples.
- Query `POST /api/diagnostics` with `expect.nonEmpty=true`.
- Run `go run ./cmd/dogtap diagnose -expect-non-empty` and inspect
  `.dogtap/diagnostics/*/summary.md` when a directory artifact is needed.
- Confirm missing service tags fail validation.
- Confirm email and token-like values are redacted.
- Confirm generated Datadog search queries are usable.
