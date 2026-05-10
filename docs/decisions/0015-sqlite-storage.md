# 0015: Opt-In SQLite Storage

## Status

Accepted.

## Context

Dogtap started with bounded in-memory retention and optional JSON file snapshots.
That was enough for local payload inspection, but the Datadog-compatible search
API, workflow diagnostics, isolated E2E runs, and dev-cluster trials benefit
from a queryable store that survives process restarts.

Using a network database would make the local and CI path harder to adopt and
would push Dogtap toward an observability backend. The product constitution
requires bounded storage, redaction before persistence, and practical vendor
compatibility rather than broad backend scope.

## Decision

Dogtap will support `storage.kind=sqlite` as an opt-in single-file event store.
The SQLite store persists indexed event metadata plus the redacted
`EventEnvelope` JSON used by existing dashboard, diagnostics, workflow contract,
and Datadog-compatible API paths.

SQLite storage is intended for:

- local runs that should survive restarts
- CI and isolated E2E runs where artifacts are easier to inspect as one file
- dev-cluster deployments where a small persistent volume is useful
- agent debugging through familiar Datadog-compatible search/query endpoints

SQLite storage remains bounded by `storage.maxEvents` and `storage.ttl`.
Selecting SQLite does not enable raw production payload retention and does not
change forwarding behavior.

## Non-Goals

- Network database support
- Multi-replica shared storage
- Long-term telemetry retention
- Full Datadog query/index/facet semantics
- Production observability warehouse behavior

## Consequences

Positive:

- Local and CI telemetry can be inspected after restart.
- Datadog-compatible API queries have indexed metadata to build on.
- Docker Compose and dev-cluster users can keep one bounded `/data/dogtap.db`
  volume instead of relying on JSON rewrite snapshots.

Tradeoffs:

- The binary gains a pure-Go SQLite dependency.
- SQLite file deletion and TTL pruning are not a substitute for compliance
  erasure guarantees; production modes still rely on raw-payload disablement and
  redaction before persistence.
- Multi-pod deployments should keep Dogtap single-replica unless a future
  decision adds shared storage semantics.
