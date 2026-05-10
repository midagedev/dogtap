# Datadog API Compatibility

Dogtap exposes a small read-only Datadog API compatibility layer so existing
Datadog-oriented tools, curl snippets, and coding agents can query locally
retained Dogtap telemetry without learning a Dogtap-specific query API.

This is not a full Datadog API implementation. It maps documented Datadog query
paths onto Dogtap's bounded event store for local development, CI, isolated E2E
stacks, and short diagnostic trials.

## Supported Endpoints

| Datadog-compatible path | Dogtap source | Scope |
| --- | --- | --- |
| `POST /api/v2/logs/events/search` | `logs` | Returns retained log events as Datadog-style v2 log search results. |
| `POST /api/v2/rum/events/search` | `rum` | Returns retained RUM and Session Replay metadata as Datadog-style v2 RUM search results. |
| `POST /api/v2/spans/events/search` | `apm`, `otlp` traces | Expands retained trace details into Datadog-style span search results. |
| `GET /api/v1/query` | metric details | Returns retained metric samples as Datadog-style v1 timeseries query results. |

Dogtap accepts Datadog client headers on these paths but does not validate them.
Do not send real Datadog keys to local Dogtap unless your environment already
treats the Dogtap process as trusted.

## Search Request Shape

Logs and RUM support the standard v2 search shape:

```json
{
  "filter": {
    "query": "service:api env:local @trace_id:trace-1"
  },
  "page": {
    "limit": 10
  },
  "sort": "-timestamp"
}
```

Spans also accept the nested v2 shape used by Datadog clients:

```json
{
  "data": {
    "attributes": {
      "filter": {
        "query": "service:api trace_id:trace-1"
      },
      "page": {
        "limit": 10
      },
      "sort": "-timestamp"
    }
  }
}
```

Metric queries support a compact v1 query subset:

```text
GET /api/v1/query?from=0&to=9999999999&query=avg:http.server.request.duration{service:api}
```

## Query Subset

The first compatibility slice intentionally supports the fields agents most
often need while debugging missing telemetry:

- `service`
- `env`
- `version`
- `host`
- `trace_id`, `trace.id`, `dd.trace_id`
- `span_id`, `span.id`, `dd.span_id`
- `session.id`
- `view.id`
- `usr.id`, `user.id`
- `account.id`
- `workspace.id`
- `case.id`
- `route`, `http.route`, `resource_name`
- `source`
- `type`
- `status`
- free-text tokens for log messages and span names/resources

Wildcard suffixes such as `service:api-*` are supported for simple prefix
matching. Boolean expression parsing, facets, indexes, storage tiers, cursor
pagination, quoted phrase matching, permissions, formulas, rollups, and
Datadog's full query language are outside this first slice.

## Examples

Search logs that mention login for a trace:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v2/logs/events/search \
  -H 'Content-Type: application/json' \
  -d '{"filter":{"query":"service:api @trace_id:trace-1 login"},"page":{"limit":5}}'
```

Search Browser RUM by session:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v2/rum/events/search \
  -H 'Content-Type: application/json' \
  -d '{"filter":{"query":"service:web @session.id:session-1 @usr.id:user-1"},"page":{"limit":5}}'
```

Search retained spans by trace:

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v2/spans/events/search \
  -H 'Content-Type: application/json' \
  -d '{"data":{"attributes":{"filter":{"query":"service:api trace_id:trace-1"},"page":{"limit":5}}}}'
```

Query metric samples:

```bash
curl -sS 'http://127.0.0.1:8080/api/v1/query?from=0&to=9999999999&query=avg:http.server.request.duration{service:api}'
```

## Safety Boundary

The compatibility layer is read-only. It does not create monitors, dashboards,
notebooks, incidents, users, API keys, service definitions, or long-term data
storage. It returns only telemetry already retained by Dogtap's configured
bounded memory/file/SQLite store.

This keeps Dogtap useful for agent-driven debugging while preserving the product
boundary: Dogtap is an inspector and validation gateway, not a Datadog clone.
