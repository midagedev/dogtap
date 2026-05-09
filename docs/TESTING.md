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
- Datadog Java tracer
- Datadog logs HTTP intake
- OTLP HTTP
- OTLP gRPC

Each fixture should assert:

- accepted endpoint
- decoded payload shape
- normalized fields
- validation outcome
- redaction outcome

### Replay tests

Replay captured fixtures into Dogtap and compare reports. Replay tests protect against parser drift.

### Integration tests

Run sample apps against Dogtap:

- browser app with Datadog RUM
- Spring app with Datadog Java tracer
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
- Confirm missing service tags fail validation.
- Confirm email and token-like values are redacted.
- Confirm generated Datadog search queries are usable.
