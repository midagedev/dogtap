# G5 Datadog API Compatibility

Date: 2026-05-11

## Scope

This gate covers the first read-only Datadog API compatibility slice for
agent-driven local and CI telemetry debugging.

## Evidence

Implemented:

- Logs search compatibility:
  `POST /api/v2/logs/events/search`
- RUM search compatibility:
  `POST /api/v2/rum/events/search`
- Span search compatibility:
  `POST /api/v2/spans/events/search`
- Metric query compatibility:
  `GET /api/v1/query`
- Documentation:
  `docs/DATADOG_API_COMPATIBILITY.md`
- Decision record:
  `docs/decisions/0014-datadog-api-compatibility.md`

Verification:

```bash
go test ./internal/server
go test ./...
```

The server tests cover retained log, RUM, span, and metric sample queries
through Datadog-compatible paths and assert Datadog-shaped response fields.

## Gate Status

Passed for the first read-only compatibility subset.
