# G2 SQLite Storage Evidence

## Scope

This gate covers the first persistent queryable storage slice:

- `storage.kind=sqlite`
- bounded TTL and max-event retention
- indexed metadata columns plus redacted `EventEnvelope` JSON
- compatibility with dashboard, diagnostics, workflow contracts, and
  Datadog-compatible read APIs through the existing store interface

## Evidence

Implemented files:

- `internal/store/sqlite.go`
- `internal/store/sqlite_test.go`
- `internal/config/config.go`
- `internal/server/server.go`

Configuration examples:

- `dogtap.example.yaml`
- `configs/generic-local.yaml`
- `compose.yaml`

Verification commands:

```bash
go test ./internal/store ./internal/config ./internal/server
```

Covered behavior:

- SQLite persistence across reopen
- TTL pruning
- max-event pruning
- store query filtering
- config loading from env
- missing path validation
- server wiring through `storage.kind=sqlite`
- redacted persistence in forward mode
- fail-open storage error behavior when SQLite writes fail in a production-facing
  mode

## Gate Status

Passed for the runtime contract persistent storage subset.

SQLite is not a G7 production warehouse claim. Production-facing usage still
requires bounded sampling, no raw payloads by default, and a reversible
deployment plan.
