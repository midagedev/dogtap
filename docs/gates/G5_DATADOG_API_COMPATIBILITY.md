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

Maintenance slice, 2026-05-11:

- Structured log fields are now retained in the redacted detail model for
  route, method, status, service/env/version, trace/span IDs, context IDs,
  request ID, and correlation ID.
- Logs search matches common field aliases such as `@http.status_code`,
  `@http.method`, `@endpoint`, `@payload_kind`, `@validation.status`,
  `@dogtap.id`, `@request_id`, and `@correlation_id`.
- Metric query scope matching uses retained redacted point tags for service,
  env, version, route, method, and HTTP status aliases.
- Metric series include additive `dogtap_event_ids` so agents can jump from a
  Datadog-compatible query result back to retained Dogtap events.
- Trace search understands exact IDs, leading-zero hex forms, and decimal IDs
  that match the low 64 bits of a 128-bit trace ID.

Hardening slice, 2026-05-11:

- Logs, RUM, and span search tokenization preserves simple quoted phrases and
  quoted attribute values, including path-like route values with slashes.
- Metric scope matching accepts quoted tag values such as
  `http.route:"/api/v1/orders"`.
- Public docs clarify that these quoted forms are supported while advanced
  boolean expression parsing remains outside Dogtap's compatibility subset.

## Gate Status

Passed for the first read-only compatibility subset and the structured
debugging maintenance slice.
